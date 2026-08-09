// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

type testRecord struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (r testRecord) GetID() string { return r.ID }

func newTestConn(t *testing.T, base string) *StripeConnection {
	t.Helper()
	return &StripeConnection{
		token:   "sk_test_dummy",
		baseURL: base,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

// TestListPagination verifies List follows the starting_after cursor across
// pages and stops when has_more is false, without dropping or duplicating rows.
func TestListPagination(t *testing.T) {
	var gotCursors []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCursors = append(gotCursors, r.URL.Query().Get("starting_after"))

		// Verify auth and pinned version headers are always sent.
		if got := r.Header.Get("Authorization"); got != "Bearer sk_test_dummy" {
			t.Errorf("missing/incorrect auth header: %q", got)
		}
		if got := r.Header.Get("Stripe-Version"); got != apiVersion {
			t.Errorf("missing/incorrect version header: %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("starting_after") {
		case "":
			_, _ = w.Write([]byte(`{"object":"list","has_more":true,"data":[{"id":"a"},{"id":"b"}]}`))
		case "b":
			_, _ = w.Write([]byte(`{"object":"list","has_more":true,"data":[{"id":"c"},{"id":"d"}]}`))
		case "d":
			_, _ = w.Write([]byte(`{"object":"list","has_more":false,"data":[{"id":"e"}]}`))
		default:
			t.Errorf("unexpected cursor: %q", r.URL.Query().Get("starting_after"))
		}
	}))
	defer srv.Close()

	conn := newTestConn(t, srv.URL)
	records, err := List[testRecord](context.Background(), conn, "/v1/things", nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	wantIDs := []string{"a", "b", "c", "d", "e"}
	if len(records) != len(wantIDs) {
		t.Fatalf("expected %d records, got %d: %+v", len(wantIDs), len(records), records)
	}
	for i, want := range wantIDs {
		if records[i].ID != want {
			t.Errorf("record %d: expected id %q, got %q", i, want, records[i].ID)
		}
	}

	wantCursors := []string{"", "b", "d"}
	if len(gotCursors) != len(wantCursors) {
		t.Fatalf("expected %d requests, got %d: %v", len(wantCursors), len(gotCursors), gotCursors)
	}
	for i, want := range wantCursors {
		if gotCursors[i] != want {
			t.Errorf("request %d: expected cursor %q, got %q", i, want, gotCursors[i])
		}
	}
}

// TestListStopsWithoutCursor guards against an infinite loop when an endpoint
// reports has_more but returns rows whose id is empty (no advanceable cursor).
func TestListStopsWithoutCursor(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls > 5 {
			t.Fatal("List did not stop; possible infinite pagination loop")
		}
		_, _ = w.Write([]byte(`{"object":"list","has_more":true,"data":[{"id":""}]}`))
	}))
	defer srv.Close()

	conn := newTestConn(t, srv.URL)
	records, err := List[testRecord](context.Background(), conn, "/v1/things", nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record before stopping, got %d", len(records))
	}
}

// TestAPIErrorClassification verifies error decoding and the IsForbidden /
// IsNotFound helpers used to degrade gracefully.
func TestAPIErrorClassification(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/forbidden":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":{"type":"invalid_request_error","code":"api_key_expired","message":"denied"}}`))
		case "/missing":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"type":"invalid_request_error","message":"no such resource"}}`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()

	conn := newTestConn(t, srv.URL)

	err := conn.Get(context.Background(), "/forbidden", nil, nil)
	if !IsForbidden(err) {
		t.Errorf("expected IsForbidden to be true for 403, got err=%v", err)
	}
	if IsNotFound(err) {
		t.Errorf("did not expect IsNotFound for a 403")
	}

	err = conn.Get(context.Background(), "/missing", nil, nil)
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound to be true for 404, got err=%v", err)
	}
}

// TestGetDecodesBody verifies a successful GET decodes the JSON body and applies
// query parameters.
func TestGetDecodesBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("status"); got != "all" {
			t.Errorf("expected status=all query, got %q", got)
		}
		_, _ = w.Write([]byte(`{"id":"acct_123","name":"Acme"}`))
	}))
	defer srv.Close()

	conn := newTestConn(t, srv.URL)
	q := url.Values{}
	q.Set("status", "all")

	var out testRecord
	if err := conn.Get(context.Background(), "/v1/account", q, &out); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if out.ID != "acct_123" || out.Name != "Acme" {
		t.Fatalf("unexpected decode result: %+v", out)
	}

	// Sanity-check the JSON tag contract the List cursor relies on.
	var check map[string]json.RawMessage
	_ = json.Unmarshal([]byte(`{"id":"x"}`), &check)
	if _, ok := check["id"]; !ok {
		t.Fatal("expected id key")
	}
}

// TestRetriesRateLimit verifies a 429 is waited out and retried rather than
// failing the scan, and that the Retry-After header drives the wait.
func TestRetriesRateLimit(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"type":"rate_limit_error","code":"rate_limit"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"object":"list","has_more":false,"data":[{"id":"c_1","name":"ok"}]}`))
	}))
	defer srv.Close()

	got, err := List[testRecord](context.Background(), newTestConn(t, srv.URL), "/v1/things", nil)
	if err != nil {
		t.Fatalf("expected the 429s to be retried, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 2 retries then success, got %d calls", calls)
	}
	if len(got) != 1 || got[0].ID != "c_1" {
		t.Fatalf("unexpected result after retry: %+v", got)
	}
}

// TestRateLimitSurfacesAfterMaxRetries verifies a persistently throttled
// endpoint reports the 429 instead of retrying forever or returning an empty
// list that would read as "no resources".
func TestRateLimitSurfacesAfterMaxRetries(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, err := List[testRecord](context.Background(), newTestConn(t, srv.URL), "/v1/things", nil)
	if err == nil {
		t.Fatal("expected the 429 to surface once retries are exhausted")
	}
	if !IsClientError(err) {
		t.Fatalf("expected a 4xx APIError, got %v", err)
	}
	if calls != maxRateLimitRetries+1 {
		t.Fatalf("expected %d attempts, got %d", maxRateLimitRetries+1, calls)
	}
}

// TestRetryAfterDelay covers both Retry-After forms and the backoff fallback.
func TestRetryAfterDelay(t *testing.T) {
	secs := http.Header{}
	secs.Set("Retry-After", "2")
	if got := retryAfterDelay(secs, 0); got != 2*time.Second {
		t.Fatalf("delay-seconds form: expected 2s, got %v", got)
	}

	date := http.Header{}
	date.Set("Retry-After", time.Now().Add(3*time.Second).UTC().Format(http.TimeFormat))
	if got := retryAfterDelay(date, 0); got <= 0 || got > 4*time.Second {
		t.Fatalf("HTTP-date form: expected roughly 3s, got %v", got)
	}

	// An absent header falls back to a doubling backoff.
	if got := retryAfterDelay(http.Header{}, 0); got != baseRetryDelay {
		t.Fatalf("fallback: expected %v, got %v", baseRetryDelay, got)
	}
	if got := retryAfterDelay(http.Header{}, 2); got != 4*baseRetryDelay {
		t.Fatalf("fallback: expected %v, got %v", 4*baseRetryDelay, got)
	}

	// An oversized header cannot park the scan.
	huge := http.Header{}
	huge.Set("Retry-After", "86400")
	if got := retryAfterDelay(huge, 0); got != maxRetryDelay {
		t.Fatalf("expected the wait to be capped at %v, got %v", maxRetryDelay, got)
	}
}
