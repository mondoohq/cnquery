// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadAgentSelf reads a captured agent self report and decodes it the way the
// resource does. The fixtures are the real payloads of a Consul 1.20.1 agent,
// one in its default configuration and one with ACLs, gossip encryption and
// internal RPC TLS verification switched on, so the decode is asserted against
// the shape the agent actually serves rather than against the documentation.
func loadAgentSelf(t *testing.T, name string) *agentSelf {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)

	var payload map[string]map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))

	self, err := decodeAgentSelf(payload)
	require.NoError(t, err)
	return self
}

func TestDecodeAgentSelfOpenAgent(t *testing.T) {
	self := loadAgentSelf(t, "agent-self-open.json")

	t.Run("summary block", func(t *testing.T) {
		assert.Equal(t, "dc1", self.Config.Datacenter)
		assert.Equal(t, "dc1", self.Config.PrimaryDatacenter)
		assert.Equal(t, "vm", self.Config.NodeName)
		assert.Equal(t, "bb0d4e5c-4177-a9be-a540-f01a3cc45ecc", self.Config.NodeID)
		assert.Equal(t, "1.20.1", self.Config.Version)
		assert.Equal(t, "920cc7c6", self.Config.Revision)
		assert.True(t, self.Config.Server)
	})

	// every security setting must read as off on the default agent, or the
	// hardened fixture proves nothing when the same fields read as on
	t.Run("nothing is switched on", func(t *testing.T) {
		assert.False(t, self.DebugConfig.aclEnabled())
		assert.Equal(t, "allow", self.DebugConfig.aclDefaultPolicy())
		assert.Equal(t, "extend-cache", self.DebugConfig.aclDownPolicy())
		assert.False(t, self.DebugConfig.ACLTokenReplication)
		assert.False(t, self.DebugConfig.ACLEnableKeyListPolicy)

		rpc := self.DebugConfig.internalRPCTLS()
		assert.False(t, rpc.VerifyIncoming)
		assert.False(t, rpc.VerifyOutgoing)
		assert.False(t, rpc.VerifyServerHostname)

		encrypted, present, err := self.serfEncrypted("serf_lan")
		require.NoError(t, err)
		assert.True(t, present)
		assert.False(t, encrypted)
	})

	t.Run("service mesh is on in dev mode", func(t *testing.T) {
		assert.True(t, self.DebugConfig.ConnectEnabled)
	})
}

func TestDecodeAgentSelfHardenedAgent(t *testing.T) {
	self := loadAgentSelf(t, "agent-self-hardened.json")

	t.Run("summary block", func(t *testing.T) {
		assert.Equal(t, "dc-secure", self.Config.Datacenter)
		assert.Equal(t, "secure-node", self.Config.NodeName)
		assert.True(t, self.Config.Server)
	})

	// the same fields the open fixture reads as off must read as on here; a
	// field reading the same in both would be a tag that never decoded
	t.Run("acl system", func(t *testing.T) {
		assert.True(t, self.DebugConfig.aclEnabled())
		assert.Equal(t, "deny", self.DebugConfig.aclDefaultPolicy())
		assert.Equal(t, "extend-cache", self.DebugConfig.aclDownPolicy())
	})

	t.Run("internal rpc tls", func(t *testing.T) {
		rpc := self.DebugConfig.internalRPCTLS()
		assert.True(t, rpc.VerifyIncoming)
		assert.True(t, rpc.VerifyOutgoing)
		assert.True(t, rpc.VerifyServerHostname)
		assert.Equal(t, "TLSv1_2", rpc.TLSMinVersion)
		assert.Equal(t, "/etc/consul.d/tls/consul-agent-ca.pem", rpc.CAFile)
	})

	// the three layers disagree on this agent, which is why they are separate
	// rows rather than one set of booleans
	t.Run("layers are read independently", func(t *testing.T) {
		scopes := self.DebugConfig.tlsScopes()
		require.Len(t, scopes, 3)

		byScope := map[string]tlsScopeConfig{}
		for _, scoped := range scopes {
			byScope[scoped.Scope] = scoped.Config
		}

		assert.True(t, byScope[tlsScopeInternalRPC].VerifyOutgoing)
		assert.True(t, byScope[tlsScopeHTTPS].VerifyOutgoing)
		// the gRPC layer did not inherit verify_outgoing from the defaults
		assert.False(t, byScope[tlsScopeGRPC].VerifyOutgoing)
		assert.False(t, byScope[tlsScopeHTTPS].VerifyIncoming)
	})

	t.Run("gossip encryption", func(t *testing.T) {
		lan, present, err := self.serfEncrypted("serf_lan")
		require.NoError(t, err)
		assert.True(t, present)
		assert.True(t, lan)

		wan, present, err := self.serfEncrypted("serf_wan")
		require.NoError(t, err)
		assert.True(t, present)
		assert.True(t, wan)
	})

	// the configured gossip key is redacted in the agent's own report whether
	// or not one is set, which is exactly why encryption is read from the serf
	// statistics instead
	t.Run("the gossip key is not a usable signal", func(t *testing.T) {
		open := loadAgentSelf(t, "agent-self-open.json")
		assert.Equal(t, "hidden", open.DebugConfig.EncryptKey)
		assert.Equal(t, "hidden", self.DebugConfig.EncryptKey)
	})
}

func TestDecodeAgentSelfRejectsNothing(t *testing.T) {
	_, err := decodeAgentSelf(nil)
	require.Error(t, err)
}

