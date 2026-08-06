// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractPowerBiEnvelope(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		envelope string
		expected string
		wantErr  bool
	}{
		{
			name:     "collection under value",
			body:     `{"@odata.context":"ctx","value":[{"id":"a"},{"id":"b"}]}`,
			envelope: "value",
			expected: `[{"id":"a"},{"id":"b"}]`,
		},
		{
			name:     "artifact access entities envelope",
			body:     `{"ArtifactAccessEntities":[{"artifactId":"1"}]}`,
			envelope: "ArtifactAccessEntities",
			expected: `[{"artifactId":"1"}]`,
		},
		{
			name:     "fabric tenant settings envelope",
			body:     `{"tenantSettings":[{"settingName":"s"}]}`,
			envelope: "tenantSettings",
			expected: `[{"settingName":"s"}]`,
		},
		{
			// an endpoint with nothing to report omits the property; that is an
			// empty result, not a failure
			name:     "missing envelope yields nil",
			body:     `{"@odata.context":"ctx"}`,
			envelope: "value",
			expected: "",
		},
		{
			name:     "empty collection",
			body:     `{"value":[]}`,
			envelope: "value",
			expected: `[]`,
		},
		{
			name:     "malformed json",
			body:     `not json`,
			envelope: "value",
			wantErr:  true,
		},
		{
			name:     "top level array is not an envelope",
			body:     `[{"id":"a"}]`,
			envelope: "value",
			wantErr:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw, err := extractPowerBiEnvelope([]byte(test.body), test.envelope)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.expected, string(raw))
		})
	}
}

