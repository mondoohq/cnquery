// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	mondoovault "go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

type testRecord struct {
	ID string `json:"id"`
}

// newTestConn builds a connection pointed at a test server. It goes through the
// real constructor so the tests exercise the same option and credential
// handling a scan does.
func newTestConn(t *testing.T, baseURL string) *ConfluentConnection {
	t.Helper()
	conn, err := NewConfluentConnection(1, &inventory.Asset{}, &inventory.Config{
		Options: map[string]string{
			OptionAPIKey:  "cloud-key",
			OptionBaseURL: baseURL,
		},
		Credentials: []*mondoovault.Credential{
			{Type: mondoovault.CredentialType_password, Secret: []byte("cloud-secret")},
		},
	})
	require.NoError(t, err)
	return conn
}

func TestGetPagedWalksEveryPage(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page_token")

		var body map[string]any
		switch page {
		case "":
			body = map[string]any{
				"data":     []testRecord{{ID: "a"}, {ID: "b"}},
				"metadata": map[string]any{"next": srv.URL + "/things?page_token=2"},
			}
		case "2":
			body = map[string]any{
				"data":     []testRecord{{ID: "c"}},
				"metadata": map[string]any{"next": srv.URL + "/things?page_token=3"},
			}
		default:
			body = map[string]any{
				"data":     []testRecord{{ID: "d"}},
				"metadata": map[string]any{"next": nil},
			}
		}
		require.NoError(t, json.NewEncoder(w).Encode(body))
	}))
	defer srv.Close()

	conn := newTestConn(t, srv.URL)
	got, err := GetPaged[testRecord](context.Background(), conn, conn.CloudTarget(), "/things", nil)
	require.NoError(t, err)
	assert.Equal(t, []testRecord{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}}, got)
}

// An endpoint that ignores its cursor answers every request with the same page
// and hands the same URL back. Without the guard the walk would append the same
// records once per page up to the page cap, so a cluster with three ACLs would
// report fifteen hundred.
func TestGetPagedStopsOnStuckCursor(t *testing.T) {
	var calls atomic.Int64
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		// The cursor handed back is exactly the URL that was just fetched.
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"data":     []testRecord{{ID: "a"}},
			"metadata": map[string]any{"next": srv.URL + r.URL.RequestURI()},
		}))
	}))
	defer srv.Close()

	conn := newTestConn(t, srv.URL)
	got, err := GetPaged[testRecord](context.Background(), conn, conn.CloudTarget(), "/things", nil)
	require.NoError(t, err)
	assert.Equal(t, []testRecord{{ID: "a"}}, got, "the stuck page must be collected exactly once")
	assert.Equal(t, int64(1), calls.Load(), "the walk must not re-fetch a page it already saw")
}

// A cursor that revisits an earlier page, rather than the current one, is the
// same defect one step removed and must also terminate.
func TestGetPagedStopsOnCursorCycle(t *testing.T) {
	var calls atomic.Int64
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		next := srv.URL + "/things?page_size=100&page_token=2"
		if r.URL.Query().Get("page_token") == "2" {
			// point back at the first page
			next = srv.URL + "/things?page_size=100"
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"data":     []testRecord{{ID: r.URL.Query().Get("page_token")}},
			"metadata": map[string]any{"next": next},
		}))
	}))
	defer srv.Close()

	conn := newTestConn(t, srv.URL)
	got, err := GetPaged[testRecord](context.Background(), conn, conn.CloudTarget(), "/things", nil)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, int64(2), calls.Load())
}

// An endpoint that keeps producing fresh cursors forever is bounded by the page
// cap rather than looping until the scan is killed.
func TestGetPagedIsBoundedByThePageCap(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, _ := strconv.Atoi(r.URL.Query().Get("page_token"))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"data":     []testRecord{{ID: strconv.Itoa(token)}},
			"metadata": map[string]any{"next": srv.URL + "/things?page_token=" + strconv.Itoa(token+1)},
		}))
	}))
	defer srv.Close()

	conn := newTestConn(t, srv.URL)
	got, err := GetPaged[testRecord](context.Background(), conn, conn.CloudTarget(), "/things", nil)
	require.NoError(t, err)
	assert.Len(t, got, maxPages)
}

