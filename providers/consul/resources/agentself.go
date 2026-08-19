// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"
)

// agentSelf is the part of the agent's self report this provider reads. The
// endpoint is served untyped, so it is decoded into these structs rather than
// walked as maps: a wrong key then shows up as a zero value in one field
// instead of a panic somewhere downstream.
type agentSelf struct {
	Config      agentSelfConfig  `json:"Config"`
	DebugConfig agentDebugConfig `json:"DebugConfig"`
	// Stats is left untyped per value because the agent renders every
	// statistic as text and a future release adding a nested object must not
	// fail the decode of the whole block.
	Stats map[string]map[string]any `json:"Stats"`
}

// agentSelfConfig is the small, stable summary block of the self report.
type agentSelfConfig struct {
	Datacenter        string `json:"Datacenter"`
	PrimaryDatacenter string `json:"PrimaryDatacenter"`
	NodeName          string `json:"NodeName"`
	NodeID            string `json:"NodeID"`
	Revision          string `json:"Revision"`
	Server            bool   `json:"Server"`
	Version           string `json:"Version"`
	BuildDate         string `json:"BuildDate"`
}

// agentDebugConfig is the agent's fully resolved runtime configuration, which
// is where the security settings live after defaults and overrides are applied.
type agentDebugConfig struct {
	ACLsEnabled            bool                `json:"ACLsEnabled"`
	ACLEnableKeyListPolicy bool                `json:"ACLEnableKeyListPolicy"`
	ACLTokenReplication    bool                `json:"ACLTokenReplication"`
	ACLResolverSettings    aclResolverSettings `json:"ACLResolverSettings"`
	ConnectEnabled         bool                `json:"ConnectEnabled"`
	AutoEncryptTLS         bool                `json:"AutoEncryptTLS"`
	AutoEncryptAllowTLS    bool                `json:"AutoEncryptAllowTLS"`

	// EncryptKey is the configured gossip key. The agent redacts it to
	// "hidden" whether or not one is set, so it says nothing about whether the
	// gossip pool is encrypted and no resource exposes it. It is decoded only
	// so a test can pin that behaviour, which is what forces the encryption
	// state to be read from the serf statistics instead.
	EncryptKey string `json:"EncryptKey"`

	// TLS holds the per-layer settings from Consul 1.12 onwards. It is a
	// pointer so an older agent, which has no such block, is distinguishable
	// from one whose every layer happens to verify nothing.
	TLS *tlsDebugConfig `json:"TLS"`

	// Consul before 1.12 reported the verification settings at the top level
	// and had no per-layer block. They are pointers for the same reason: an
	// absent setting must not be confused with one explicitly set to false.
	VerifyIncoming       *bool `json:"VerifyIncoming"`
	VerifyOutgoing       *bool `json:"VerifyOutgoing"`
	VerifyServerHostname *bool `json:"VerifyServerHostname"`

	// Consul before 1.12 also reported the ACL defaults at the top level
	// rather than inside the resolver settings.
	ACLDefaultPolicy *string `json:"ACLDefaultPolicy"`
	ACLDownPolicy    *string `json:"ACLDownPolicy"`
}

type aclResolverSettings struct {
	ACLDefaultPolicy string `json:"ACLDefaultPolicy"`
	ACLDownPolicy    string `json:"ACLDownPolicy"`
	ACLsEnabled      bool   `json:"ACLsEnabled"`
}

type tlsDebugConfig struct {
	InternalRPC tlsScopeConfig `json:"InternalRPC"`
	HTTPS       tlsScopeConfig `json:"HTTPS"`
	GRPC        tlsScopeConfig `json:"GRPC"`
}

type tlsScopeConfig struct {
	CAFile               string   `json:"CAFile"`
	CAPath               string   `json:"CAPath"`
	CertFile             string   `json:"CertFile"`
	CipherSuites         []string `json:"CipherSuites"`
	TLSMinVersion        string   `json:"TLSMinVersion"`
	UseAutoCert          bool     `json:"UseAutoCert"`
	VerifyIncoming       bool     `json:"VerifyIncoming"`
	VerifyOutgoing       bool     `json:"VerifyOutgoing"`
	VerifyServerHostname bool     `json:"VerifyServerHostname"`
}

// TLS scope names, used as the selection key on consul.tlsProfile.
const (
	tlsScopeInternalRPC = "internalRpc"
	tlsScopeHTTPS       = "https"
	tlsScopeGRPC        = "grpc"
)

