// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ApiExtension handles cases where Okta's SDK doesn't expose a particular API.
// The v5 SDK no longer ships a public RequestExecutor, so we issue the raw
// authenticated requests ourselves using the org host and SSWS token carried by
// the connection.
type ApiExtension struct {
	// Host is the org host (e.g. "dev-12345.okta.com"), without scheme.
	Host string
	// Token is the Okta API token used for SSWS authorization.
	Token string
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
	req.Header.Set("Authorization", "SSWS "+m.Token)

	resp, err := http.DefaultClient.Do(req)
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

// url builds an absolute org URL for the given API path (e.g. "/api/v1/zones").
func (m *ApiExtension) url(path string) string {
	return fmt.Sprintf("https://%s%s", m.Host, path)
}
