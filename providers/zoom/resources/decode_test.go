// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"

	"go.mondoo.com/mql/v13/providers/zoom/connection"
)

// The zoom.user resource decodes its records by struct tag alone and derives
// the `verified` and `ssoLinked` booleans from numeric fields. A mistyped tag
// or an off-by-one in a derivation compiles, lints, and yields a confident
// wrong answer rather than an error, so these tests pin the decoding and the
// derivations against payloads shaped like the documented API response.

func TestUserRecordDecodesFields(t *testing.T) {
	const payload = `{
		"id": "z9f8abc",
		"email": "jane@acme.com",
		"first_name": "Jane",
		"last_name": "Doe",
		"display_name": "Jane Doe",
		"type": 2,
		"status": "active",
		"verified": 1,
		"login_type": 100,
		"role_id": "0",
		"group_ids": ["g1", "g2"],
		"last_login_time": "2024-01-02T03:04:05Z",
		"created_at": "2023-06-07T08:09:10Z"
	}`

	var u connection.User
	if err := json.Unmarshal([]byte(payload), &u); err != nil {
		t.Fatalf("failed to decode user payload: %v", err)
	}

	if u.ID != "z9f8abc" {
		t.Errorf("id: got %q, want %q", u.ID, "z9f8abc")
	}
	if u.Email != "jane@acme.com" {
		t.Errorf("email: got %q, want %q", u.Email, "jane@acme.com")
	}
	if u.Type != 2 {
		t.Errorf("type: got %d, want 2", u.Type)
	}
	if u.Status != "active" {
		t.Errorf("status: got %q, want %q", u.Status, "active")
	}
	if u.Verified != 1 {
		t.Errorf("verified: got %d, want 1", u.Verified)
	}
	if u.LoginType != 100 {
		t.Errorf("loginType: got %d, want 100", u.LoginType)
	}
	if u.RoleID != "0" {
		t.Errorf("roleId: got %q, want %q", u.RoleID, "0")
	}
	if len(u.GroupIDs) != 2 || u.GroupIDs[0] != "g1" || u.GroupIDs[1] != "g2" {
		t.Errorf("groupIds: got %v, want [g1 g2]", u.GroupIDs)
	}
	if u.LastLoginTime == nil {
		t.Error("lastLoginTime: got nil, want a timestamp")
	}
	if u.CreatedAt == nil {
		t.Error("createdAt: got nil, want a timestamp")
	}
}

func TestUserRecordAbsentTimestampsStayNil(t *testing.T) {
	const payload = `{"id": "u1", "email": "a@b.com", "type": 1, "status": "pending"}`

	var u connection.User
	if err := json.Unmarshal([]byte(payload), &u); err != nil {
		t.Fatalf("failed to decode user payload: %v", err)
	}

	// An absent timestamp must stay null rather than becoming the zero time,
	// which would report 1 January year 1 as a real login/creation date.
	if u.LastLoginTime != nil {
		t.Errorf("lastLoginTime: got %v, want nil", u.LastLoginTime)
	}
	if u.CreatedAt != nil {
		t.Errorf("createdAt: got %v, want nil", u.CreatedAt)
	}
}

func TestUserVerified(t *testing.T) {
	cases := []struct {
		name     string
		verified int
		want     bool
	}{
		{"unverified zero", 0, false},
		{"verified one", 1, true},
		{"any nonzero counts as verified", 2, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := &connection.User{Verified: tc.verified}
			if got := userVerified(u); got != tc.want {
				t.Errorf("userVerified(%d): got %v, want %v", tc.verified, got, tc.want)
			}
		})
	}
}

func TestUserSsoLinked(t *testing.T) {
	cases := []struct {
		name      string
		loginType int64
		want      bool
	}{
		{"sso login type", 100, true},
		{"google oauth", 1, false},
		{"zero default", 0, false},
		{"api login type", 99, false},
		{"zoom work email", 101, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := &connection.User{LoginType: tc.loginType}
			if got := userSsoLinked(u); got != tc.want {
				t.Errorf("userSsoLinked(%d): got %v, want %v", tc.loginType, got, tc.want)
			}
		})
	}
}
