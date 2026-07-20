// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveOktaResourceTarget(t *testing.T) {
	tests := []struct {
		name     string
		orn      string
		href     string
		wantType string
		wantID   string
	}{
		{
			name:     "group ORN",
			orn:      "orn:okta:directory:00o1a2b3c4:groups:00g5d6e7f8",
			wantType: "group",
			wantID:   "00g5d6e7f8",
		},
		{
			name:     "user ORN",
			orn:      "orn:okta:directory:00o1a2b3c4:users:00u9g8h7i6",
			wantType: "user",
			wantID:   "00u9g8h7i6",
		},
		{
			name:     "app ORN with app-type segment",
			orn:      "orn:okta:idp:00o1a2b3c4:apps:oidc:0oa1b2c3d4",
			wantType: "application",
			wantID:   "0oa1b2c3d4",
		},
		{
			name:     "group URL",
			href:     "https://example.okta.com/api/v1/groups/00g5d6e7f8",
			wantType: "group",
			wantID:   "00g5d6e7f8",
		},
		{
			name:     "user URL with trailing path",
			href:     "https://example.okta.com/api/v1/users/00u9g8h7i6/roles",
			wantType: "user",
			wantID:   "00u9g8h7i6",
		},
		{
			name:     "application URL",
			href:     "https://example.okta.com/api/v1/apps/0oa1b2c3d4",
			wantType: "application",
			wantID:   "0oa1b2c3d4",
		},
		{
			name:     "ORN preferred over href",
			orn:      "orn:okta:directory:00o1a2b3c4:groups:fromORN",
			href:     "https://example.okta.com/api/v1/users/fromHREF",
			wantType: "group",
			wantID:   "fromORN",
		},
		{
			name:     "resource-set entry self-link does not false-match",
			href:     "https://example.okta.com/api/v1/iam/resource-sets/iam1/resources/ires1",
			wantType: "",
			wantID:   "",
		},
		{
			name:     "empty inputs",
			wantType: "",
			wantID:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotType, gotID := resolveOktaResourceTarget(tc.orn, tc.href)
			assert.Equal(t, tc.wantType, gotType)
			assert.Equal(t, tc.wantID, gotID)
		})
	}
}

func TestOktaRoleIdFromPermissionsHref(t *testing.T) {
	tests := []struct {
		href string
		want string
	}{
		{"https://example.okta.com/api/v1/iam/roles/cr0AbCdEf/permissions", "cr0AbCdEf"},
		{"https://example.okta.com/api/v1/iam/roles/cr0AbCdEf/permissions/okta.users.read", "cr0AbCdEf"},
		// A bare roles href (no /permissions suffix) still yields the role id.
		{"https://example.okta.com/api/v1/iam/roles/cr0AbCdEf", "cr0AbCdEf"},
		{"https://example.okta.com/api/v1/apps/0oa1", ""},
		{"", ""},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, oktaRoleIdFromPermissionsHref(tc.href))
	}
}

func TestOktaLinkHref(t *testing.T) {
	assert.Equal(t, "https://x/api/v1/groups/g1",
		oktaLinkHref(map[string]interface{}{"href": "https://x/api/v1/groups/g1"}))
	assert.Equal(t, "", oktaLinkHref(map[string]interface{}{"noHref": "x"}))
	assert.Equal(t, "", oktaLinkHref("not-a-map"))
	assert.Equal(t, "", oktaLinkHref(nil))
}

func TestLastPathSegment(t *testing.T) {
	assert.Equal(t, "g1", lastPathSegment("https://x/api/v1/groups/g1"))
	assert.Equal(t, "g1", lastPathSegment("https://x/api/v1/groups/g1/"))
	assert.Equal(t, "solo", lastPathSegment("solo"))
	assert.Equal(t, "", lastPathSegment(""))
}
