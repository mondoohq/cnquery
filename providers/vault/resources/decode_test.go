// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsAutoUnseal(t *testing.T) {
	tests := []struct {
		sealType string
		want     bool
	}{
		{"shamir", false},
		{"awskms", true},
		{"azurekeyvault", true},
		{"gcpckms", true},
		{"transit", true},
		{"pkcs11", true},
		// an unknown seal type must not be reported as Shamir, or a server that
		// unseals on its own would look like one needing manual key shares
		{"somethingnew", true},
		// an absent type is reported as Shamir, the answer that prompts a look
		{"", false},
	}
	for _, tc := range tests {
		t.Run(tc.sealType, func(t *testing.T) {
			assert.Equal(t, tc.want, isAutoUnseal(tc.sealType))
		})
	}
}

func TestOptionBool(t *testing.T) {
	// Vault renders audit device options as strings, so a plain type assertion
	// on a bool would always miss and report log_raw as off
	assert.True(t, optionBool(map[string]string{"log_raw": "true"}, "log_raw"))
	assert.True(t, optionBool(map[string]string{"log_raw": "1"}, "log_raw"))
	assert.False(t, optionBool(map[string]string{"log_raw": "false"}, "log_raw"))
	assert.False(t, optionBool(map[string]string{}, "log_raw"), "absent option is off")
	assert.False(t, optionBool(nil, "log_raw"), "nil options is off")
	assert.False(t, optionBool(map[string]string{"log_raw": "maybe"}, "log_raw"), "garbage is off")
}

func TestKvVersion(t *testing.T) {
	tests := []struct {
		name  string
		mount *vaultapi.MountOutput
		want  int64
	}{
		{"kv v2", &vaultapi.MountOutput{Type: "kv", Options: map[string]string{"version": "2"}}, 2},
		{"kv v1 explicit", &vaultapi.MountOutput{Type: "kv", Options: map[string]string{"version": "1"}}, 1},
		// a v1 mount commonly omits the option entirely
		{"kv with no option is v1", &vaultapi.MountOutput{Type: "kv"}, 1},
		{"kv with empty option is v1", &vaultapi.MountOutput{Type: "kv", Options: map[string]string{"version": ""}}, 1},
		{"kv with garbage option is v1", &vaultapi.MountOutput{Type: "kv", Options: map[string]string{"version": "two"}}, 1},
		// a non-kv mount has no key/value version at all
		{"pki reports zero", &vaultapi.MountOutput{Type: "pki", Options: map[string]string{"version": "2"}}, 0},
		{"nil mount reports zero", nil, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, kvVersion(tc.mount))
		})
	}
}

func TestStringMap(t *testing.T) {
	assert.Equal(t, map[string]string{"a": "1"}, stringMap(map[string]any{"a": "1"}))
	// non-string values are dropped rather than coerced
	assert.Equal(t, map[string]string{"a": "1"}, stringMap(map[string]any{"a": "1", "b": 2}))
	assert.Nil(t, stringMap("not an object"))
	assert.Nil(t, stringMap(nil))
}

// newTestClient points a real Vault client at a test server, so the error
// classifier is exercised against errors the client actually produces rather
// than ones hand-built in the test.
func newTestClient(t *testing.T, status int, body string) *vaultapi.Client {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	cfg := vaultapi.DefaultConfig()
	cfg.Address = srv.URL
	// the classifier must see the server's status, not a retry giving up
	cfg.MaxRetries = 0
	client, err := vaultapi.NewClient(cfg)
	require.NoError(t, err)
	client.SetToken("test-token")
	return client
}

func TestIsFeatureAbsent(t *testing.T) {
	t.Run("nil is not a definitive answer", func(t *testing.T) {
		assert.False(t, isFeatureAbsent(nil))
	})

	// a transport failure must never be read as "the feature is absent", or a
	// network blip would silently report an empty namespace list
	t.Run("transport error is not a definitive answer", func(t *testing.T) {
		assert.False(t, isFeatureAbsent(errors.New("dial tcp: connection refused")))
	})

	// status mapping is asserted against the error type directly, because the
	// client rewrites some statuses before the caller sees them: a LIST that
	// draws a 405 is retried as a GET, so it never reaches us as a 405
	t.Run("status mapping", func(t *testing.T) {
		tests := map[int]bool{
			http.StatusNotFound:         true,
			http.StatusMethodNotAllowed: true,
			// a token that may not read the endpoint tells us nothing about
			// what is behind it, so "none" would turn a missing permission
			// into a clean audit pass
			http.StatusForbidden:           false,
			http.StatusUnauthorized:        false,
			http.StatusInternalServerError: false,
			http.StatusBadGateway:          false,
			http.StatusTooManyRequests:     false,
		}
		for status, want := range tests {
			t.Run(http.StatusText(status), func(t *testing.T) {
				err := &vaultapi.ResponseError{StatusCode: status, Errors: []string{"boom"}}
				assert.Equal(t, want, isFeatureAbsent(err))
				// it must also survive being wrapped
				assert.Equal(t, want, isFeatureAbsent(fmt.Errorf("reading namespaces: %w", err)))
			})
		}
	})

	// prove the classifier matches an error a real client produces, not only a
	// hand-built one
	t.Run("matches a real client error", func(t *testing.T) {
		client := newTestClient(t, http.StatusForbidden, `{"errors":["permission denied"]}`)
		_, err := client.Logical().Read("sys/namespaces")
		require.Error(t, err)
		assert.False(t, isFeatureAbsent(err), "permission denied is not absence")
	})
}
