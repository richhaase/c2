package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/models"
)

const requestTimeout = 30 * time.Second

type Client struct {
	baseURL   string
	token     string
	userAgent string
	http      *http.Client
}

func New(baseURL, token, version string) *Client {
	return &Client{
		baseURL:   baseURL,
		token:     token,
		userAgent: "c2/" + version,
		http:      &http.Client{Timeout: requestTimeout},
	}
}

func FromConfig(cfg config.Config, version string) *Client {
	return New(cfg.API.BaseURL, cfg.API.Token, version)
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("API error (%d) from %s", resp.StatusCode, path)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) GetUser(ctx context.Context) (models.UserProfile, error) {
	var resp models.UserResponse
	if err := c.get(ctx, "/api/users/me", &resp); err != nil {
		return models.UserProfile{}, err
	}
	return resp.Data, nil
}

func (c *Client) GetResults(ctx context.Context, from, to string, page int) (models.ResultsResponse, error) {
	params := url.Values{}
	params.Set("type", "rower")
	params.Set("page", strconv.Itoa(page))
	if from != "" {
		params.Set("from", from)
	}
	if to != "" {
		params.Set("to", to)
	}
	var resp models.ResultsResponse
	err := c.get(ctx, "/api/users/me/results?"+params.Encode(), &resp)
	return resp, err
}

func (c *Client) GetAllResults(ctx context.Context, from, to string) ([]models.Workout, error) {
	var all []models.Workout
	page := 1
	for {
		resp, err := c.GetResults(ctx, from, to, page)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Data...)

		hasMore := resp.Meta != nil && resp.Meta.Pagination != nil &&
			resp.Meta.Pagination.CurrentPage < resp.Meta.Pagination.TotalPages
		if !hasMore || len(resp.Data) == 0 {
			break
		}
		page++
	}
	return all, nil
}

func (c *Client) GetStrokes(ctx context.Context, workoutID int64) ([]models.StrokeData, error) {
	var resp models.StrokeDataResponse
	path := fmt.Sprintf("/api/users/me/results/%d/strokes", workoutID)
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}