// A cursor naming a different host would send the credentials somewhere they do
// not belong, so it fails loudly rather than being followed or silently
// truncating the result.
func TestGetPagedRejectsForeignCursorHost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"data":     []testRecord{{ID: "a"}},
			"metadata": map[string]any{"next": "https://evil.example.com/things?page_token=2"},
		}))
	}))
	defer srv.Close()

	conn := newTestConn(t, srv.URL)
	_, err := GetPaged[testRecord](context.Background(), conn, conn.CloudTarget(), "/things", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "evil.example.com")
}

func TestGetPagedRejectsRelativeCursor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"data":     []testRecord{{ID: "a"}},
			"metadata": map[string]any{"next": "/things?page_token=2"},
		}))
	}))
	defer srv.Close()

	conn := newTestConn(t, srv.URL)
	_, err := GetPaged[testRecord](context.Background(), conn, conn.CloudTarget(), "/things", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "absolute URL")
}

func TestGetPagedHandlesAnEmptyListing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[],"metadata":{"next":null}}`))
	}))
	defer srv.Close()

	conn := newTestConn(t, srv.URL)
	got, err := GetPaged[testRecord](context.Background(), conn, conn.CloudTarget(), "/things", nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestGetPagedSendsCredentialsAndPageSize(t *testing.T) {
	type seen struct {
		user, pass, pageSize string
	}
	var got seen
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, _ := r.BasicAuth()
		got = seen{user: user, pass: pass, pageSize: r.URL.Query().Get("page_size")}
		_, _ = w.Write([]byte(`{"data":[],"metadata":{}}`))
	}))
	defer srv.Close()

	conn := newTestConn(t, srv.URL)
	_, err := GetPaged[testRecord](context.Background(), conn, conn.CloudTarget(), "/things", nil)
	require.NoError(t, err)
	assert.Equal(t, "cloud-key", got.user)
	assert.Equal(t, "cloud-secret", got.pass)
	assert.Equal(t, "100", got.pageSize)
}

// The per-cluster REST endpoints document no page_size parameter, so the walk
// must not add one there.
func TestGetPagedOmitsPageSizeOnKafkaEndpoints(t *testing.T) {
	var rawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		user, pass, _ := r.BasicAuth()
		assert.Equal(t, "kafka-key", user)
		assert.Equal(t, "kafka-secret", pass)
		_, _ = w.Write([]byte(`{"data":[],"metadata":{}}`))
	}))
	defer srv.Close()

	t.Setenv(EnvKafkaAPIKey, "kafka-key")
	t.Setenv(EnvKafkaAPISecret, "kafka-secret")

	conn := newTestConn(t, "https://api.confluent.cloud")
	target, err := conn.KafkaTarget("lkc-abc123", srv.URL)
	require.NoError(t, err)

	_, err = GetPaged[testRecord](context.Background(), conn, target, "/kafka/v3/clusters/lkc-abc123/topics", nil)
	require.NoError(t, err)
	assert.Empty(t, rawQuery)
}

func TestErrorEnvelopes(t *testing.T) {
	t.Run("management API errors array", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"errors":[{"code":"insufficient_scope","title":"Forbidden","detail":"the key may not read this"}]}`))
		}))
		defer srv.Close()

		conn := newTestConn(t, srv.URL)
		err := conn.Get(context.Background(), conn.CloudTarget(), "/things", nil, nil)
		require.Error(t, err)
		assert.True(t, IsForbidden(err))
		assert.Contains(t, err.Error(), "insufficient_scope")
		assert.Contains(t, err.Error(), "the key may not read this")
	})

	t.Run("management API errors array without detail falls back to title", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errors":[{"code":"not_found","title":"No such object"}]}`))
		}))
		defer srv.Close()

		conn := newTestConn(t, srv.URL)
		err := conn.Get(context.Background(), conn.CloudTarget(), "/things", nil, nil)
		require.Error(t, err)
		assert.True(t, IsNotFound(err))
		assert.Contains(t, err.Error(), "No such object")
	})

	t.Run("kafka REST error_code envelope", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error_code":40101,"message":"Unauthorized"}`))
		}))
		defer srv.Close()

		conn := newTestConn(t, srv.URL)
		err := conn.Get(context.Background(), conn.CloudTarget(), "/kafka/v3/clusters/x/acls", nil, nil)
		require.Error(t, err)
		assert.True(t, IsForbidden(err))
		assert.Contains(t, err.Error(), "40101")
	})

	t.Run("an unparseable body still carries the status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`<html>gateway</html>`))
		}))
		defer srv.Close()

		conn := newTestConn(t, srv.URL)
		err := conn.Get(context.Background(), conn.CloudTarget(), "/things", nil, nil)
		require.Error(t, err)
		assert.Equal(t, http.StatusBadGateway, StatusCode(err))
		assert.False(t, IsForbidden(err))
		assert.False(t, IsNotFound(err))
	})
}

