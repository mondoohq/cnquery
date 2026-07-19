// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type pagedItem struct {
	ID string `json:"id"`
}

func testConn(srv *httptest.Server) *VercelConnection {
	return &VercelConnection{
		token:   "test-token",
		baseURL: srv.URL,
		client:  srv.Client(),
	}
}

func TestGetPagedFollowsCursor(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("missing/incorrect auth header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("until") {
		case "":
			// first page hands back a cursor
			io.WriteString(w, `{"items":[{"id":"a"},{"id":"b"}],"pagination":{"next":100}}`)
		case "100":
			// last page terminates the cursor
			io.WriteString(w, `{"items":[{"id":"c"}],"pagination":{"next":null}}`)
		default:
			t.Errorf("unexpected until cursor: %q", r.URL.Query().Get("until"))
		}
	}))
	defer srv.Close()

	got, err := GetPaged[pagedItem](context.Background(), testConn(srv), "/things", nil, "items")
	if err != nil {
		t.Fatalf("GetPaged: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 page requests, got %d", calls)
	}
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i].ID != want[i] {
			t.Fatalf("index %d: expected %q, got %q", i, want[i], got[i].ID)
		}
	}
}

func TestGetPagedSinglePageNoPagination(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		io.WriteString(w, `{"items":[{"id":"only"}]}`)
	}))
	defer srv.Close()

	got, err := GetPaged[pagedItem](context.Background(), testConn(srv), "/things", nil, "items")
	if err != nil {
		t.Fatalf("GetPaged: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 request without a pagination envelope, got %d", calls)
	}
	if len(got) != 1 || got[0].ID != "only" {
		t.Fatalf("expected [only], got %v", got)
	}
}

func TestGetPagedMissingKeyReturnsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"other":[{"id":"x"}]}`)
	}))
	defer srv.Close()

	got, err := GetPaged[pagedItem](context.Background(), testConn(srv), "/things", nil, "items")
	if err != nil {
		t.Fatalf("GetPaged: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty result when key absent, got %v", got)
	}
}

func TestGetPagedSurfacesAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, `{"error":{"code":"forbidden","message":"nope"}}`)
	}))
	defer srv.Close()

	_, err := GetPaged[pagedItem](context.Background(), testConn(srv), "/things", nil, "items")
	if err == nil {
		t.Fatal("expected an error on 403")
	}
	if !IsForbidden(err) {
		t.Fatalf("expected IsForbidden to be true, got %v", err)
	}
	var apiErr *APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("expected an *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusForbidden || apiErr.Code != "forbidden" || apiErr.Message != "nope" {
		t.Fatalf("APIError not populated from envelope: %+v", apiErr)
	}
}

func TestErrorClassificationThroughWrap(t *testing.T) {
	// IsForbidden / IsNotFound must see through a wrapped APIError, otherwise
	// enterprise-gated 403s propagate as hard errors instead of degrading.
	forbidden := fmt.Errorf("call failed: %w", &APIError{StatusCode: http.StatusForbidden})
	if !IsForbidden(forbidden) {
		t.Error("IsForbidden should unwrap a wrapped 403")
	}
	if IsNotFound(forbidden) {
		t.Error("IsNotFound should be false for a 403")
	}

	notFound := fmt.Errorf("call failed: %w", &APIError{StatusCode: http.StatusNotFound})
	if !IsNotFound(notFound) {
		t.Error("IsNotFound should unwrap a wrapped 404")
	}
	if IsForbidden(notFound) {
		t.Error("IsForbidden should be false for a 404")
	}

	if IsForbidden(fmt.Errorf("plain error")) {
		t.Error("IsForbidden should be false for a non-APIError")
	}
}
