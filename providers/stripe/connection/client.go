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
	"time"
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

// IsClientError reports whether err is any 4xx response. Callers use it to
// degrade an endpoint that is simply unavailable for the current account
// (for example external accounts on a standalone account) to an empty result
// rather than failing the whole scan.
func IsClientError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode >= 400 && apiErr.StatusCode < 500
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

const (
	// maxRateLimitRetries bounds how often a throttled request is retried
	// before the 429 surfaces as an error. Stripe throttles reads at 100
	// requests a second in live mode.
	maxRateLimitRetries = 3
	// baseRetryDelay is the first backoff step used when a 429 arrives
	// without a usable Retry-After header. It doubles per attempt.
	baseRetryDelay = 500 * time.Millisecond
	// maxRetryDelay caps a single wait so an oversized Retry-After cannot
	// park a scan.
	maxRetryDelay = 30 * time.Second
)

func (c *StripeConnection) do(ctx context.Context, path string, query url.Values) ([]byte, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	for attempt := 0; ; attempt++ {
		body, status, header, err := c.roundTrip(ctx, u)
		if err != nil {
			return nil, fmt.Errorf("stripe API %s: %w", path, err)
		}

		if status >= 200 && status < 300 {
			return body, nil
		}

		apiErr := newAPIError(status, path, body)
		if status != http.StatusTooManyRequests || attempt >= maxRateLimitRetries {
			return nil, apiErr
		}
		if err := sleepCtx(ctx, retryAfterDelay(header, attempt)); err != nil {
			return nil, err
		}
	}
}

// roundTrip issues one authenticated GET and returns the body alongside the
// status and headers, so the caller can decide whether to retry.
func (c *StripeConnection) roundTrip(ctx context.Context, u string) ([]byte, int, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Stripe-Version", apiVersion)
	if c.account != "" {
		req.Header.Set("Stripe-Account", c.account)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("read response: %w", err)
	}
	return body, resp.StatusCode, resp.Header, nil
}

// newAPIError builds an APIError from a non-2xx response, pulling the Stripe
// error envelope out of the body when it is present.
func newAPIError(status int, path string, body []byte) *APIError {
	apiErr := &APIError{StatusCode: status, Path: path}
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
	return apiErr
}

// retryAfterDelay reads the cooldown Stripe reports on a 429, accepting both
// the delay-seconds and HTTP-date forms of Retry-After, and falls back to an
// exponential backoff when the header is missing or unparseable.
func retryAfterDelay(header http.Header, attempt int) time.Duration {
	if v := header.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			return clampDelay(time.Duration(secs) * time.Second)
		}
		if when, err := http.ParseTime(v); err == nil {
			return clampDelay(time.Until(when))
		}
	}
	return clampDelay(baseRetryDelay << attempt)
}

func clampDelay(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	if d > maxRetryDelay {
		return maxRetryDelay
	}
	return d
}

// sleepCtx waits for d, giving up early if the context is cancelled.
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
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
