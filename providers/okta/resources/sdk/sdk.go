// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ApiExtension handles cases where Okta's SDK doesn't expose a particular API.
// The v5 SDK no longer ships a public RequestExecutor, so we issue the raw
// authenticated requests ourselves against the org host carried by the
// connection.
type ApiExtension struct {
	// Host is the org host (e.g. "dev-12345.okta.com"), without scheme.
	Host string
	// Authorize stamps the Authorization header onto each outgoing request.
	// The connection supplies it so this path authenticates the same way the
	// generated SDK does, whether that is an SSWS API token or a service app's
	// private key JWT. A nil Authorize sends the request unauthenticated,
	// which Okta answers with a 401.
	Authorize func(req *http.Request) error
	// HTTPClient issues the requests. When nil, http.DefaultClient is used.
	// Tests inject a client with a custom transport so pagination can be
	// exercised without mutating any global state.
	HTTPClient *http.Client
}

// httpClient returns the configured client, falling back to the shared default.
func (m *ApiExtension) httpClient() *http.Client {
	if m.HTTPClient != nil {
		return m.HTTPClient
	}
	return http.DefaultClient
}

// get issues an authenticated GET against an absolute Okta URL and decodes the
// JSON body into out (when non-nil). It returns the raw http.Response so callers
// can inspect the status code and Link headers for pagination. On a >= 400
// status it returns the response together with an error so callers can still
// branch on resp.StatusCode (e.g. to treat 404 as an empty result).
func (m *ApiExtension) get(ctx context.Context, url string, out any) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if m.Authorize != nil {
		if err := m.Authorize(req); err != nil {
			return nil, err
		}
	}

	resp, err := m.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return resp, fmt.Errorf("okta API request to %s failed: %s: %s", url, resp.Status, string(raw))
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp, err
		}
	}
	return resp, nil
}

// getPaged walks every page of a collection endpoint, following the `Link:
// rel="next"` header until Okta stops offering one, and returns the
// concatenated items. The returned http.Response is the first page's, so
// callers can branch on its status code to treat a 404 as an empty result.
//
// It is a free function rather than a method because Go does not allow type
// parameters on methods.
func getPaged[T any](ctx context.Context, m *ApiExtension, path string) ([]T, *http.Response, error) {
	items := []T{}
	nextURL := m.url(path)
	var firstResp *http.Response

	for nextURL != "" {
		var page []T
		resp, err := m.get(ctx, nextURL, &page)
		if firstResp == nil {
			firstResp = resp
		}
		if err != nil {
			return nil, resp, err
		}
		items = append(items, page...)
		if resp == nil {
			break
		}
		nextURL = nextLinkURL(resp.Header.Values("Link"))
	}

	return items, firstResp, nil
}

// url builds an absolute org URL for the given API path (e.g. "/api/v1/zones").
func (m *ApiExtension) url(path string) string {
	return fmt.Sprintf("https://%s%s", m.Host, path)
}

// urlWithParams builds an absolute org URL with a query string, omitting the
// "?" entirely when there are no parameters so the URL does not end in a bare
// separator.
func (m *ApiExtension) urlWithParams(path string, params url.Values) string {
	u := m.url(path)
	if len(params) == 0 {
		return u
	}
	return u + "?" + params.Encode()
}
