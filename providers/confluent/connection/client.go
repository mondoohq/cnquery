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
	"strings"
)

// APIError captures a non-2xx response from a Confluent endpoint.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Path       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("confluent API %s: %d %s (%s)", e.Path, e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("confluent API %s: %d %s", e.Path, e.StatusCode, e.Code)
}

// IsForbidden reports whether err is a 401/403 access-denied response.
//
// It matches on the status the server returned, never on the error text, so a
// transport failure is never mistaken for a definitive answer about what the
// caller may read.
func IsForbidden(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden
	}
	return false
}

// IsNotFound reports whether err is a 404 response. Only that answer means the
// object is absent; a 403 says the caller may not look, which is not the same
// thing and must surface as an error.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// StatusCode returns the HTTP status carried by err, or 0 when err did not come
// from a server response.
func StatusCode(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}
	return 0
}

// RequestTarget names one endpoint family and the credentials that open it.
// The management API and a cluster's own REST endpoint take different keys, so
// every request carries the pair it belongs to rather than reading a single
// connection-wide credential.
type RequestTarget struct {
	// BaseURL is the scheme and host requests are issued against, without a
	// trailing slash.
	BaseURL string
	// Key and Secret are the HTTP basic credentials for this endpoint.
	Key    string
	Secret string
	// Label names the endpoint in error messages.
	Label string
	// SupportsPageSize reports whether the endpoint family accepts a page_size
	// parameter. The management API documents one; the per-cluster REST
	// endpoints do not, and sending it there would add an unrecognized
	// parameter to every request.
	SupportsPageSize bool
}

// CloudTarget is the management API at api.confluent.cloud, opened by the Cloud
// API key.
func (c *ConfluentConnection) CloudTarget() RequestTarget {
	return RequestTarget{
		BaseURL: c.baseURL,
		Key:     c.apiKey,
		Secret:  c.apiSecret,
		Label:   "cloud",

		SupportsPageSize: true,
	}
}

// KafkaTarget is one cluster's REST endpoint, opened by a cluster-scoped Kafka
// API key. The Cloud API key is not accepted there, so a missing Kafka key is
// reported as an error rather than producing an empty topic or ACL list, which
// would read as a cluster with no ACLs at all.
func (c *ConfluentConnection) KafkaTarget(clusterID, restEndpoint string) (RequestTarget, error) {
	if restEndpoint == "" {
		return RequestTarget{}, fmt.Errorf("kafka cluster %s reports no REST endpoint", clusterID)
	}
	key, secret := c.KafkaCredentialsFor(clusterID)
	if key == "" || secret == "" {
		suffix := KafkaEnvSuffix(clusterID)
		hint := EnvKafkaAPIKey + " and " + EnvKafkaAPISecret
		if suffix != "" {
			hint += ", or " + EnvKafkaAPIKey + "_" + suffix + " and " + EnvKafkaAPISecret + "_" + suffix
		}
		return RequestTarget{}, fmt.Errorf(
			"a cluster-scoped Kafka API key is required to read %s (set %s, or pass --kafka-api-key and --kafka-api-secret)",
			clusterID, hint)
	}
	return RequestTarget{
		BaseURL: strings.TrimRight(restEndpoint, "/"),
		Key:     key,
		Secret:  secret,
		Label:   "kafka " + clusterID,
	}, nil
}

// Get performs a GET against one endpoint and decodes the JSON body into out.
func (c *ConfluentConnection) Get(ctx context.Context, target RequestTarget, path string, query url.Values, out any) error {
	u := target.BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	body, err := c.do(ctx, target, u, path)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("confluent API %s: decode response: %w", path, err)
	}
	return nil
}

func (c *ConfluentConnection) do(ctx context.Context, target RequestTarget, rawURL, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(target.Key, target.Secret)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("confluent API %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("confluent API %s: read response: %w", path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newAPIError(resp.StatusCode, path, body)
	}
	return body, nil
}

