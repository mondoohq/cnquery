// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package client

import "encoding/json"

// LibraryItem is one entry in the Iru library catalog. Library items are
// heterogeneous (App Store apps, custom apps, custom profiles, profiles,
// scripts, MDM restrictions, FileVault, Gatekeeper, …); the `Kind` field
// discriminates and `Payload` carries the kind-specific config as raw JSON.
type LibraryItem struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Kind         string   `json:"type"`
	Active       bool     `json:"active"`
	BlueprintIDs []string `json:"blueprints"`
	Created      string   `json:"created"`
	UpdatedAt    string   `json:"updated_at"`
	// Counts is the per-status assignment summary (installed / pending /
	// failed / etc.) the API returns alongside the item.
	Counts map[string]int `json:"counts"`
	// Payload is the kind-specific config. Shape varies — for scripts it
	// holds `script`/`run_on`/`run_frequency`; for app-store-app it holds
	// `app_id`/`bundle_id`/`version`; for custom-profile it holds the
	// profile metadata. Exposed as a generic map for MQL dict consumption.
	Payload map[string]any `json:"payload"`
}

// ListLibraryItems walks /v1/library/library-items.
func (c *Client) ListLibraryItems() ([]LibraryItem, error) {
	var all []LibraryItem
	err := c.paginate("/api/v1/library/library-items", func(raw json.RawMessage) (int, error) {
		var page []LibraryItem
		if len(raw) > 0 && raw[0] == '[' {
			if err := json.Unmarshal(raw, &page); err != nil {
				return 0, err
			}
		} else {
			var envelope struct {
				Results []LibraryItem `json:"results"`
			}
			if err := json.Unmarshal(raw, &envelope); err != nil {
				return 0, err
			}
			page = envelope.Results
		}
		all = append(all, page...)
		return len(page), nil
	})
	if err != nil {
		return nil, err
	}
	return all, nil
}
