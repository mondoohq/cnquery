// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
	"go.mondoo.com/mql/v13/providers/azure/connection"
)

// connOpts returns the Options map from the single connection ParseCLI builds,
// failing the test if the response shape is unexpected.
func connOpts(t *testing.T, res *plugin.ParseCLIRes) map[string]string {
	t.Helper()
	require.NotNil(t, res)
	require.NotNil(t, res.Asset)
	require.Len(t, res.Asset.Connections, 1)
	return res.Asset.Connections[0].Options
}

// TestParseCLINilFlags is a regression test: the flags map only contains keys
// the CLI actually registered, so absent keys (notably the legacy singular
// "subscription", which is never registered) resolve to a nil *llx.Primitive.
// ParseCLI used to dereference those unconditionally and panic before reaching
// any resource. It must now tolerate missing and explicitly-nil flag values.
func TestParseCLINilFlags(t *testing.T) {
	s := Init()

	t.Run("empty flags map does not panic", func(t *testing.T) {
		var res *plugin.ParseCLIRes
		var err error
		require.NotPanics(t, func() {
			res, err = s.ParseCLI(&plugin.ParseCLIReq{
				Flags: map[string]*llx.Primitive{},
			})
		})
		require.NoError(t, err)
		opts := connOpts(t, res)
		// unset auth flags map to empty strings, not a crash
		assert.Empty(t, opts["tenant-id"])
		assert.Empty(t, opts["client-id"])
		// no subscription of either spelling was provided
		assert.NotContains(t, opts, "subscriptions")
	})

	t.Run("explicitly nil flag value does not panic", func(t *testing.T) {
		require.NotPanics(t, func() {
			_, err := s.ParseCLI(&plugin.ParseCLIReq{
				Flags: map[string]*llx.Primitive{
					"tenant-id":     nil,
					"subscriptions": nil,
				},
			})
			require.NoError(t, err)
		})
	})

	t.Run("nil flags map does not panic", func(t *testing.T) {
		require.NotPanics(t, func() {
			_, err := s.ParseCLI(&plugin.ParseCLIReq{Flags: nil})
			require.NoError(t, err)
		})
	})
}

// TestParseCLISubscriptionFlags checks that both the plural "subscriptions"
// flag and the legacy singular "subscription" key populate the connection's
// "subscriptions" option, and that "subscriptions-exclude" flows through.
func TestParseCLISubscriptionFlags(t *testing.T) {
	s := Init()

	t.Run("plural subscriptions flag", func(t *testing.T) {
		res, err := s.ParseCLI(&plugin.ParseCLIReq{
			Flags: map[string]*llx.Primitive{
				"subscriptions": llx.StringPrimitive("sub-a,sub-b"),
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "sub-a,sub-b", connOpts(t, res)["subscriptions"])
	})

	t.Run("singular subscription flag still honored", func(t *testing.T) {
		res, err := s.ParseCLI(&plugin.ParseCLIReq{
			Flags: map[string]*llx.Primitive{
				"subscription": llx.StringPrimitive("sub-single"),
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "sub-single", connOpts(t, res)["subscriptions"])
	})

	t.Run("subscriptions-exclude flows through", func(t *testing.T) {
		res, err := s.ParseCLI(&plugin.ParseCLIReq{
			Flags: map[string]*llx.Primitive{
				"subscriptions-exclude": llx.StringPrimitive("sub-x"),
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "sub-x", connOpts(t, res)["subscriptions-exclude"])
	})
}

// TestParseCLICredentials verifies auth flags produce the expected credential
// and option shape without tripping any nil dereference.
func TestParseCLICredentials(t *testing.T) {
	s := Init()

	t.Run("service principal with client secret", func(t *testing.T) {
		res, err := s.ParseCLI(&plugin.ParseCLIReq{
			Flags: map[string]*llx.Primitive{
				"tenant-id":     llx.StringPrimitive("tenant-1"),
				"client-id":     llx.StringPrimitive("client-1"),
				"client-secret": llx.StringPrimitive("secret-1"),
				"subscriptions": llx.StringPrimitive("sub-a"),
			},
		})
		require.NoError(t, err)
		opts := connOpts(t, res)
		assert.Equal(t, "tenant-1", opts["tenant-id"])
		assert.Equal(t, "client-1", opts["client-id"])

		creds := res.Asset.Connections[0].Credentials
		require.Len(t, creds, 1)
		assert.Equal(t, vault.CredentialType_password, creds[0].Type)
		assert.Equal(t, []byte("secret-1"), creds[0].Secret)
	})

	t.Run("federated token file sets its option", func(t *testing.T) {
		res, err := s.ParseCLI(&plugin.ParseCLIReq{
			Flags: map[string]*llx.Primitive{
				"federated-token-file": llx.StringPrimitive("/var/run/token"),
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "/var/run/token", connOpts(t, res)[connection.OptionFederatedTokenFile])
	})

	t.Run("no auth flags yields no credentials", func(t *testing.T) {
		res, err := s.ParseCLI(&plugin.ParseCLIReq{
			Flags: map[string]*llx.Primitive{},
		})
		require.NoError(t, err)
		assert.Empty(t, res.Asset.Connections[0].Credentials)
	})
}
