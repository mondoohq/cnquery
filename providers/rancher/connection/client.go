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
	"strings"
	"sync"
)

// maxPages caps a pagination walk. Rancher defaults to 1000 records per page,
// so the cap is far above any real fleet and exists only to bound a server that
// keeps handing out a next link.
const maxPages = 200

// defaultPageLimit is the page size requested from the server. Rancher's own
// default is 1000; asking explicitly keeps the walk predictable across
// versions that changed it.
const defaultPageLimit = 1000

// Client talks to the Rancher Manager API over HTTP. It is deliberately a thin
// JSON client rather than Rancher's generated one.
//
// The generated client was read before this was written. It is missing three
// things this needs, each of which turns a failure into a quiet wrong answer:
//
//   - Its ListAll shadows the error variable in the pagination loop's init
//     statement, so the `if err != nil` after the loop reads the outer err,
//     which is always nil by then. A page-two fetch that fails ends the walk
//     and returns a truncated list with no error at all. (v2.12.9,
//     pkg/client/generated/management/v3/zz_generated_token.go:118-134, and
//     the same shape in every other generated ListAll.)
//   - Its Next follows pagination.next with the Authorization header attached,
//     without checking that the link names the configured server and without a
//     cap on how many links it will follow. This one refuses a cross-host link
//     and stops on a repeated one.
//   - Its IsNotFound uses a bare type assertion rather than errors.As, so any
//     wrapping defeats it, and it has no IsForbidden at all. Reading a 403 as
//     "not found" is what would turn an unreadable endpoint into an absent
//     feature. (github.com/rancher/norman@v0.9.7 clientbase/common.go:86-93.)
//
// Two of its habits are also wrong for a provider that handles tokens:
// clientbase.DoGet puts the entire response body into the error text when the
// body does not parse (ops.go:95), and an init() turns full request and
// response logging on from RANCHER_CLIENT_DEBUG (common.go:403-407) writing to
// stdout, which is the go-plugin handshake channel.
//
// The collection envelope below is the same shape as norman's types.Collection
// and types.Pagination, field for field on the parts that are read.
type Client struct {
	baseURL string
	token   string
	http    *http.Client

	cacheMu sync.Mutex
	cache   map[string]*cacheEntry
}

// NewClient builds a client for one Rancher Manager. The base URL is the server
// root without a trailing slash, for example https://rancher.example.com.
func NewClient(baseURL, token string, httpClient *http.Client) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    httpClient,
		cache:   map[string]*cacheEntry{},
	}
}

// collection is the envelope every Rancher v3 list endpoint returns. The
// records arrive under data, and pagination.next carries an absolute URL to the
// following page rather than an opaque cursor.
type collection struct {
	Data       []json.RawMessage `json:"data"`
	Pagination *pagination       `json:"pagination"`
}

type pagination struct {
	Next    string `json:"next"`
	Limit   *int64 `json:"limit"`
	Total   *int64 `json:"total"`
	Partial bool   `json:"partial"`
}

// APIError is the answer the server gave. It carries the HTTP status so that
// callers classify on what the server said rather than on the text of a
// message, which a transport failure would also produce.
type APIError struct {
	StatusCode int
	Status     string
	Message    string
	Path       string
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = e.Status
	}
	return fmt.Sprintf("rancher api %s: %d %s", e.Path, e.StatusCode, msg)
}

// IsNotFound reports whether the server answered that the endpoint or object
// does not exist. Only that answer may be turned into an absent feature.
//
// A transport failure is deliberately excluded: it is not an answer from the
// server at all, and treating a network blip as "absent" would turn an
// unreachable Rancher into a clean audit pass. So is a 403, which says the
// token may not read the endpoint and tells us nothing about what is behind it.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusNotFound
}

// IsForbidden reports whether the token was refused access to the endpoint.
func IsForbidden(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusForbidden || apiErr.StatusCode == http.StatusUnauthorized
}

