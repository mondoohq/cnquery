// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package mistralai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	DefaultBaseURL   = "https://api.mistral.ai"
	maxErrorBodySize = 64 << 10
	defaultPageSize  = 100
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

type ClientOption func(*Client)

func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

func NewClient(token string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL: DefaultBaseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) request(ctx context.Context, method, endpoint string, result any) error {
	u, err := url.Parse(c.baseURL + endpoint)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySize))
		var apiErr APIError
		if err := json.Unmarshal(bodyBytes, &apiErr); err != nil {
			return &APIError{
				Message:    fmt.Sprintf("request failed (status: %d): %s", resp.StatusCode, string(bodyBytes)),
				StatusCode: resp.StatusCode,
			}
		}
		apiErr.StatusCode = resp.StatusCode
		return &apiErr
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) ListModels(ctx context.Context) (*ModelList, error) {
	var models ModelList
	if err := c.request(ctx, http.MethodGet, "/v1/models", &models); err != nil {
		return nil, err
	}
	return &models, nil
}

func (c *Client) GetModel(ctx context.Context, modelID string) (*Model, error) {
	var model Model
	if err := c.request(ctx, http.MethodGet, "/v1/models/"+url.PathEscape(modelID), &model); err != nil {
		return nil, err
	}
	return &model, nil
}

func (c *Client) ListFineTuningJobs(ctx context.Context) ([]FineTuningJob, error) {
	var all []FineTuningJob
	page := 0
	for {
		var resp FineTuningJobList
		endpoint := fmt.Sprintf("/v1/fine_tuning/jobs?page=%d&page_size=%d", page, defaultPageSize)
		if err := c.request(ctx, http.MethodGet, endpoint, &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.Data...)
		if len(all) >= resp.Total || len(resp.Data) == 0 {
			break
		}
		page++
	}
	return all, nil
}

func (c *Client) ListFiles(ctx context.Context) ([]File, error) {
	var all []File
	page := 0
	for {
		var resp FileList
		endpoint := "/v1/files?page=" + strconv.Itoa(page) + "&page_size=" + strconv.Itoa(defaultPageSize)
		if err := c.request(ctx, http.MethodGet, endpoint, &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.Data...)
		if resp.Total == nil || len(all) >= *resp.Total || len(resp.Data) == 0 {
			break
		}
		page++
	}
	return all, nil
}

func (c *Client) ListBatchJobs(ctx context.Context) ([]BatchJob, error) {
	var all []BatchJob
	page := 0
	for {
		var resp BatchJobList
		endpoint := fmt.Sprintf("/v1/batch/jobs?page=%d&page_size=%d", page, defaultPageSize)
		if err := c.request(ctx, http.MethodGet, endpoint, &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.Data...)
		if len(all) >= resp.Total || len(resp.Data) == 0 {
			break
		}
		page++
	}
	return all, nil
}

func IsAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	if apiErr, ok := errors.AsType[*APIError](err); ok {
		return apiErr.StatusCode == 401 || apiErr.StatusCode == 403
	}
	return false
}
