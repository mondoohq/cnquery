// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

// /v1/me is undocumented, so its response size is not a contract we control.
// A body past the cap must stop the read rather than let connect-time org
// detection allocate whatever the endpoint sends.
func TestFetchAccountInfoCapsTheBody(t *testing.T) {
	padding := strings.Repeat("x", maxAccountInfoBody)
	body := `{"orgs":{"data":[{"id":"org-abc","name":"` + padding + `","is_default":true}]}}`
	require.Greater(t, len(body), maxAccountInfoBody, "the test body has to exceed the cap to exercise it")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	defer srv.Close()

	info, err := fetchAccountInfo(srv.URL, "sk-test")
	require.Error(t, err, "the read stopped at the cap")
	assert.Contains(t, err.Error(), "exceeded the 1048576 byte limit",
		"a capped read has to name itself: the truncated JSON would otherwise fail with a generic syntax error that reads like a malformed response")
	assert.Nil(t, info)
}
