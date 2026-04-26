// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
func ListApiTokens(ctx context.Context, host, token string) ([]*ApiToken, error) {
	urlPath := fmt.Sprintf("https://%s/api/v1/api-tokens?limit=200", host)
	client := http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlPath, nil)
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch API tokens from %s: %s", urlPath, resp.Status)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := []*ApiToken{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result, nil
}
