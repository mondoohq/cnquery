// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// APIError captures a non-2xx response from the Netlify API.
type APIError struct {
	StatusCode int
	Message    string
	Path       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("netlify API %s: %d (%s)", e.Path, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("netlify API %s: %d", e.Path, e.StatusCode)
}

// IsForbidden reports whether err is a 401/403 access-denied response. Callers
// degrade to null rather than failing the whole scan, since plan-gated
// endpoints answer 403 on the free tier and tokens can be issued with a
// narrower scope than the resource tree covers.
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

// Get performs a GET against the Netlify API and decodes the JSON body into out.
func (c *NetlifyConnection) Get(ctx context.Context, path string, query url.Values, out any) error {
	body, err := c.do(ctx, path, query)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("netlify API %s: decode response: %w", path, err)
	}
	return nil
}

func (c *NetlifyConnection) do(ctx context.Context, path string, query url.Values) ([]byte, error) {
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

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("netlify API %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("netlify API %s: read response: %w", path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode, Path: path}
		var envelope struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		if json.Unmarshal(body, &envelope) == nil {
			apiErr.Message = envelope.Message
			if apiErr.Message == "" {
				apiErr.Message = envelope.Error
			}
		}
		return nil, apiErr
	}

	return body, nil
}

// pageSize is the number of records requested per page. Netlify caps per_page
// at 100 on the endpoints that paginate.
const pageSize = 100

// maxPages bounds a paginated walk. An endpoint that ignores the page
// parameter answers every request with the first page, which would otherwise
// loop forever; the duplicate-page guard in GetPaged catches the common shape
// of that bug and this cap catches the rest.
const maxPages = 200

// GetPaged walks Netlify's page-based pagination for a list endpoint and
// returns every record. Endpoints that ignore paging return their single
// response unchanged.
func GetPaged[T any](ctx context.Context, c *NetlifyConnection, path string, query url.Values) ([]T, error) {
	if query == nil {
		query = url.Values{}
	}
	query.Set("per_page", strconv.Itoa(pageSize))

	var results []T
	var previous []byte
	for page := 1; page <= maxPages; page++ {
		query.Set("page", strconv.Itoa(page))

		body, err := c.do(ctx, path, query)
		if err != nil {
			return nil, err
		}

		// An endpoint that ignores the page parameter answers every request
		// with the first page. Detect that by the response repeating verbatim,
		// and drop it instead of appending the same records a second time.
		if previous != nil && bytes.Equal(previous, body) {
			break
		}
		previous = body

		var batch []T
		if err := json.Unmarshal(body, &batch); err != nil {
			return nil, fmt.Errorf("netlify API %s: decode response: %w", path, err)
		}

		results = append(results, batch...)

		// A short page is the last page.
		if len(batch) < pageSize {
			break
		}
	}

	return results, nil
}

// GetList fetches a list endpoint that does not paginate.
func GetList[T any](ctx context.Context, c *NetlifyConnection, path string, query url.Values) ([]T, error) {
	var out []T
	if err := c.Get(ctx, path, query, &out); err != nil {
		return nil, err
	}
	return out, nil
}
