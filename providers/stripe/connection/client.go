// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// APIError captures a non-2xx response from the Stripe API.
type APIError struct {
	StatusCode int
	Type       string
	Code       string
	Message    string
	Path       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("stripe API %s: %d %s (%s)", e.Path, e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("stripe API %s: %d %s", e.Path, e.StatusCode, e.Code)
}

// IsForbidden reports whether err is a 401/403 access-denied response. Callers
// degrade to null rather than failing the whole scan, since a restricted key
// may lack read access to some resources.
func IsForbidden(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden
	}
	return false
}

// IsNotFound reports whether err is a 404 response.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// Get performs a GET against the Stripe API and decodes the JSON body into out.
func (c *StripeConnection) Get(ctx context.Context, path string, query url.Values, out any) error {
	body, err := c.do(ctx, path, query)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("stripe API %s: decode response: %w", path, err)
	}
	return nil
}

func (c *StripeConnection) do(ctx context.Context, path string, query url.Values) ([]byte, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Stripe-Version", apiVersion)
	if c.account != "" {
		req.Header.Set("Stripe-Account", c.account)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stripe API %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("stripe API %s: read response: %w", path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode, Path: path}
		var envelope struct {
			Error struct {
				Type    string `json:"type"`
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &envelope) == nil {
			apiErr.Type = envelope.Error.Type
			apiErr.Code = envelope.Error.Code
			apiErr.Message = envelope.Error.Message
		}
		return nil, apiErr
	}

	return body, nil
}

// identifiable is satisfied by every Stripe list element, which always carries
// a stable id. List uses it to derive the starting_after cursor for the next
// page.
type identifiable interface {
	GetID() string
}

// pageMaxLimit is the maximum page size Stripe accepts for list endpoints.
const pageMaxLimit = 100

// List follows Stripe's cursor pagination for a list endpoint, collecting every
// element into a single slice. Stripe returns {"object":"list","has_more":bool,
// "data":[...]} and expects the id of the last element as starting_after for the
// next page. The element type must report its id via GetID so the cursor can be
// advanced.
func List[T identifiable](ctx context.Context, c *StripeConnection, path string, query url.Values) ([]T, error) {
	if query == nil {
		query = url.Values{}
	}
	query.Set("limit", strconv.Itoa(pageMaxLimit))

	var results []T
	for {
		body, err := c.do(ctx, path, query)
		if err != nil {
			return nil, err
		}

		var page struct {
			Data    []T  `json:"data"`
			HasMore bool `json:"has_more"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("stripe API %s: decode response: %w", path, err)
		}

		results = append(results, page.Data...)

		if !page.HasMore || len(page.Data) == 0 {
			break
		}
		last := page.Data[len(page.Data)-1]
		cursor := last.GetID()
		if cursor == "" {
			// Without a cursor we cannot advance safely; stop rather than
			// replaying the first page indefinitely.
			break
		}
		query.Set("starting_after", cursor)
	}

	return results, nil
}
