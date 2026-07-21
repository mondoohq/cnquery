// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	cloudflarev6 "github.com/cloudflare/cloudflare-go/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
)

func TestTimeOrNil(t *testing.T) {
	// A zero time (how the v6 SDK models a JSON null timestamp) must resolve to
	// MQL null, not the 0001-01-01 zero value — the basis of the null-time fix.
	assert.Equal(t, llx.NilData, timeOrNil(time.Time{}))

	ts := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	got := timeOrNil(ts)
	require.NotEqual(t, llx.NilData, got)
	gotTime, ok := got.Value.(*time.Time)
	require.True(t, ok, "non-zero time must carry a *time.Time value")
	assert.True(t, gotTime.Equal(ts))
}

func TestParseRFC3339(t *testing.T) {
	assert.True(t, parseRFC3339("").IsZero(), "empty string is zero time")
	assert.True(t, parseRFC3339("not-a-timestamp").IsZero(), "unparseable string is zero time")

	got := parseRFC3339("2026-07-20T12:00:00Z")
	require.False(t, got.IsZero())
	assert.Equal(t, 2026, got.Year())
	assert.Equal(t, time.July, got.Month())
}

func TestIsUnavailable(t *testing.T) {
	for _, code := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound} {
		assert.Truef(t, isUnavailable(&cloudflarev6.Error{StatusCode: code}), "status %d should be treated as unavailable", code)
	}
	for _, code := range []int{http.StatusInternalServerError, http.StatusTooManyRequests, http.StatusBadRequest} {
		assert.Falsef(t, isUnavailable(&cloudflarev6.Error{StatusCode: code}), "status %d should NOT be unavailable", code)
	}
	assert.False(t, isUnavailable(errors.New("plain error")))
	assert.False(t, isUnavailable(nil))
}

func TestDegradedList(t *testing.T) {
	// An unavailable-resource error degrades to an empty (non-nil) list.
	got, err := degradedList(&cloudflarev6.Error{StatusCode: http.StatusForbidden})
	require.NoError(t, err)
	assert.Equal(t, []any{}, got)

	// A 404 likewise degrades.
	got, err = degradedList(&cloudflarev6.Error{StatusCode: http.StatusNotFound})
	require.NoError(t, err)
	assert.Empty(t, got)

	// Any other error propagates unchanged (not swallowed).
	sentinel := errors.New("rate limited")
	got, err = degradedList(&cloudflarev6.Error{StatusCode: http.StatusTooManyRequests})
	assert.Nil(t, got)
	require.Error(t, err)

	got, err = degradedList(sentinel)
	assert.Nil(t, got)
	assert.Equal(t, sentinel, err)
}

type pagedTestItem struct {
	ID string `json:"id"`
}

// TestCfGetPagedTerminatesOnZeroEchoedPage is the regression guard for the
// pagination-termination fix: when the API advertises total_pages>1 but echoes
// result_info.page as 0 (or omits it), the old `ResultInfo.Page >= TotalPages`
// check never became true and the loop spun forever. The fix compares the local
// page counter instead, so the walk stops after total_pages requests.
func TestCfGetPagedTerminatesOnZeroEchoedPage(t *testing.T) {
	env := setupTestEnv(t)

	const totalPages = 3
	var calls int32
	env.Mux.HandleFunc("/widgets", func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if int(n) > totalPages+2 {
			// Bound a regression so it fails loudly instead of hanging CI.
			t.Errorf("cfGetPaged did not terminate: %d requests for %d pages", n, totalPages)
			jsonResponse(w, `{"success":true,"result":[],"result_info":{"page":0,"total_pages":0}}`)
			return
		}
		reqPage, _ := strconv.Atoi(r.URL.Query().Get("page"))
		// Echo page:0 regardless of the requested page while advertising 3 pages.
		jsonResponse(w, fmt.Sprintf(`{"success":true,"errors":[],"messages":[],
			"result":[{"id":"w%d"}],
			"result_info":{"page":0,"per_page":100,"total_pages":%d}}`, reqPage, totalPages))
	})

	got, err := cfGetPaged[pagedTestItem](env.Conn, "widgets")
	require.NoError(t, err)
	require.Len(t, got, totalPages, "one item collected per page, then stop")
	assert.Equal(t, int32(totalPages), atomic.LoadInt32(&calls), "exactly one request per page — no infinite loop")
	assert.Equal(t, []string{"w1", "w2", "w3"}, []string{got[0].ID, got[1].ID, got[2].ID})
}
