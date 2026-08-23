// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

func testConnection(t *testing.T, host string, opts ...func(*inventory.Config)) *OllamaConnection {
	t.Helper()
	conf := &inventory.Config{Options: map[string]string{HostOption: host}}
	for _, opt := range opts {
		opt(conf)
	}
	conn, err := NewOllamaConnection(1, &inventory.Asset{}, conf)
	require.NoError(t, err)
	return conn
}

func TestTLS(t *testing.T) {
	assert.False(t, testConnection(t, "http://10.0.0.5:11434").TLS())
	assert.True(t, testConnection(t, "https://ollama.example.com").TLS())
	assert.True(t, testConnection(t, "HTTPS://ollama.example.com").TLS(), "the scheme is compared case-insensitively")
}

func TestIsLocal(t *testing.T) {
	local := []string{
		"http://localhost:11434",
		"http://LOCALHOST:11434",
		"http://127.0.0.1:11434",
		"http://127.0.0.53:11434",
		"http://[::1]:11434",
		"http://ollama.localhost:11434",
	}
	for _, host := range local {
		assert.True(t, testConnection(t, host).IsLocal(), host)
	}

	remote := []string{
		"http://0.0.0.0:11434",
		"http://10.0.0.5:11434",
		"https://ollama.example.com",
		"http://[2001:db8::1]:11434",
	}
	for _, host := range remote {
		assert.False(t, testConnection(t, host).IsLocal(), host)
	}
}

func TestAnonymousStatusSendsNoToken(t *testing.T) {
	var gotAuth string
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	conn := testConnection(t, srv.URL, func(conf *inventory.Config) {
		conf.Options[TokenOption] = "super-secret-token"
	})

	code, err := conn.AnonymousStatus(t.Context())
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, code)
	assert.Empty(t, gotAuth, "the probe must not carry the configured API token")
	assert.Equal(t, "/api/tags", gotPath, "the probe must use a read-only endpoint")
}

func TestAnonymousStatusUnauthenticatedInstance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	code, err := testConnection(t, srv.URL).AnonymousStatus(t.Context())
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
}

func TestVersionIsFetchedOnce(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"version":"0.12.6"}`))
	}))
	defer srv.Close()

	conn := testConnection(t, srv.URL)
	for range 3 {
		version, err := conn.Version(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "0.12.6", version)
	}
	assert.Equal(t, 1, calls, "asset detection and the version field share a single call")
}

func TestAnonymousWriteStatusCannotMutate(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotAuth   string
		gotBody   []byte
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	conn := testConnection(t, srv.URL, func(c *inventory.Config) {
		c.Options[TokenOption] = "a-token-the-probe-must-not-send"
	})

	code, err := conn.AnonymousWriteStatus(t.Context())
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, code)

	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/api/create", gotPath,
		"the probe must go to the creation endpoint, never to delete or pull")
	assert.Empty(t, gotAuth, "the probe must be unauthenticated or it proves nothing")

	// The body is what makes the probe safe: it cannot be decoded, so no model
	// name, file, or digest can be read out of it.
	var decoded any
	assert.Error(t, json.Unmarshal(gotBody, &decoded),
		"the probe body must not be valid JSON, or the server could act on it")
	assert.NotContains(t, string(gotBody), "model",
		"the probe body must not name a model under any key")
}

func TestAnonymousWriteStatusGatedInstance(t *testing.T) {
	var reached bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	code, err := testConnection(t, srv.URL).AnonymousWriteStatus(t.Context())
	require.NoError(t, err)
	assert.True(t, reached)
	assert.Equal(t, http.StatusUnauthorized, code)
}

func TestAnonymousWriteStatusUnreachableHost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	_, err := testConnection(t, url).AnonymousWriteStatus(t.Context())
	assert.Error(t, err, "a transport failure must surface, never read as writes being open")
}
