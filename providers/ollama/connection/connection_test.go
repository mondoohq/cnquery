// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
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
