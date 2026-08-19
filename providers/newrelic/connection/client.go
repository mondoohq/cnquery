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
	"strings"
)

// Client issues NerdGraph queries against one New Relic region.
type Client struct {
	httpClient *http.Client
	endpoint   string
	apiKey     string
}

// NewClient builds a NerdGraph client. The endpoint is a full GraphQL URL, so
// the region is resolved before the client is constructed rather than inside
// it.
func NewClient(httpClient *http.Client, endpoint, apiKey string) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{httpClient: httpClient, endpoint: endpoint, apiKey: apiKey}
}

// Endpoint is the GraphQL URL the client talks to.
func (c *Client) Endpoint() string { return c.endpoint }

// graphQLRequest is the wire shape of a NerdGraph request.
type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// graphQLResponse is the wire shape of a NerdGraph response. NerdGraph answers
// 200 even when the query failed, putting the failure in errors, so both halves
// have to be read.
type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []GraphQLError  `json:"errors"`
}

// GraphQLError is one entry of a NerdGraph errors array.
type GraphQLError struct {
	Message    string                 `json:"message"`
	Path       []any                  `json:"path"`
	Extensions GraphQLErrorExtensions `json:"extensions"`
}

// GraphQLErrorExtensions carries the machine-readable classification of a
// GraphQL error. The errorClass is what the classifiers match on: the message
// is prose that New Relic is free to reword.
type GraphQLErrorExtensions struct {
	ErrorClass string `json:"errorClass"`
	Code       string `json:"code"`
}

// QueryError reports that NerdGraph answered with a GraphQL errors array.
type QueryError struct {
	Errors []GraphQLError
}

func (e *QueryError) Error() string {
	msgs := make([]string, 0, len(e.Errors))
	for _, err := range e.Errors {
		if err.Extensions.ErrorClass != "" {
			msgs = append(msgs, err.Extensions.ErrorClass+": "+err.Message)
			continue
		}
		msgs = append(msgs, err.Message)
	}
	if len(msgs) == 0 {
		return "the New Relic API reported an unspecified error"
	}
	return "the New Relic API reported: " + strings.Join(msgs, "; ")
}

// Classes lists the error classes NerdGraph attached to the failure.
func (e *QueryError) Classes() []string {
	out := make([]string, 0, len(e.Errors))
	for _, err := range e.Errors {
		out = append(out, err.Extensions.ErrorClass)
	}
	return out
}

// HTTPStatusError reports that the API answered with a non-200 status. The
// status is kept so classification never has to read the body text.
type HTTPStatusError struct {
	StatusCode int
	Body       string
}

func (e *HTTPStatusError) Error() string {
	body := strings.TrimSpace(e.Body)
	if len(body) > 512 {
		body = body[:512]
	}
	if body == "" {
		return fmt.Sprintf("the New Relic API answered with HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("the New Relic API answered with HTTP %d: %s", e.StatusCode, body)
}

// IsForbidden reports whether the API refused the request because the supplied
// key may not perform it.
//
// It matches only on definitive answers the server gave: the HTTP status, or
// the errorClass NerdGraph attached to a GraphQL error. A transport failure is
// deliberately not a match, because a dropped connection tells us nothing about
// what the key may read, and turning it into "not permitted" would let a
// network blip degrade a field to null and pass an audit on data nobody read.
func IsForbidden(err error) bool {
	if err == nil {
		return false
	}

	var statusErr *HTTPStatusError
	if errors.As(err, &statusErr) {
		return statusErr.StatusCode == http.StatusUnauthorized || statusErr.StatusCode == http.StatusForbidden
	}

	var queryErr *QueryError
	if errors.As(err, &queryErr) {
		for _, e := range queryErr.Errors {
			switch e.Extensions.ErrorClass {
			case "UNAUTHORIZED", "FORBIDDEN":
				return true
			}
		}
	}
	return false
}

// IsNotFound reports whether the API answered that the thing asked for does not
// exist. Like IsForbidden it matches only on the status or the errorClass, so a
// transport failure is never read as an absence.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	var statusErr *HTTPStatusError
	if errors.As(err, &statusErr) {
		return statusErr.StatusCode == http.StatusNotFound
	}

	var queryErr *QueryError
	if errors.As(err, &queryErr) {
		for _, e := range queryErr.Errors {
			if e.Extensions.ErrorClass == "NOT_FOUND" {
				return true
			}
		}
	}
	return false
}

// Query runs one NerdGraph query and decodes the data half of the response into
// out.
//
// A GraphQL errors array fails the call even when data came back alongside it.
// NerdGraph routinely answers with both, and taking the partial data would
// report a truncated list as a complete one.
func (c *Client) Query(ctx context.Context, query string, vars map[string]any, out any) error {
	body, err := json.Marshal(graphQLRequest{Query: query, Variables: vars})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return &HTTPStatusError{StatusCode: resp.StatusCode, Body: string(payload)}
	}

	var parsed graphQLResponse
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return fmt.Errorf("could not decode the New Relic API response: %w", err)
	}
	if len(parsed.Errors) > 0 {
		return &QueryError{Errors: parsed.Errors}
	}
	if len(parsed.Data) == 0 {
		return errors.New("the New Relic API returned no data and no error")
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(parsed.Data, out)
}
