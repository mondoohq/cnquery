// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
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
