// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphTargetIDs(t *testing.T) {
	conns := []*GraphConnection{
		{To: graphTarget{ID: "u1", Type: "user"}},
		{To: graphTarget{ID: "g1", Type: "user_group"}},
		{To: graphTarget{ID: "u2", Type: "user"}},
		{To: graphTarget{ID: "", Type: "user"}},   // empty id skipped
		{To: graphTarget{ID: "u1", Type: "user"}}, // duplicate skipped
		nil, // nil entry skipped
	}

	t.Run("no filter keeps every non-empty unique id in order", func(t *testing.T) {
		assert.Equal(t, []string{"u1", "g1", "u2"}, GraphTargetIDs(conns, ""))
	})

	t.Run("type filter narrows to matching targets", func(t *testing.T) {
		assert.Equal(t, []string{"u1", "u2"}, GraphTargetIDs(conns, "user"))
		assert.Equal(t, []string{"g1"}, GraphTargetIDs(conns, "user_group"))
	})

	t.Run("empty input yields empty result", func(t *testing.T) {
		assert.Empty(t, GraphTargetIDs(nil, ""))
	})
}

func TestUserMFAConfigured(t *testing.T) {
	tests := []struct {
		name string
		user *SystemUser
		want bool
	}{
		{name: "nil user", user: nil, want: false},
		{name: "totp enabled", user: &SystemUser{TotpEnabled: true}, want: true},
		{name: "mfa object configured", user: &SystemUser{MFA: &userMFA{Configured: true}}, want: true},
		{name: "mfa object not configured", user: &SystemUser{MFA: &userMFA{Configured: false}}, want: false},
		{name: "no mfa at all", user: &SystemUser{}, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, UserMFAConfigured(tc.user))
		})
	}
}

func TestParseTime(t *testing.T) {
	t.Run("valid RFC3339", func(t *testing.T) {
		got := ParseTime("2026-01-02T15:04:05Z")
		require.NotNil(t, got)
		assert.Equal(t, time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC), got.UTC())
	})
	t.Run("empty string is nil", func(t *testing.T) {
		assert.Nil(t, ParseTime(""))
	})
	t.Run("unparseable string is nil", func(t *testing.T) {
		assert.Nil(t, ParseTime("not-a-date"))
	})
}

func TestEffectiveID(t *testing.T) {
	t.Run("user prefers id over _id", func(t *testing.T) {
		assert.Equal(t, "id-1", (&SystemUser{ID: "id-1", XID: "x-1"}).EffectiveID())
		assert.Equal(t, "x-1", (&SystemUser{XID: "x-1"}).EffectiveID())
		assert.Equal(t, "", (&SystemUser{}).EffectiveID())
	})
	t.Run("system prefers id over _id", func(t *testing.T) {
		assert.Equal(t, "id-1", (&System{ID: "id-1", XID: "x-1"}).EffectiveID())
		assert.Equal(t, "x-1", (&System{XID: "x-1"}).EffectiveID())
	})
	t.Run("command prefers id over _id", func(t *testing.T) {
		assert.Equal(t, "id-1", (&Command{ID: "id-1", XID: "x-1"}).EffectiveID())
		assert.Equal(t, "x-1", (&Command{XID: "x-1"}).EffectiveID())
	})
}

func TestSystemPredicates(t *testing.T) {
	t.Run("fde active", func(t *testing.T) {
		assert.True(t, (&System{FDE: &systemFDE{Active: true}}).FdeActive())
		assert.False(t, (&System{FDE: &systemFDE{Active: false}}).FdeActive())
		assert.False(t, (&System{}).FdeActive()) // nil fde
	})
	t.Run("system insights enabled", func(t *testing.T) {
		assert.True(t, (&System{SystemInsights: &systemState{State: "enabled"}}).InsightsEnabled())
		assert.False(t, (&System{SystemInsights: &systemState{State: "disabled"}}).InsightsEnabled())
		assert.False(t, (&System{}).InsightsEnabled()) // nil insights
	})
}

func TestPolicyTemplateName(t *testing.T) {
	t.Run("prefers display name", func(t *testing.T) {
		p := &Policy{Template: &policyTemplate{Name: "n", DisplayName: "Display"}}
		assert.Equal(t, "Display", p.TemplateName())
	})
	t.Run("falls back to name", func(t *testing.T) {
		p := &Policy{Template: &policyTemplate{Name: "n"}}
		assert.Equal(t, "n", p.TemplateName())
	})
	t.Run("nil template", func(t *testing.T) {
		assert.Equal(t, "", (&Policy{}).TemplateName())
	})
}

