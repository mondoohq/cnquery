// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sdk

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// PolicyWrapper models the fields we expose from Okta's /api/v1/policies
// endpoint. We decode the policy ourselves (rather than via the generated SDK)
// because the v5 policy listing is a discriminated union whose `settings` block
// varies per policy type; capturing `conditions` and `settings` as raw JSON
// preserves the full shape for the dict fields.
type PolicyWrapper struct {
	Id          string          `json:"id,omitempty"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Priority    int64           `json:"priority,omitempty"`
	Status      string          `json:"status,omitempty"`
	System      *bool           `json:"system,omitempty"`
	Type        string          `json:"type,omitempty"`
	Created     *time.Time      `json:"created,omitempty"`
	LastUpdated *time.Time      `json:"lastUpdated,omitempty"`
	Conditions  json.RawMessage `json:"conditions,omitempty"`
	Settings    json.RawMessage `json:"settings,omitempty"`
}

// ListPolicies retrieves all policies of the given type. It returns the raw
// http.Response so callers can branch on the status code (e.g. treat 404 or an
// "invalid policy type" 400 as an empty result).
func (m *ApiExtension) ListPolicies(ctx context.Context, policyType string, limit int) ([]*PolicyWrapper, *http.Response, error) {
	url := m.url("/api/v1/policies")
	url += "?type=" + policyType
	if limit > 0 {
		url += "&limit=" + strconv.Itoa(limit)
	}

	var policies []*PolicyWrapper
	resp, err := m.get(ctx, url, &policies)
	if err != nil {
		return nil, resp, err
	}
	return policies, resp, nil
}