func TestPowerBiGet(t *testing.T) {
	t.Run("sends bearer token and extracts envelope", func(t *testing.T) {
		var gotAuth, gotAccept string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			gotAccept = r.Header.Get("Accept")
			fmt.Fprint(w, `{"value":[{"id":"cap-1"}]}`)
		}))
		defer srv.Close()

		raw, err := powerBiGet(context.Background(), "tok-123", srv.URL, "value")
		require.NoError(t, err)
		assert.Equal(t, `[{"id":"cap-1"}]`, string(raw))
		assert.Equal(t, "Bearer tok-123", gotAuth)
		assert.Equal(t, "application/json", gotAccept)
	})

	// 401 and 403 are the shapes a tenant hits when the service principal is
	// not allowed on the admin APIs, so both must produce the actionable
	// message rather than a bare status code
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run("access denied on "+strconv.Itoa(status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				fmt.Fprint(w, `{"error":"PowerBINotLicensed"}`)
			}))
			defer srv.Close()

			_, err := powerBiGet(context.Background(), "tok", srv.URL, "value")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "read-only admin API access")
		})
	}

	t.Run("other status carries code and body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "backend exploded")
		}))
		defer srv.Close()

		_, err := powerBiGet(context.Background(), "tok", srv.URL, "value")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
		assert.Contains(t, err.Error(), "backend exploded")
	})

	t.Run("malformed success body errors", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"value":`)
		}))
		defer srv.Close()

		_, err := powerBiGet(context.Background(), "tok", srv.URL, "value")
		require.Error(t, err)
	})

	t.Run("unreachable host errors", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL
		srv.Close()

		_, err := powerBiGet(context.Background(), "tok", url, "value")
		require.Error(t, err)
	})
}

// workspacePageServer serves `total` workspaces through the admin groups
// endpoint, honouring $top and $skip the way the real API does.
func workspacePageServer(t *testing.T, total int, requests *[]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*requests = append(*requests, r.URL.RequestURI())

		top, err := strconv.Atoi(r.URL.Query().Get("$top"))
		require.NoError(t, err)
		skip, err := strconv.Atoi(r.URL.Query().Get("$skip"))
		require.NoError(t, err)
		assert.Equal(t, "users", r.URL.Query().Get("$expand"))

		items := []string{}
		for i := skip; i < skip+top && i < total; i++ {
			items = append(items, fmt.Sprintf(`{"id":"ws-%d"}`, i))
		}
		fmt.Fprintf(w, `{"value":[%s]}`, strings.Join(items, ","))
	}))
}

func TestFetchPowerBiWorkspacePages(t *testing.T) {
	// the boundary cases are what a truncation bug hides behind: a tenant whose
	// workspace count is an exact multiple of the page size only reveals the
	// bug on the follow-up empty page
	tests := []struct {
		name          string
		total         int
		pageSize      int
		wantRequests  int
		wantWorkspace int
	}{
		{name: "no workspaces", total: 0, pageSize: 2, wantRequests: 1, wantWorkspace: 0},
		{name: "single short page", total: 1, pageSize: 2, wantRequests: 1, wantWorkspace: 1},
		{name: "exact single page", total: 2, pageSize: 2, wantRequests: 2, wantWorkspace: 2},
		{name: "spills to second page", total: 3, pageSize: 2, wantRequests: 2, wantWorkspace: 3},
		{name: "exact two pages", total: 4, pageSize: 2, wantRequests: 3, wantWorkspace: 4},
		{name: "many pages", total: 11, pageSize: 2, wantRequests: 6, wantWorkspace: 11},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var requests []string
			srv := workspacePageServer(t, test.total, &requests)
			defer srv.Close()

			raw, err := fetchPowerBiWorkspacePages(context.Background(), "tok", srv.URL+"/", test.pageSize)
			require.NoError(t, err)

			var got []powerBiWorkspace
			require.NoError(t, json.Unmarshal(raw, &got))
			assert.Len(t, got, test.wantWorkspace)
			assert.Len(t, requests, test.wantRequests)

			// every workspace is present exactly once and in order
			for i := range got {
				assert.Equal(t, fmt.Sprintf("ws-%d", i), got[i].Id)
			}
		})
	}
}

func TestFetchPowerBiWorkspacePagesRequestShape(t *testing.T) {
	var requests []string
	srv := workspacePageServer(t, 3, &requests)
	defer srv.Close()

	_, err := fetchPowerBiWorkspacePages(context.Background(), "tok", srv.URL+"/", 2)
	require.NoError(t, err)

	require.Len(t, requests, 2)
	assert.Contains(t, requests[0], "admin/groups")
	assert.Contains(t, requests[0], "$skip=0")
	assert.Contains(t, requests[1], "$skip=2")
}

func TestFetchPowerBiWorkspacePagesPropagatesError(t *testing.T) {
	// a failure on a later page must not silently return the pages already
	// collected, which would report a partial workspace list as complete
	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if page > 0 {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		page++
		fmt.Fprint(w, `{"value":[{"id":"ws-0"},{"id":"ws-1"}]}`)
	}))
	defer srv.Close()

	_, err := fetchPowerBiWorkspacePages(context.Background(), "tok", srv.URL+"/", 2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read-only admin API access")
}

func TestUnmarshalPowerBiSection(t *testing.T) {
	t.Run("section error takes precedence", func(t *testing.T) {
		msg := "insufficient privileges"
		_, err := unmarshalPowerBiSection[powerBiCapacity](powerBiSection{
			Data:  json.RawMessage(`[{"id":"c1"}]`),
			Error: &msg,
		})
		require.Error(t, err)
		assert.Equal(t, msg, err.Error())
	})

	t.Run("empty error string is not an error", func(t *testing.T) {
		empty := ""
		out, err := unmarshalPowerBiSection[powerBiCapacity](powerBiSection{
			Data:  json.RawMessage(`[{"id":"c1"}]`),
			Error: &empty,
		})
		require.NoError(t, err)
		assert.Len(t, out, 1)
	})

	t.Run("nil payload", func(t *testing.T) {
		out, err := unmarshalPowerBiSection[powerBiCapacity](powerBiSection{})
		require.NoError(t, err)
		assert.Empty(t, out)
	})

	t.Run("json null payload", func(t *testing.T) {
		out, err := unmarshalPowerBiSection[powerBiCapacity](powerBiSection{Data: json.RawMessage(`null`)})
		require.NoError(t, err)
		assert.Empty(t, out)
	})

	t.Run("array payload", func(t *testing.T) {
		out, err := unmarshalPowerBiSection[powerBiCapacity](powerBiSection{
			Data: json.RawMessage(`[{"id":"c1","sku":"A1"},{"id":"c2","sku":"A2"}]`),
		})
		require.NoError(t, err)
		require.Len(t, out, 2)
		assert.Equal(t, "c1", out[0].Id)
		assert.Equal(t, "A2", out[1].Sku)
	})

	t.Run("bare object payload", func(t *testing.T) {
		out, err := unmarshalPowerBiSection[powerBiCapacity](powerBiSection{
			Data: json.RawMessage(`{"id":"c1","sku":"A1"}`),
		})
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.Equal(t, "c1", out[0].Id)
	})

	t.Run("decode failure surfaces", func(t *testing.T) {
		_, err := unmarshalPowerBiSection[powerBiCapacity](powerBiSection{
			Data: json.RawMessage(`{"id":`),
		})
		require.Error(t, err)
	})
}

func TestPowerBiWorkspaceDecodesUsers(t *testing.T) {
	// $expand=users nests the access list inside each workspace; losing it
	// would blank microsoft.powerbi.workspace.users without any error
	raw := `[{"id":"ws-1","name":"Finance","isOnDedicatedCapacity":true,"capacityId":"cap-1",
	  "users":[{"emailAddress":"a@contoso.com","groupUserAccessRight":"Admin","principalType":"User"}]}]`

	out, err := unmarshalPowerBiSection[powerBiWorkspace](powerBiSection{Data: json.RawMessage(raw)})
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "Finance", out[0].Name)
	assert.True(t, out[0].IsOnDedicatedCapacity)
	assert.Equal(t, "cap-1", out[0].CapacityId)
	require.Len(t, out[0].Users, 1)
	assert.Equal(t, "Admin", out[0].Users[0].GroupUserAccessRight)
}