// decodeAgentSelf turns the untyped self report into the typed view above. The
// client hands back nested maps, so the payload is re-encoded and decoded
// rather than walked, which keeps the field names in one place.
func decodeAgentSelf(raw map[string]map[string]any) (*agentSelf, error) {
	if raw == nil {
		return nil, errors.New("the Consul agent returned no self report")
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var self agentSelf
	if err := json.Unmarshal(encoded, &self); err != nil {
		return nil, err
	}
	return &self, nil
}

// aclEnabled reports whether the ACL system is switched on. Consul 1.12
// onwards repeats the flag inside the resolver settings, and either saying yes
// is taken as yes, because the direction that hides an enabled ACL system is
// the one that would make an unauthorized read look like a clean result.
func (c *agentDebugConfig) aclEnabled() bool {
	return c.ACLsEnabled || c.ACLResolverSettings.ACLsEnabled
}

// aclDefaultPolicy reports what the agent does with an unmatched request. The
// resolver settings are preferred, with the top-level field as the fallback for
// agents predating that block.
func (c *agentDebugConfig) aclDefaultPolicy() string {
	if c.ACLResolverSettings.ACLDefaultPolicy != "" {
		return c.ACLResolverSettings.ACLDefaultPolicy
	}
	if c.ACLDefaultPolicy != nil {
		return *c.ACLDefaultPolicy
	}
	return ""
}

// aclDownPolicy reports what the agent does when the ACL authority is
// unreachable, with the same fallback as aclDefaultPolicy.
func (c *agentDebugConfig) aclDownPolicy() string {
	if c.ACLResolverSettings.ACLDownPolicy != "" {
		return c.ACLResolverSettings.ACLDownPolicy
	}
	if c.ACLDownPolicy != nil {
		return *c.ACLDownPolicy
	}
	return ""
}

// tlsScopes returns the per-layer TLS settings in a stable order. An agent
// predating the per-layer block reports one layer, internalRpc, carrying the
// top-level settings it does have, because those governed the RPC channel.
func (c *agentDebugConfig) tlsScopes() []struct {
	Scope  string
	Config tlsScopeConfig
} {
	type scoped = struct {
		Scope  string
		Config tlsScopeConfig
	}

	if c.TLS != nil {
		return []scoped{
			{tlsScopeInternalRPC, c.TLS.InternalRPC},
			{tlsScopeHTTPS, c.TLS.HTTPS},
			{tlsScopeGRPC, c.TLS.GRPC},
		}
	}

	return []scoped{{tlsScopeInternalRPC, tlsScopeConfig{
		VerifyIncoming:       boolValue(c.VerifyIncoming),
		VerifyOutgoing:       boolValue(c.VerifyOutgoing),
		VerifyServerHostname: boolValue(c.VerifyServerHostname),
	}}}
}

// internalRPCTLS returns the settings governing the agent-to-agent RPC
// channel, which is what the classic verify_incoming, verify_outgoing and
// verify_server_hostname options configure.
func (c *agentDebugConfig) internalRPCTLS() tlsScopeConfig {
	if c.TLS != nil {
		return c.TLS.InternalRPC
	}
	return tlsScopeConfig{
		VerifyIncoming:       boolValue(c.VerifyIncoming),
		VerifyOutgoing:       boolValue(c.VerifyOutgoing),
		VerifyServerHostname: boolValue(c.VerifyServerHostname),
	}
}

// boolValue reads an optional boolean. An absent setting is reported as false,
// which for every setting it is used on is the reading that prompts a look
// rather than the one that passes a check.
func boolValue(value *bool) bool {
	return value != nil && *value
}

// serfEncrypted reports whether one gossip pool is encrypted. The agent renders
// the statistic as text, so a plain type assertion on a boolean would always
// miss and report every pool as unencrypted.
//
// The second return value says whether the pool exists at all. A client agent
// runs no WAN pool, and that absence is not the same answer as an unencrypted
// one, so the caller reports it as null rather than as false.
func (s *agentSelf) serfEncrypted(pool string) (encrypted bool, present bool, err error) {
	stats, ok := s.Stats[pool]
	if !ok {
		return false, false, nil
	}
	raw, ok := stats["encrypted"]
	if !ok {
		return false, true, errors.New("the Consul agent reported no encryption state for the " + pool + " gossip pool")
	}
	text, ok := raw.(string)
	if !ok {
		return false, true, errors.New("the Consul agent reported a non-textual encryption state for the " + pool + " gossip pool")
	}
	value, parseErr := strconv.ParseBool(text)
	if parseErr != nil {
		return false, true, errors.New("cannot read the encryption state of the " + pool + " gossip pool: " + text)
	}
	return value, true, nil
}

// buildTime parses the agent's build timestamp. A build that reports none, or
// one whose timestamp cannot be read, is reported as null rather than as the
// zero time, which would render as a date in year one.
func (c *agentSelfConfig) buildTime() *time.Time {
	if c.BuildDate == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, c.BuildDate)
	if err != nil {
		return nil
	}
	return &parsed
}
