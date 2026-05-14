// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package client

// Tenant captures the tenant-level metadata exposed by GET /v1/settings.
// Not every Iru deployment populates every field; treat each as best-effort.
type Tenant struct {
	Subdomain           string `json:"subdomain"`
	OrganizationName    string `json:"organization_name"`
	Region              string `json:"region"`
	AgentMinimumVersion string `json:"agent_minimum_version"`
	AgentLatestVersion  string `json:"agent_latest_version"`
}

// GetTenant fetches /v1/settings. Some tenants return 404 for accounts
// without the settings permission; callers should tolerate IsAccessDenied.
func (c *Client) GetTenant() (*Tenant, error) {
	var out Tenant
	if err := c.do("/api/v1/settings", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
