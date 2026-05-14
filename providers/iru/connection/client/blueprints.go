// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package client

import "encoding/json"

// Blueprint is a configuration bundle assigned to devices.
type Blueprint struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	EnrollmentCode string   `json:"enrollment_code"`
	LibraryItems   []string `json:"library_items"`
	DevicesCount   int      `json:"devices_count"`
	Created        string   `json:"created"`
	UpdatedAt      string   `json:"updated_at"`
}

// ListBlueprints walks /v1/blueprints.
func (c *Client) ListBlueprints() ([]Blueprint, error) {
	var all []Blueprint
	err := c.paginate("/api/v1/blueprints", func(raw json.RawMessage) (int, error) {
		// Blueprints can come back either as a bare array or under
		// {"results": [...]}; tolerate both.
		var page []Blueprint
		if len(raw) > 0 && raw[0] == '[' {
			if err := json.Unmarshal(raw, &page); err != nil {
				return 0, err
			}
		} else {
			var envelope struct {
				Results []Blueprint `json:"results"`
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
