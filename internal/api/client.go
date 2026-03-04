package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/richhaase/c2cli/internal/config"
	"github.com/richhaase/c2cli/internal/models"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		baseURL:    baseURL,
		token:      token,
		httpClient: &http.Client{},
	}
}

func FromConfig(cfg *config.Config) *Client {
	return New(cfg.API.BaseURL, cfg.API.Token)
}

func (c *Client) get(path string) ([]byte, error) {
	url := c.baseURL + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (c *Client) GetUser() (*models.UserProfile, error) {
	body, err := c.get("/api/users/me")
	if err != nil {
		return nil, err
	}
	var resp models.UserResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse user profile: %w", err)
	}
	return &resp.Data, nil
}

func (c *Client) GetResults(from, to string, page int) (*models.ResultsResponse, error) {
	path := fmt.Sprintf("/api/users/me/results?type=rower&page=%d", page)
	if from != "" {
		path += "&from=" + from
	}
	if to != "" {
		path += "&to=" + to
	}

	body, err := c.get(path)
	if err != nil {
		return nil, err
	}

	var resp models.ResultsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse results: %w", err)
	}
	return &resp, nil
}

func (c *Client) GetAllResults(from, to string) ([]models.Workout, error) {
	var all []models.Workout
	page := 1

	for {
		resp, err := c.GetResults(from, to, page)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Data...)

		hasMore := resp.Meta != nil &&
			resp.Meta.Pagination != nil &&
			resp.Meta.Pagination.CurrentPage < resp.Meta.Pagination.TotalPages

		if !hasMore || len(resp.Data) == 0 {
			break
		}
		page++
	}

	return all, nil
}

func (c *Client) GetStrokes(workoutID int64) ([]models.StrokeData, error) {
	path := fmt.Sprintf("/api/users/me/results/%d/strokes", workoutID)
	body, err := c.get(path)
	if err != nil {
		return nil, nil // not all workouts have stroke data
	}

	var strokes []models.StrokeData
	if err := json.Unmarshal(body, &strokes); err != nil {
		return nil, fmt.Errorf("failed to parse strokes: %w", err)
	}
	return strokes, nil
}
