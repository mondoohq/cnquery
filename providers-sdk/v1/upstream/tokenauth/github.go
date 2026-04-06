// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tokenauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// GitHubTokenProvider fetches an identity token from GitHub Actions.
type GitHubTokenProvider struct{}

func (p *GitHubTokenProvider) GetToken(ctx context.Context, audience string) (string, error) {
	tokenRequestToken := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	tokenRequestURL := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL")

	if tokenRequestToken == "" || tokenRequestURL == "" {
		return "", fmt.Errorf("github Actions environment variables not set")
	}

	tokenURL := fmt.Sprintf("%s&audience=%s", tokenRequestURL, audience)
	req, err := http.NewRequestWithContext(ctx, "GET", tokenURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", "bearer "+tokenRequestToken)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github Actions token service returned non-OK status: %d", resp.StatusCode)
	}

	var result struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Value, nil
}