// newAPIError decodes the two error envelopes Confluent uses. The management
// API answers with a JSON:API style `errors` array; the per-cluster Kafka REST
// endpoints answer with a flat `error_code`/`message` pair.
func newAPIError(status int, path string, body []byte) *APIError {
	apiErr := &APIError{StatusCode: status, Path: path}

	var cloud struct {
		Errors []struct {
			Code   string `json:"code"`
			Title  string `json:"title"`
			Detail string `json:"detail"`
		} `json:"errors"`
	}
	if json.Unmarshal(body, &cloud) == nil && len(cloud.Errors) > 0 {
		first := cloud.Errors[0]
		apiErr.Code = first.Code
		apiErr.Message = first.Detail
		if apiErr.Message == "" {
			apiErr.Message = first.Title
		}
		return apiErr
	}

	var kafka struct {
		ErrorCode *int   `json:"error_code"`
		Message   string `json:"message"`
	}
	if json.Unmarshal(body, &kafka) == nil && kafka.ErrorCode != nil {
		apiErr.Code = strconv.Itoa(*kafka.ErrorCode)
		apiErr.Message = kafka.Message
	}
	return apiErr
}

// pageSize is the number of records requested per page. It is the documented
// maximum for the management API list endpoints; a smaller default would
// multiply the number of round trips for no benefit.
const pageSize = 100

// maxPages bounds a paginated walk so an endpoint that answers with a
// stationary cursor cannot loop forever.
const maxPages = 500

// listEnvelope is the shape every Confluent list endpoint shares: a data array
// and a metadata block whose `next` is an absolute URL to the following page,
// or absent on the last page.
type listEnvelope[T any] struct {
	Data     []T `json:"data"`
	Metadata struct {
		Next *string `json:"next"`
	} `json:"metadata"`
}

// GetPaged walks a Confluent list endpoint, collecting every element of the
// data array. Both the management API and the per-cluster Kafka REST endpoints
// page by handing back an absolute URL for the next page.
//
// The walk stops when the next URL is absent, repeats the URL just fetched, or
// leaves the host the walk started on. An endpoint that echoes its own URL back
// would otherwise replay the same page until the page cap, multiplying every
// record it holds; an endpoint that hands back a foreign host would send
// credentials somewhere they do not belong.
func GetPaged[T any](ctx context.Context, c *ConfluentConnection, target RequestTarget, path string, query url.Values) ([]T, error) {
	if query == nil {
		query = url.Values{}
	}
	if target.SupportsPageSize && query.Get("page_size") == "" {
		query.Set("page_size", strconv.Itoa(pageSize))
	}

	current := target.BaseURL + path
	if encoded := query.Encode(); encoded != "" {
		current += "?" + encoded
	}

	origin, err := hostOf(current)
	if err != nil {
		return nil, fmt.Errorf("confluent API %s: %w", path, err)
	}

	var results []T
	seen := map[string]bool{}

	for page := 0; page < maxPages; page++ {
		if seen[current] {
			break
		}
		seen[current] = true

		body, err := c.do(ctx, target, current, path)
		if err != nil {
			return nil, err
		}

		var envelope listEnvelope[T]
		if err := json.Unmarshal(body, &envelope); err != nil {
			return nil, fmt.Errorf("confluent API %s: decode response: %w", path, err)
		}
		results = append(results, envelope.Data...)

		next := ""
		if envelope.Metadata.Next != nil {
			next = strings.TrimSpace(*envelope.Metadata.Next)
		}
		if next == "" {
			break
		}

		nextHost, err := hostOf(next)
		if err != nil {
			return nil, fmt.Errorf("confluent API %s: pagination cursor %q: %w", path, next, err)
		}
		if nextHost != origin {
			return nil, fmt.Errorf("confluent API %s: pagination cursor points at %s, expected %s", path, nextHost, origin)
		}

		current = next
	}

	return results, nil
}

// hostOf renders the scheme and authority of a URL, which is what a pagination
// cursor has to keep constant for the walk to stay on the endpoint it started
// on.
func hostOf(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("%q is not an absolute URL", rawURL)
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}