// A misspelled struct tag decodes to the zero value, which for these fields is
// the safe-looking answer: a locked, expired, or suspended account would read
// as healthy and an audit would pass on it. Pin every security-relevant tag
// against a payload shaped like the documented response.
func TestSystemUserDecodePinsSecurityTags(t *testing.T) {
	raw := `{
		"_id": "x1",
		"id": "u1",
		"username": "alice",
		"email": "alice@example.com",
		"activated": true,
		"suspended": true,
		"account_locked": true,
		"password_expired": true,
		"passwordless_sudo": true,
		"sudo": true,
		"ldap_binding_user": true,
		"enable_managed_uid": true,
		"totp_enabled": true,
		"state": "ACTIVATED",
		"mfa": {"configured": true, "exclusion": true}
	}`

	var u SystemUser
	require.NoError(t, json.Unmarshal([]byte(raw), &u))

	assert.Equal(t, "u1", u.ID)
	assert.Equal(t, "x1", u.XID)
	assert.Equal(t, "alice", u.Username)
	assert.True(t, u.Activated)
	assert.True(t, u.Suspended)
	assert.True(t, u.AccountLocked)
	assert.True(t, u.PasswordExpired)
	assert.True(t, u.PasswordlessSudo)
	assert.True(t, u.Sudo)
	assert.True(t, u.LdapBindingUser)
	assert.True(t, u.EnableManagedUID)
	assert.True(t, u.TotpEnabled)
	assert.Equal(t, "ACTIVATED", u.State)
	require.NotNil(t, u.MFA)
	assert.True(t, u.MFA.Configured)
	assert.True(t, u.MFA.Exclusion)
}

// An absent field must decode to the safe reading rather than being mistaken
// for a value the API actually reported.
func TestSystemUserDecodeAbsentFields(t *testing.T) {
	var u SystemUser
	require.NoError(t, json.Unmarshal([]byte(`{"id":"u1"}`), &u))

	assert.False(t, u.Suspended)
	assert.False(t, u.AccountLocked)
	assert.False(t, u.PasswordExpired)
	assert.False(t, u.TotpEnabled)
	assert.Nil(t, u.MFA, "absent mfa must stay nil, not an empty configuration")
}

// The SSH toggles decide whether a host accepts root logins and passwords, so a
// wrong tag here reports a permissive host as hardened.
func TestSystemDecodePinsSecurityTags(t *testing.T) {
	raw := `{
		"_id": "x1",
		"id": "s1",
		"hostname": "host-1",
		"displayName": "Host One",
		"os": "Ubuntu",
		"version": "24.04",
		"agentVersion": "1.2.3",
		"arch": "x86_64",
		"active": true,
		"allowSshRootLogin": true,
		"allowSshPasswordAuthentication": true,
		"allowMultiFactorAuthentication": true,
		"allowPublicKeyAuthentication": true,
		"hasServiceAccount": true,
		"remoteIP": "203.0.113.10",
		"fde": {"active": true},
		"systemInsights": {"state": "enabled"}
	}`

	var s System
	require.NoError(t, json.Unmarshal([]byte(raw), &s))

	assert.Equal(t, "s1", s.ID)
	assert.Equal(t, "x1", s.XID)
	assert.Equal(t, "host-1", s.Hostname)
	assert.Equal(t, "Host One", s.DisplayName)
	assert.Equal(t, "x86_64", s.Arch)
	assert.True(t, s.Active)
	assert.True(t, s.AllowSshRootLogin)
	assert.True(t, s.AllowSshPasswordAuthentication)
	assert.True(t, s.AllowMultiFactorAuthentication)
	assert.True(t, s.AllowPublicKeyAuthentication)
	assert.True(t, s.HasServiceAccount)
	assert.Equal(t, "203.0.113.10", s.RemoteIP)
	require.NotNil(t, s.FDE)
	assert.True(t, s.FDE.Active)
	assert.True(t, s.FdeActive())
	assert.True(t, s.InsightsEnabled())
}

func TestSystemDecodeAbsentFields(t *testing.T) {
	var s System
	require.NoError(t, json.Unmarshal([]byte(`{"id":"s1"}`), &s))

	assert.False(t, s.AllowSshRootLogin)
	assert.False(t, s.AllowSshPasswordAuthentication)
	assert.Nil(t, s.FDE)
	assert.False(t, s.FdeActive(), "a host with no fde object is not encrypted")
	assert.False(t, s.InsightsEnabled())
}

// ssoUrl is a top-level string on the application document (the `sso` object
// that sits beside it carries the connector's type and flags, not the URL), so
// the tag is pinned here against that shape.
func TestApplicationDecodePinsSsoURL(t *testing.T) {
	raw := `{
		"id": "a1",
		"name": "app-1",
		"displayName": "App One",
		"ssoUrl": "https://sso.jumpcloud.com/saml2/app-1",
		"active": true,
		"sso": {"type": "saml", "jit": true}
	}`

	var a Application
	require.NoError(t, json.Unmarshal([]byte(raw), &a))

	assert.Equal(t, "a1", a.ID)
	assert.Equal(t, "app-1", a.Name)
	assert.Equal(t, "App One", a.DisplayName)
	assert.Equal(t, "https://sso.jumpcloud.com/saml2/app-1", a.SsoURL)
	assert.True(t, a.Active)
}