func TestErrorClassifiers(t *testing.T) {
	statuses := map[int]struct{ forbidden, notFound bool }{
		http.StatusUnauthorized:        {forbidden: true},
		http.StatusForbidden:           {forbidden: true},
		http.StatusNotFound:            {notFound: true},
		http.StatusBadRequest:          {},
		http.StatusTooManyRequests:     {},
		http.StatusInternalServerError: {},
		http.StatusBadGateway:          {},
		http.StatusServiceUnavailable:  {},
	}
	for status, want := range statuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			err := &APIError{StatusCode: status, Path: "/things"}
			assert.Equal(t, want.forbidden, IsForbidden(err))
			assert.Equal(t, want.notFound, IsNotFound(err))

			// The classification must survive wrapping, since callers add
			// context before the answer is read.
			wrapped := fmt.Errorf("reading topics: %w", err)
			assert.Equal(t, want.forbidden, IsForbidden(wrapped))
			assert.Equal(t, want.notFound, IsNotFound(wrapped))
		})
	}

	t.Run("nil is not a definitive answer", func(t *testing.T) {
		assert.False(t, IsForbidden(nil))
		assert.False(t, IsNotFound(nil))
		assert.Equal(t, 0, StatusCode(nil))
	})
}

// A transport failure says nothing about what is on the other end. Reading it as
// "absent" or "forbidden" would let a network blip degrade a field to null and
// an audit pass on data that was never read.
func TestTransportFailureIsNotClassified(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := srv.URL
	srv.Close()

	conn := newTestConn(t, closedURL)
	_, err := GetPaged[testRecord](context.Background(), conn, conn.CloudTarget(), "/things", nil)
	require.Error(t, err)
	assert.False(t, IsForbidden(err), "a refused connection is not a permission answer")
	assert.False(t, IsNotFound(err), "a refused connection is not an absence answer")
	assert.Equal(t, 0, StatusCode(err))
}

func TestGetDecodeFailureIsReported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	conn := newTestConn(t, srv.URL)
	var out testRecord
	err := conn.Get(context.Background(), conn.CloudTarget(), "/things", nil, &out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}

func TestGetPassesQueryParameters(t *testing.T) {
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	conn := newTestConn(t, srv.URL)
	query := url.Values{}
	query.Set("environment", "env-abc123")
	require.NoError(t, conn.Get(context.Background(), conn.CloudTarget(), "/cmk/v2/clusters", query, nil))
	assert.Equal(t, "env-abc123", got.Get("environment"))
}