// Get fetches a single object and decodes it into out.
func (c *Client) Get(ctx context.Context, path string, out any) error {
	body, err := c.do(ctx, c.absolute(path))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

// List walks every page of a collection endpoint and returns the raw records.
//
// Pagination follows the absolute URL the server puts in pagination.next. A
// server that echoes a link it already gave would otherwise loop forever and
// repeat every record, so a link that was already fetched ends the walk with an
// error: a listing that cannot be trusted must not be reported as a short one.
func (c *Client) List(ctx context.Context, path string) ([]json.RawMessage, error) {
	next := c.absolute(withLimit(path, defaultPageLimit))

	seen := map[string]struct{}{}
	records := []json.RawMessage{}

	for pages := 0; next != ""; pages++ {
		if pages >= maxPages {
			return nil, fmt.Errorf("rancher api %s: pagination did not finish within %d pages", path, maxPages)
		}
		if _, repeated := seen[next]; repeated {
			return nil, fmt.Errorf("rancher api %s: pagination repeated the page at %s", path, next)
		}
		seen[next] = struct{}{}

		body, err := c.do(ctx, next)
		if err != nil {
			return nil, err
		}

		var page collection
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("rancher api %s: %w", path, err)
		}
		records = append(records, page.Data...)

		if page.Pagination == nil || page.Pagination.Next == "" {
			break
		}
		following, err := c.sameServer(page.Pagination.Next)
		if err != nil {
			return nil, fmt.Errorf("rancher api %s: %w", path, err)
		}
		next = following
	}

	return records, nil
}

// do issues one authenticated request and returns its body.
func (c *Client) do(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Message:    errorMessage(body),
			Path:       requestPath(rawURL),
		}
	}
	return body, nil
}

// errorMessage pulls the human-readable half out of a Rancher error body,
// falling back to nothing when the body is not the shape we expect. The raw
// body is not used as the message: an HTML error page from an intermediate
// proxy would otherwise land in the error text.
func errorMessage(body []byte) string {
	var payload struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if payload.Message != "" {
		return payload.Message
	}
	return payload.Code
}

// absolute turns an API path into a full URL against the configured server.
func (c *Client) absolute(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return c.baseURL + "/" + strings.TrimLeft(path, "/")
}

// sameServer checks that a link the server handed back points at the same
// server before it is followed. The request carries the API token, so a next
// link naming another host would send the credential somewhere the operator
// never configured.
func (c *Client) sameServer(rawURL string) (string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	next, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if !next.IsAbs() {
		return c.absolute(rawURL), nil
	}
	if !strings.EqualFold(next.Host, base.Host) || !strings.EqualFold(next.Scheme, base.Scheme) {
		return "", fmt.Errorf("pagination link points at %s://%s, not the configured server", next.Scheme, next.Host)
	}
	return next.String(), nil
}

// withLimit adds an explicit page size to a collection path.
func withLimit(path string, limit int) string {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return fmt.Sprintf("%s%slimit=%d", path, separator, limit)
}

// requestPath reduces a URL to its path, so an error names the endpoint without
// repeating the host and query on every message.
func requestPath(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return parsed.Path
}

// ListCached walks a collection endpoint once per connection and hands the same
// records to every later caller.
//
// The listings are small and read-only for the life of a scan, and several
// resources reach for the same one: a cluster's projects, a project's bindings
// and a template's clusters all read a list that another field has usually
// fetched already. Without the cache each of those would be one more call per
// record.
func (c *Client) ListCached(ctx context.Context, path string) ([]json.RawMessage, error) {
	c.cacheMu.Lock()
	entry, ok := c.cache[path]
	if !ok {
		entry = &cacheEntry{}
		if c.cache == nil {
			c.cache = map[string]*cacheEntry{}
		}
		c.cache[path] = entry
	}
	c.cacheMu.Unlock()

	entry.once.Do(func() {
		entry.records, entry.err = c.List(ctx, path)
	})
	return entry.records, entry.err
}

type cacheEntry struct {
	once    sync.Once
	records []json.RawMessage
	err     error
}
