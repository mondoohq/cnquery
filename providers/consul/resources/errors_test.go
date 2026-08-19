// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient points a real Consul client at a test server, so the
// classifiers are exercised against errors the client actually produces rather
// than ones hand-built in the test.
func newTestClient(t *testing.T, status int, body string) *consulapi.Client {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	cfg := consulapi.DefaultConfig()
	cfg.Address = srv.URL
	cfg.Token = ""
	client, err := consulapi.NewClient(cfg)
	require.NoError(t, err)
	return client
}

func TestIsACLSystemDisabled(t *testing.T) {
	t.Run("nil is not a definitive answer", func(t *testing.T) {
		assert.False(t, isACLSystemDisabled(nil))
	})

	// a transport failure must never be read as "the ACL system is off", or a
	// network blip would silently report an empty token inventory and every
	// check written against it would pass
	t.Run("transport error is not a definitive answer", func(t *testing.T) {
		assert.False(t, isACLSystemDisabled(errors.New("dial tcp: connection refused")))
		assert.False(t, isACLSystemDisabled(errors.New("Unexpected response code: 401 (ACL support disabled)")),
			"a plain error carrying the same text must not match")
	})

	t.Run("status and body mapping", func(t *testing.T) {
		tests := []struct {
			name string
			code int
			body string
			want bool
		}{
			// the one answer that may become an empty inventory
			{"acls switched off", http.StatusUnauthorized, aclDisabledMessage, true},
			// an unrecognized token: says nothing about what is behind the
			// endpoint, so it must surface as an error
			{"unknown token", http.StatusForbidden, "ACL not found", false},
			// a token lacking acl:read: reporting "none" would turn a missing
			// permission into a clean audit pass
			{"permission denied", http.StatusForbidden, "Permission denied: token with AccessorID 'x' lacks permission 'acl:read'", false},
			// a 401 from something other than the ACL subsystem must fail
			// loudly rather than read as an absent feature
			{"unrelated 401", http.StatusUnauthorized, "Proxy Authentication Required", false},
			{"server error", http.StatusInternalServerError, "boom", false},
			{"rate limited", http.StatusTooManyRequests, "slow down", false},
			{"not found", http.StatusNotFound, "nope", false},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				err := consulapi.StatusError{Code: tc.code, Body: tc.body}
				assert.Equal(t, tc.want, isACLSystemDisabled(err))
				// it must also survive being wrapped
				assert.Equal(t, tc.want, isACLSystemDisabled(fmt.Errorf("listing tokens: %w", err)))
			})
		}
	})

	// prove the classifiers match errors a real client produces, not only
	// hand-built ones
	t.Run("matches a real client error", func(t *testing.T) {
		disabled := newTestClient(t, http.StatusUnauthorized, aclDisabledMessage)
		_, _, err := disabled.ACL().TokenList(nil)
		require.Error(t, err)
		assert.True(t, isACLSystemDisabled(err))

		denied := newTestClient(t, http.StatusForbidden, "Permission denied: lacks permission 'acl:read'")
		_, _, err = denied.ACL().TokenList(nil)
		require.Error(t, err)
		assert.False(t, isACLSystemDisabled(err), "permission denied is not an absent feature")
	})
}

func TestIsNotFound(t *testing.T) {
	assert.False(t, isNotFound(nil))
	assert.False(t, isNotFound(errors.New("dial tcp: connection refused")))
	assert.True(t, isNotFound(consulapi.StatusError{Code: http.StatusNotFound, Body: "nope"}))
	assert.True(t, isNotFound(fmt.Errorf("reading policy: %w",
		consulapi.StatusError{Code: http.StatusNotFound})))
	// a permission failure is not an absence
	assert.False(t, isNotFound(consulapi.StatusError{Code: http.StatusForbidden}))
	assert.False(t, isNotFound(consulapi.StatusError{Code: http.StatusUnauthorized}))
}
