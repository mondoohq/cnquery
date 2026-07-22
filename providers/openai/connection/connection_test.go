// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsAdminToken(t *testing.T) {
	tests := []struct {
		token string
		want  bool
	}{
		{"sk-admin-abc123", true},
		{"sk-proj-abc123", false},
		{"sk-abc123", false},
		{"", false},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, isAdminToken(tc.token), tc.token)
	}
}

func TestFetchAccountInfo(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantOrgID string
		wantName  string
	}{
		{
			name:      "single org",
			body:      `{"orgs":{"data":[{"id":"org-1","name":"Acme","is_default":true}]}}`,
			wantOrgID: "org-1",
			wantName:  "Acme",
		},
		{
			name:      "prefers the default org over the first",
			body:      `{"orgs":{"data":[{"id":"org-1","name":"First","is_default":false},{"id":"org-2","name":"Default","is_default":true}]}}`,
			wantOrgID: "org-2",
			wantName:  "Default",
		},
		{
			name:      "falls back to the first org when none is default",
			body:      `{"orgs":{"data":[{"id":"org-1","name":"First","is_default":false},{"id":"org-2","name":"Second","is_default":false}]}}`,
			wantOrgID: "org-1",
			wantName:  "First",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/me", r.URL.Path)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			info, err := fetchAccountInfo(srv.URL, "test-token")
			require.NoError(t, err)
			assert.Equal(t, tc.wantOrgID, info.OrgID)
			assert.Equal(t, tc.wantName, info.OrgName)
		})
	}
}
