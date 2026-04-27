package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/model"
)

const defaultTimeout = 30 * time.Second

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
	UserAgent  string
}

func FromConfig(cfg config.Config, version string) *Client {
	return &Client{
		BaseURL:    cfg.API.BaseURL,
		Token:      cfg.API.Token,
		HTTPClient: &http.Client{Timeout: defaultTimeout},
		UserAgent:  "c2/" + version,
	}
}

func (c *Client) GetUser(ctx context.Context) (model.UserProfile, error) {
	var resp model.UserResponse
	if err := c.get(ctx, "/api/users/me", &resp); err != nil {
		return model.UserProfile{}, err
	}
	return resp.Data, nil
}

func (c *Client) GetResults(ctx context.Context, from string, to string, page int) (model.ResultsResponse, error) {
	params := url.Values{}
	params.Set("type", "rower")
	params.Set("page", strconv.Itoa(page))
	if from != "" {
		params.Set("from", from)
	}
	if to != "" {
		params.Set("to", to)
	}

	var resp model.ResultsResponse
	path := "/api/users/me/results?" + params.Encode()
	if err := c.get(ctx, path, &resp); err != nil {
		return model.ResultsResponse{}, err
	}
	return resp, nil
}

func (c *Client) GetAllResults(ctx context.Context, from string, to string) ([]model.Workout, error) {
	var all []model.Workout
	for page := 1; ; page++ {
		resp, err := c.GetResults(ctx, from, to, page)
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
	}
	return all, nil
}

func (c *Client) GetStrokes(ctx context.Context, workoutID int) ([]model.StrokeData, error) {
	var resp model.StrokeDataResponse
	path := fmt.Sprintf("/api/users/me/results/%d/strokes", workoutID)
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error (%d) from %s", resp.StatusCode, pathOnly(path))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func pathOnly(path string) string {
	if i := strings.IndexByte(path, '?'); i >= 0 {
		return path[:i]
	}
	return path
}
