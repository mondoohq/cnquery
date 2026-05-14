// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package client

import "encoding/json"

// User is an end-user record in the Iru directory. The shape is shared
// between the top-level /users listing and the embedded `user` field on a
// device row.
type User struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Username   string `json:"username"`
	IsArchived bool   `json:"is_archived"`
	Department string `json:"department"`
	JobTitle   string `json:"job_title"`
}

// ListUsers walks /v1/users.
func (c *Client) ListUsers() ([]User, error) {
	var all []User
	err := c.paginate("/api/v1/users", func(raw json.RawMessage) (int, error) {
		var page []User
		if len(raw) > 0 && raw[0] == '[' {
			if err := json.Unmarshal(raw, &page); err != nil {
				return 0, err
			}
		} else {
			var envelope struct {
				Results []User `json:"results"`
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