func TestSerfEncrypted(t *testing.T) {
	t.Run("absent pool is absent, not unencrypted", func(t *testing.T) {
		self := &agentSelf{Stats: map[string]map[string]any{
			"serf_lan": {"encrypted": "true"},
		}}
		encrypted, present, err := self.serfEncrypted("serf_wan")
		require.NoError(t, err)
		assert.False(t, present, "a client agent runs no WAN pool")
		assert.False(t, encrypted)
	})

	// the agent renders the flag as text, so a type assertion on a boolean
	// would always miss and report every pool as unencrypted
	t.Run("text is parsed", func(t *testing.T) {
		for _, tc := range []struct {
			raw  string
			want bool
		}{
			{"true", true},
			{"True", true},
			{"1", true},
			{"false", false},
			{"0", false},
		} {
			self := &agentSelf{Stats: map[string]map[string]any{
				"serf_lan": {"encrypted": tc.raw},
			}}
			encrypted, present, err := self.serfEncrypted("serf_lan")
			require.NoError(t, err, tc.raw)
			assert.True(t, present)
			assert.Equal(t, tc.want, encrypted, tc.raw)
		}
	})

	// an unreadable value must not degrade to "not encrypted", which would be
	// a claim about something that was never read
	t.Run("unreadable values are errors", func(t *testing.T) {
		for name, stats := range map[string]map[string]any{
			"missing key":  {"members": "1"},
			"not text":     {"encrypted": true},
			"not a bool":   {"encrypted": "maybe"},
			"empty string": {"encrypted": ""},
		} {
			t.Run(name, func(t *testing.T) {
				self := &agentSelf{Stats: map[string]map[string]any{"serf_lan": stats}}
				_, present, err := self.serfEncrypted("serf_lan")
				assert.True(t, present)
				require.Error(t, err)
			})
		}
	})
}

func TestAgentDebugConfigFallbacks(t *testing.T) {
	// Consul before 1.12 reported these at the top level and had no per-layer
	// TLS block. The fallback never runs against a modern agent, so it is
	// exercised here or it ships unverified.
	legacy := agentDebugConfig{
		ACLsEnabled:          true,
		ACLDefaultPolicy:     stringPtr("deny"),
		ACLDownPolicy:        stringPtr("async-cache"),
		VerifyIncoming:       boolPtr(true),
		VerifyOutgoing:       boolPtr(true),
		VerifyServerHostname: boolPtr(false),
	}

	t.Run("acl settings fall back to the top level", func(t *testing.T) {
		assert.True(t, legacy.aclEnabled())
		assert.Equal(t, "deny", legacy.aclDefaultPolicy())
		assert.Equal(t, "async-cache", legacy.aclDownPolicy())
	})

	t.Run("one internal rpc layer is synthesized", func(t *testing.T) {
		scopes := legacy.tlsScopes()
		require.Len(t, scopes, 1)
		assert.Equal(t, tlsScopeInternalRPC, scopes[0].Scope)
		assert.True(t, scopes[0].Config.VerifyIncoming)

		rpc := legacy.internalRPCTLS()
		assert.True(t, rpc.VerifyIncoming)
		assert.True(t, rpc.VerifyOutgoing)
		assert.False(t, rpc.VerifyServerHostname)
	})

	// the per-layer block wins whenever it is present, so a modern agent is
	// never read through the fallback
	t.Run("the per-layer block wins", func(t *testing.T) {
		modern := legacy
		modern.TLS = &tlsDebugConfig{
			InternalRPC: tlsScopeConfig{VerifyIncoming: false, VerifyServerHostname: true},
		}
		rpc := modern.internalRPCTLS()
		assert.False(t, rpc.VerifyIncoming)
		assert.True(t, rpc.VerifyServerHostname)
		assert.Len(t, modern.tlsScopes(), 3)
	})

	// the resolver block repeats the flag; either saying yes is taken as yes,
	// because the direction that hides an enabled ACL system is the one that
	// would make an unauthorized read look like a clean result
	t.Run("either acl flag counts", func(t *testing.T) {
		assert.True(t, (&agentDebugConfig{ACLsEnabled: true}).aclEnabled())
		assert.True(t, (&agentDebugConfig{
			ACLResolverSettings: aclResolverSettings{ACLsEnabled: true},
		}).aclEnabled())
		assert.False(t, (&agentDebugConfig{}).aclEnabled())
	})

	t.Run("nothing configured reports nothing", func(t *testing.T) {
		empty := agentDebugConfig{}
		assert.Equal(t, "", empty.aclDefaultPolicy())
		assert.Equal(t, "", empty.aclDownPolicy())
	})
}

func TestBoolValue(t *testing.T) {
	// an absent optional setting must decode to the reading that prompts a
	// look, never to the one that passes a check
	assert.False(t, boolValue(nil))
	assert.False(t, boolValue(boolPtr(false)))
	assert.True(t, boolValue(boolPtr(true)))
}

func TestBuildTime(t *testing.T) {
	t.Run("parsed", func(t *testing.T) {
		cfg := agentSelfConfig{BuildDate: "2024-10-29T19:04:05Z"}
		built := cfg.buildTime()
		require.NotNil(t, built)
		assert.Equal(t, time.Date(2024, 10, 29, 19, 4, 5, 0, time.UTC), built.UTC())
	})

	// an absent or unreadable timestamp stays null rather than becoming the
	// zero time, which would report a date in year one as a real build date
	t.Run("absent stays null", func(t *testing.T) {
		assert.Nil(t, (&agentSelfConfig{}).buildTime())
		assert.Nil(t, (&agentSelfConfig{BuildDate: "yesterday"}).buildTime())
	})

	t.Run("the real fixture parses", func(t *testing.T) {
		self := loadAgentSelf(t, "agent-self-open.json")
		built := self.Config.buildTime()
		require.NotNil(t, built)
		assert.Equal(t, 2024, built.UTC().Year())
	})
}

func boolPtr(v bool) *bool { return &v }

func stringPtr(v string) *string { return &v }
