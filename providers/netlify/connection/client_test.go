// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

type pagedItem struct {
	ID string `json:"id"`
}

func testConn(srv *httptest.Server) *NetlifyConnection {
	return &NetlifyConnection{
		token:   "test-token",
		baseURL: srv.URL,
		client:  srv.Client(),
	}
}

// writeItems answers with a page of n items, each carrying a unique id derived
// from the page number so the test can tell pages apart.
func writeItems(w http.ResponseWriter, page, n int) {
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, "[")
	for i := 0; i < n; i++ {
		if i > 0 {
			io.WriteString(w, ",")
		}
		fmt.Fprintf(w, `{"id":"p%d-%d"}`, page, i)
	}
	io.WriteString(w, "]")
}

func TestGetPagedWalksUntilShortPage(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("missing/incorrect auth header: %q", got)
		}
		if got := r.URL.Query().Get("per_page"); got != strconv.Itoa(pageSize) {
			t.Errorf("expected per_page=%d, got %q", pageSize, got)
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		switch page {
		case 1:
			writeItems(w, 1, pageSize)
		case 2:
			writeItems(w, 2, 3)
		default:
			t.Errorf("unexpected page request: %d", page)
		}
	}))
	defer srv.Close()

	got, err := GetPaged[pagedItem](context.Background(), testConn(srv), "/things", nil)
	if err != nil {
		t.Fatalf("GetPaged: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 page requests, got %d", calls)
	}
	if len(got) != pageSize+3 {
		t.Fatalf("expected %d records, got %d", pageSize+3, len(got))
	}
	if got[0].ID != "p1-0" || got[pageSize].ID != "p2-0" {
		t.Fatalf("pages collected out of order: %q then %q", got[0].ID, got[pageSize].ID)
	}
}

// An endpoint that ignores the page parameter replays the first page forever.
// Without the duplicate guard the walk would collect the same records up to the
// page cap, silently multiplying every result.
func TestGetPagedStopsWhenEndpointIgnoresPaging(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		writeItems(w, 1, pageSize)
	}))
	defer srv.Close()

	got, err := GetPaged[pagedItem](context.Background(), testConn(srv), "/things", nil)
	if err != nil {
		t.Fatalf("GetPaged: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected the walk to stop after the repeated page, got %d requests", calls)
	}
	if len(got) != pageSize {
		t.Fatalf("expected the repeated page to be dropped, got %d records", len(got))
	}
}

func TestGetPagedSinglePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeItems(w, 1, 2)
	}))
	defer srv.Close()

	got, err := GetPaged[pagedItem](context.Background(), testConn(srv), "/things", nil)
	if err != nil {
		t.Fatalf("GetPaged: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 records, got %d", len(got))
	}
}

func TestErrorsCarryStatusAndMessage(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		body        string
		wantMessage string
		forbidden   bool
		notFound    bool
	}{
		{
			name:        "forbidden with message envelope",
			status:      http.StatusForbidden,
			body:        `{"message":"You must be an owner"}`,
			wantMessage: "You must be an owner",
			forbidden:   true,
		},
		{
			name:        "unauthorized counts as forbidden",
			status:      http.StatusUnauthorized,
			body:        `{"message":"Access denied"}`,
			wantMessage: "Access denied",
			forbidden:   true,
		},
		{
			name:        "error envelope key",
			status:      http.StatusNotFound,
			body:        `{"error":"Not Found"}`,
			wantMessage: "Not Found",
			notFound:    true,
		},
		{
			name:   "non-json body still reports the status",
			status: http.StatusInternalServerError,
			body:   `<html>oops</html>`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			}))
			defer srv.Close()

			err := testConn(srv).Get(context.Background(), "/thing", nil, nil)
			if err == nil {
				t.Fatal("expected an error")
			}
			apiErr, ok := err.(*APIError)
			if !ok {
				t.Fatalf("expected *APIError, got %T", err)
			}
			if apiErr.StatusCode != tc.status {
				t.Errorf("expected status %d, got %d", tc.status, apiErr.StatusCode)
			}
			if apiErr.Message != tc.wantMessage {
				t.Errorf("expected message %q, got %q", tc.wantMessage, apiErr.Message)
			}
			if IsForbidden(err) != tc.forbidden {
				t.Errorf("IsForbidden: expected %v", tc.forbidden)
			}
			if IsNotFound(err) != tc.notFound {
				t.Errorf("IsNotFound: expected %v", tc.notFound)
			}
		})
	}
}

// A transport failure is not an APIError, so the degrade-to-null helpers must
// not treat it as an access-denied response.
func TestClassifiersRejectNonAPIErrors(t *testing.T) {
	err := fmt.Errorf("dial tcp: connection refused")
	if IsForbidden(err) {
		t.Error("IsForbidden must not match a transport error")
	}
	if IsNotFound(err) {
		t.Error("IsNotFound must not match a transport error")
	}
}
