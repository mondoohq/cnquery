// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ApiToken represents an Okta API token. Okta's golang SDK does not (yet) expose
// the /api/v1/api-tokens endpoint, so we model the response shape ourselves.
//
// Reference: https://developer.okta.com/docs/reference/api/api-token/
type ApiToken struct {
	Id          string     `json:"id,omitempty"`
	Name        string     `json:"name,omitempty"`
	UserId      string     `json:"userId,omitempty"`
	ClientName  string     `json:"clientName,omitempty"`
	Created     *time.Time `json:"created,omitempty"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	LastUpdated *time.Time `json:"lastUpdated,omitempty"`
	TokenWindow string     `json:"tokenWindow,omitempty"`
}

// ListApiTokens fetches all API tokens for the org. The endpoint requires Super Admin
// privileges. We use raw HTTP because the Okta golang SDK v2 does not include this resource.
//
// Pagination follows Okta's `Link: <url>; rel="next"` response header convention until
// no `next` link is returned.
func ListApiTokens(ctx context.Context, host, token string) ([]*ApiToken, error) {
	client := http.Client{}
	result := []*ApiToken{}
	nextURL := fmt.Sprintf("https://%s/api/v1/api-tokens?limit=200", host)

	for nextURL != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, nextURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("SSWS %s", token))

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to fetch API tokens from %s: %s", nextURL, resp.Status)
		}

		raw, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		page := []*ApiToken{}
		if err := json.Unmarshal(raw, &page); err != nil {
			return nil, err
		}
		result = append(result, page...)

		nextURL = nextLinkURL(resp.Header.Values("Link"))
	}

	return result, nil
}

// nextLinkURL parses RFC 5988 `Link` headers and returns the URL whose rel is `next`,
// or an empty string when no such link is present. Okta returns one Link entry per
// header value (e.g., `<https://...>; rel="next"`).
func nextLinkURL(headers []string) string {
	for _, h := range headers {
		parts := strings.Split(h, ";")
		if len(parts) < 2 {
			continue
		}
		rel := ""
		for _, p := range parts[1:] {
			if strings.Contains(p, `rel="next"`) {
				rel = "next"
				break
			}
		}
		if rel != "next" {
			continue
		}
		raw := strings.TrimSpace(parts[0])
		raw = strings.TrimPrefix(raw, "<")
		raw = strings.TrimSuffix(raw, ">")
		return raw
	}
	return ""
}
