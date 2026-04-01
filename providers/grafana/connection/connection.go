// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/logger/zerologadapter"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

// GrafanaConnection holds the HTTP client and auth state for a Grafana instance.
type GrafanaConnection struct {
	plugin.Connection
	Conf    *inventory.Config
	asset   *inventory.Asset
	client  *http.Client
	baseURL string
	token   string
}

// NewGrafanaConnection constructs a GrafanaConnection, resolving credentials and
// base URL from vault credentials, conf options, or environment variables.
// Both token and baseURL are required; an error is returned if either is absent.
func NewGrafanaConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*GrafanaConnection, error) {
	conn := &GrafanaConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	// Resolve token: vault credential (CredentialType_password) takes precedence over env.
	token := os.Getenv("GRAFANA_TOKEN")
	for _, cred := range conf.Credentials {
		if cred.Type == vault.CredentialType_password {
			token = string(cred.Secret)
		} else {
			log.Warn().Str("credential-type", cred.Type.String()).Msg("unsupported credential type for Grafana provider")
		}
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("a valid Grafana token is required, pass --token '<yourtoken>' or set GRAFANA_TOKEN environment variable")
	}

	// Resolve base URL: conf option takes precedence over env.
	baseURL := os.Getenv("GRAFANA_URL")
	if conf.Options != nil {
		if v, ok := conf.Options["url"]; ok && v != "" {
			baseURL = v
		}
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		return nil, errors.New("a Grafana instance URL is required, pass --url '<url>' or set GRAFANA_URL environment variable")
	}

	// Build a retryablehttp client that handles transient failures automatically.
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 5
	retryClient.Logger = zerologadapter.New(log.Logger)
	httpClient := retryClient.StandardClient()
	httpClient.Timeout = 30 * time.Second

	conn.token = token
	conn.baseURL = baseURL
	conn.client = httpClient

	return conn, nil
}

// Name returns the connection type name.
func (c *GrafanaConnection) Name() string {
	return "grafana"
}

// Asset returns the inventory asset associated with this connection.
func (c *GrafanaConnection) Asset() *inventory.Asset {
	return c.asset
}

// BaseURL returns the trimmed base URL of the Grafana instance.
func (c *GrafanaConnection) BaseURL() string {
	return c.baseURL
}

// OrgID returns the org-id option value, or empty string if not set.
func (c *GrafanaConnection) OrgID() string {
	if c.Conf.Options == nil {
		return ""
	}
	return c.Conf.Options["org-id"]
}

// Get issues an authenticated GET request to baseURL+path and returns the raw
// response. The caller is responsible for closing the response body and checking
// the status code.
func (c *GrafanaConnection) Get(ctx context.Context, path string) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("grafana: failed to create request for %s: %w", path, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if orgID := c.OrgID(); orgID != "" {
		req.Header.Set("X-Grafana-Org-Id", orgID)
	}
	return c.client.Do(req)
}

// grafanaOrgPlatform returns the canonical platform descriptor for a Grafana org.
func grafanaOrgPlatform() *inventory.Platform {
	return &inventory.Platform{
		Name:                  "grafana-org",
		Title:                 "Grafana Organization",
		Family:                []string{"grafana"},
		Kind:                  "api",
		Runtime:               "grafana",
		TechnologyUrlSegments: []string{"saas", "grafana", "org"},
	}
}

// PlatformInfo returns the platform descriptor for this connection.
func (c *GrafanaConnection) PlatformInfo() (*inventory.Platform, error) {
	return grafanaOrgPlatform(), nil
}

// Identifier returns the platform MRN for this Grafana org connection.
// If org-id is set in options it is appended; otherwise the path ends at "org".
func (c *GrafanaConnection) Identifier() string {
	base := "//platformid.api.mondoo.app/runtime/grafana/org"
	orgID := c.OrgID()
	if orgID != "" {
		return base + "/" + orgID
	}
	return base
}
