// Copyright (c) 2026 Rich Haase. All rights reserved.
// Use of this source code is governed by the MIT license.

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

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
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func FromConfig(cfg *config.Config) *Client {
	return New(cfg.API.BaseURL, cfg.API.Token)
}

func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", "c2cli/0.1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s: %w", url, err)
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort close on read-only body

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d) from %s", resp.StatusCode, path)
	}

	return body, nil
}

func (c *Client) GetUser(ctx context.Context) (*models.UserProfile, error) {
	body, err := c.get(ctx, "/api/users/me")
	if err != nil {
		return nil, err
	}
	var resp models.UserResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse user profile: %w", err)
	}
	return &resp.Data, nil
}

func (c *Client) GetResults(ctx context.Context, from, to string, page int) (*models.ResultsResponse, error) {
	path := fmt.Sprintf("/api/users/me/results?type=rower&page=%d", page)
	if from != "" {
		path += "&from=" + from
	}
	if to != "" {
		path += "&to=" + to
	}

	body, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}

	var resp models.ResultsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse results: %w", err)
	}
	return &resp, nil
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

func (c *Client) GetStrokes(ctx context.Context, workoutID int64) ([]models.StrokeData, error) {
	path := fmt.Sprintf("/api/users/me/results/%d/strokes", workoutID)
	body, err := c.get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("fetch strokes for workout %d: %w", workoutID, err)
	}

	var strokes []models.StrokeData
	if err := json.Unmarshal(body, &strokes); err != nil {
		return nil, fmt.Errorf("parse strokes for workout %d: %w", workoutID, err)
	}
	return strokes, nil
}
