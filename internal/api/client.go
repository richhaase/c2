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
	"github.com/richhaase/c2/internal/models"
)

const (
	requestTimeout = 30 * time.Second
	acceptType     = "application/vnd.c2logbook.v1+json"
)

type doer interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	baseURL   *url.URL
	token     string
	userAgent string
	http      doer
}

func New(baseURL, token, version string) (*Client, error) {
	return newClient(baseURL, token, version, &http.Client{Timeout: requestTimeout})
}

func newClient(baseURL, token, version string, httpClient doer) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("Invalid Concept2 API URL: %w", err)
	}
	if parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("Concept2 API URL must be an HTTPS origin.")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return &Client{
		baseURL:   parsed,
		token:     token,
		userAgent: "c2/" + version,
		http:      httpClient,
	}, nil
}

func FromConfig(cfg config.Config, version string) (*Client, error) {
	return New(cfg.API.BaseURL, cfg.API.Token, version)
}

func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	endpoint := c.baseURL.ResolveReference(&url.URL{Path: path, RawQuery: query.Encode()})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", acceptType)

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
	if err := c.get(ctx, "/api/users/me", nil, &resp); err != nil {
		return models.UserProfile{}, err
	}
	return resp.Data, nil
}

type ResultsFilter struct {
	From         string
	To           string
	UpdatedAfter string
}

func (c *Client) GetResults(ctx context.Context, filter ResultsFilter, page int) (models.ResultsResponse, error) {
	params := url.Values{}
	params.Set("type", "rower")
	params.Set("page", strconv.Itoa(page))
	if filter.From != "" {
		params.Set("from", filter.From)
	}
	if filter.To != "" {
		params.Set("to", filter.To)
	}
	if filter.UpdatedAfter != "" {
		params.Set("updated_after", filter.UpdatedAfter)
	}
	var resp models.ResultsResponse
	err := c.get(ctx, "/api/users/me/results", params, &resp)
	return resp, err
}

func (c *Client) GetAllResults(ctx context.Context, filter ResultsFilter) ([]models.Workout, error) {
	var all []models.Workout
	page := 1
	for {
		resp, err := c.GetResults(ctx, filter, page)
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
	if err := c.get(ctx, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}
