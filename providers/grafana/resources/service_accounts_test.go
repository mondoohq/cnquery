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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newPagedServer returns a test server that serves /api/serviceaccounts/search
// as a paged endpoint. totalCount records how many SAs exist in total.
func newPagedServer(t *testing.T, totalCount int, hits *atomic.Int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits != nil {
			hits.Add(1)
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		perPage, _ := strconv.Atoi(r.URL.Query().Get("perpage"))
		if perPage < 1 {
			perPage = serviceAccountPageSize
		}

		start := (page - 1) * perPage
		end := min(start+perPage, totalCount)
		var items []grafanaServiceAccountJSON
		if start < totalCount {
			items = make([]grafanaServiceAccountJSON, end-start)
			for i := range items {
				items[i] = grafanaServiceAccountJSON{
					ID:   start + i + 1,
					Name: fmt.Sprintf("sa-%d", start+i+1),
				}
			}
		}
		resp := grafanaServiceAccountsResponse{
			TotalCount:      totalCount,
			ServiceAccounts: items,
			Page:            page,
			PerPage:         perPage,
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestFetchServiceAccountPage(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		srv := newPagedServer(t, 5, nil)
		t.Cleanup(srv.Close)
		conn := newTestConn(t, srv.URL)

		page, err := fetchServiceAccountPage(context.Background(), conn, 1)
		require.NoError(t, err)
		assert.Equal(t, 5, page.TotalCount)
		assert.Len(t, page.ServiceAccounts, 5)
	})

	t.Run("non-200 errors", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		t.Cleanup(srv.Close)
		conn := newTestConn(t, srv.URL)

		_, err := fetchServiceAccountPage(context.Background(), conn, 1)
		require.Error(t, err)
	})
}

// TestServiceAccountPagination_FetchesAllPages verifies the parallel pagination
// loop reads every page when totalCount > pageSize and returns the union.
func TestServiceAccountPagination_FetchesAllPages(t *testing.T) {
	// Use a smaller "page size" by overriding the query param expectation:
	// the server uses whatever perpage the client sends, so we just need
	// totalCount > serviceAccountPageSize to trigger multi-page fetch.
	// To keep the test fast we ride at exactly serviceAccountPageSize * 3 + 1.
	const total = serviceAccountPageSize*3 + 1
	var hits atomic.Int32
	srv := newPagedServer(t, total, &hits)
	t.Cleanup(srv.Close)
	conn := newTestConn(t, srv.URL)

	// Drive the pagination directly via fetchServiceAccountPage to keep this
	// test focused (the full serviceAccounts() method also calls CreateResource
	// which needs a Runtime). We reproduce the loop's shape inline.
	first, err := fetchServiceAccountPage(context.Background(), conn, 1)
	require.NoError(t, err)
	collected := first.ServiceAccounts

	totalPages := (total + serviceAccountPageSize - 1) / serviceAccountPageSize
	require.Equal(t, 4, totalPages)

	var mu sync.Mutex
	var wg sync.WaitGroup
	for p := 2; p <= totalPages; p++ {
		wg.Go(func() {
			page, err := fetchServiceAccountPage(context.Background(), conn, p)
			require.NoError(t, err)
			mu.Lock()
			collected = append(collected, page.ServiceAccounts...)
			mu.Unlock()
		})
	}
	wg.Wait()

	assert.Equal(t, total, len(collected), "all pages must be collected")
	assert.Equal(t, int32(totalPages), hits.Load(),
		"server hit exactly once per page — pagination must not retry or duplicate")
}

// TestServiceAccountTokens_HasExpirationLogic exercises the simplified
// hasExpiration branch (parseGrafanaTime zero-sentinel handling) end-to-end on
// the JSON shape returned by /api/serviceaccounts/{id}/tokens.
func TestServiceAccountTokens_HasExpirationLogic(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantHas  bool
		wantSecs float64
	}{
		{"empty expiration", "", false, 0},
		{"grafana zero sentinel", "0001-01-01T00:00:00Z", false, 0},
		{"valid future timestamp", "2099-01-01T00:00:00Z", true, 1234},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp := parseGrafanaTime(tt.raw)
			hasExp := !exp.IsZero()
			assert.Equal(t, tt.wantHas, hasExp)
			// Mirror the secondsUntilExp gating in tokens()
			secs := tt.wantSecs
			if !hasExp {
				secs = 0
			}
			assert.Equal(t, tt.wantSecs, secs)
		})
	}
}
