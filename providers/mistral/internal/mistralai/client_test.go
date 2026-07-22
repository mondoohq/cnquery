// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package mistralai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pagedServer serves File objects split into pages. It optionally sets
// has_more and/or total on the envelope so each pagination-termination path can
// be exercised independently. perPage controls how many items a single page
// returns, which lets us simulate a server that caps page_size below the
// requested value.
type pagedServer struct {
	total       int  // total items available
	perPage     int  // items returned per page (server-side cap)
	sendHasMore bool // include has_more in the envelope
	sendTotal   bool // include total in the envelope
	requests    int  // number of pages actually requested
}

func (s *pagedServer) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.requests++
		require.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		require.NoError(t, err)

		start := page * s.perPage
		data := []File{}
		for i := start; i < start+s.perPage && i < s.total; i++ {
			data = append(data, File{ID: fmt.Sprintf("file-%d", i)})
		}

		env := map[string]any{"object": "list", "data": data}
		if s.sendTotal {
			env["total"] = s.total
		}
		if s.sendHasMore {
			env["has_more"] = start+len(data) < s.total
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(env))
	}
}

func newTestClient(t *testing.T, h http.Handler) *Client {
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewClient("test-token", WithBaseURL(srv.URL))
}

func TestListPaged(t *testing.T) {
	tests := []struct {
		name        string
		total       int
		perPage     int
		sendHasMore bool
		sendTotal   bool
		wantItems   int
		wantPages   int
	}{
		{
			name:  "has_more drives multi-page collection",
			total: 230, perPage: defaultPageSize,
			sendHasMore: true, sendTotal: false,
			wantItems: 230, wantPages: 3,
		},
		{
			name:  "total drives multi-page collection when has_more absent",
			total: 230, perPage: defaultPageSize,
			sendTotal: true,
			wantItems: 230, wantPages: 3,
		},
		{
			name:  "short page terminates when no metadata present",
			total: 40, perPage: defaultPageSize,
			wantItems: 40, wantPages: 1,
		},
		{
			name:  "full page then empty page terminates without metadata",
			total: defaultPageSize, perPage: defaultPageSize,
			wantItems: defaultPageSize, wantPages: 2,
		},
		{
			// A server that caps page_size below the request must NOT be cut
			// short by the short-page heuristic while total is authoritative.
			name:  "server-side page_size cap does not truncate when total is present",
			total: 120, perPage: 50,
			sendTotal: true,
			wantItems: 120, wantPages: 3,
		},
		{
			name:  "empty result set",
			total: 0, perPage: defaultPageSize,
			sendTotal: true,
			wantItems: 0, wantPages: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := &pagedServer{
				total: tc.total, perPage: tc.perPage,
				sendHasMore: tc.sendHasMore, sendTotal: tc.sendTotal,
			}
			c := newTestClient(t, srv.handler(t))

			files, err := c.ListFiles(context.Background())
			require.NoError(t, err)
			assert.Len(t, files, tc.wantItems)
			assert.Equal(t, tc.wantPages, srv.requests, "page request count")

			// IDs must be contiguous and de-duplicated across pages.
			for i, f := range files {
				assert.Equal(t, fmt.Sprintf("file-%d", i), f.ID)
			}
		})
	}
}

func TestListPaged_ErrorPropagates(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"boom"}`))
	}))

	_, err := c.ListFiles(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestRequest_DetailStyleErrorSurfaces(t *testing.T) {
	// A 422 validation error uses "detail", not "message". The old code left
	// the surfaced error empty; it must now carry the detail payload.
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"detail":"page_size too large"}`))
	}))

	_, err := c.ListModels(context.Background())
	require.Error(t, err)
	assert.Equal(t, "page_size too large", err.Error())

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.StatusCode)
}

func TestRequest_NonJSONErrorBody(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream unavailable"))
	}))

	_, err := c.ListModels(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "502")
	assert.Contains(t, err.Error(), "upstream unavailable")
}

func TestAPIErrorError(t *testing.T) {
	tests := []struct {
		name string
		err  APIError
		want string
	}{
		{"message wins", APIError{Message: "nope", StatusCode: 400}, "nope"},
		{"detail string", APIError{Detail: json.RawMessage(`"bad field"`), StatusCode: 422}, "bad field"},
		{"detail array raw", APIError{Detail: json.RawMessage(`[{"msg":"x"}]`), StatusCode: 422}, `[{"msg":"x"}]`},
		{"status fallback", APIError{StatusCode: 500}, "mistral API error (status 500)"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.err.Error())
		})
	}
}

func TestIsAccessDenied(t *testing.T) {
	assert.True(t, IsAccessDenied(&APIError{StatusCode: 401}))
	assert.True(t, IsAccessDenied(&APIError{StatusCode: 403}))
	assert.False(t, IsAccessDenied(&APIError{StatusCode: 500}))
	assert.False(t, IsAccessDenied(fmt.Errorf("plain error")))
	assert.False(t, IsAccessDenied(nil))
}
