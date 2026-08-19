// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultIntentionPolicy(t *testing.T) {
	tests := []struct {
		name          string
		aclEnabled    bool
		defaultPolicy string
		want          string
	}{
		// this is the pairing that makes intentions an allowlist
		{"acls on and denying", true, "deny", intentionPolicyDeny},
		{"acls on but allowing", true, "allow", intentionPolicyAllow},
		// with ACLs off, every service in the mesh reaches every other one no
		// matter what the configured default policy says
		{"acls off", false, "deny", intentionPolicyAllow},
		{"nothing configured", false, "", intentionPolicyAllow},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := defaultIntentionPolicy(tc.aclEnabled, tc.defaultPolicy)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.want == intentionPolicyDeny, got == intentionPolicyDeny)
		})
	}
}

// An intention has no identifier of its own, so its cache key is built from the
// dimensions along which one pair of service names legitimately repeats. A
// missed dimension does not merely mislabel: the second intention would return
// the cached first one's action, and a deny would report as an allow.
func TestIntentionIDCarriesEveryDimension(t *testing.T) {
	base := &consulapi.Intention{
		SourceName:      "web",
		SourceNS:        "default",
		DestinationName: "db",
		DestinationNS:   "default",
	}

	variants := map[string]func(*consulapi.Intention){
		"source namespace":      func(i *consulapi.Intention) { i.SourceNS = "team-a" },
		"source partition":      func(i *consulapi.Intention) { i.SourcePartition = "alpha" },
		"source peer":           func(i *consulapi.Intention) { i.SourcePeer = "cluster-b" },
		"source name":           func(i *consulapi.Intention) { i.SourceName = "api" },
		"destination namespace": func(i *consulapi.Intention) { i.DestinationNS = "team-b" },
		"destination partition": func(i *consulapi.Intention) { i.DestinationPartition = "beta" },
		"destination name":      func(i *consulapi.Intention) { i.DestinationName = "cache" },
	}

	seen := map[string]string{"base": intentionID(base)}
	for name, mutate := range variants {
		t.Run(name, func(t *testing.T) {
			variant := *base
			mutate(&variant)
			id := intentionID(&variant)
			for other, otherID := range seen {
				assert.NotEqual(t, otherID, id,
					"%s must not collide with %s", name, other)
			}
			seen[name] = id
		})
	}
}

func TestIntentionIDIsStableAndEscaped(t *testing.T) {
	ixn := &consulapi.Intention{
		SourceName: "web", SourceNS: "default",
		DestinationName: "db", DestinationNS: "default",
	}
	assert.Equal(t, intentionID(ixn), intentionID(ixn), "the key must not move between reads")
	assert.Equal(t, "", intentionID(nil))

	// a value carrying the separator must not produce a key that reads as a
	// different pairing
	sneaky := &consulapi.Intention{SourceName: "a/b", DestinationName: "c"}
	plain := &consulapi.Intention{SourceName: "a", SourceNS: "b", DestinationName: "c"}
	assert.NotEqual(t, intentionID(plain), intentionID(sneaky))
}

func TestIntentionHasWildcard(t *testing.T) {
	assert.False(t, intentionHasWildcard(nil))
	assert.False(t, intentionHasWildcard(&consulapi.Intention{SourceName: "web", DestinationName: "db"}))
	assert.True(t, intentionHasWildcard(&consulapi.Intention{SourceName: "*", DestinationName: "db"}))
	assert.True(t, intentionHasWildcard(&consulapi.Intention{SourceName: "web", DestinationName: "*"}))
	// a name merely containing a star is not the wildcard
	assert.False(t, intentionHasWildcard(&consulapi.Intention{SourceName: "web*", DestinationName: "db"}))
}

// An intention carrying L7 permissions has no action of its own, so a check
// reading `action == "deny"` says nothing about it. The fixture is the real
// listing of a Consul 1.20.1 datacenter carrying all three shapes.
func TestIntentionFixtureShapes(t *testing.T) {
	var intentions []*consulapi.Intention
	loadFixture(t, "intentions.json", &intentions)
	require.Len(t, intentions, 3)

	byKey := map[string]*consulapi.Intention{}
	for _, ixn := range intentions {
		byKey[ixn.SourceName+"->"+ixn.DestinationName] = ixn
	}

	t.Run("connection level allow", func(t *testing.T) {
		ixn := byKey["web->db"]
		require.NotNil(t, ixn)
		assert.Equal(t, consulapi.IntentionActionAllow, ixn.Action)
		assert.Empty(t, ixn.Permissions)
		assert.False(t, intentionHasWildcard(ixn))
		assert.Equal(t, "web to db", ixn.Description)
		assert.Equal(t, 9, ixn.Precedence)
	})

	t.Run("wildcard deny", func(t *testing.T) {
		ixn := byKey["*->db"]
		require.NotNil(t, ixn)
		assert.Equal(t, consulapi.IntentionActionDeny, ixn.Action)
		assert.True(t, intentionHasWildcard(ixn))
		// the wildcard is evaluated after the specific pairing
		assert.Equal(t, 8, ixn.Precedence)
	})

	t.Run("request level permissions carry no action", func(t *testing.T) {
		ixn := byKey["web->api"]
		require.NotNil(t, ixn)
		assert.Empty(t, string(ixn.Action),
			"an L7 intention has no connection-level action")
		require.Len(t, ixn.Permissions, 1)
		assert.Equal(t, consulapi.IntentionActionAllow, ixn.Permissions[0].Action)
		require.NotNil(t, ixn.Permissions[0].HTTP)
		assert.Equal(t, "/public", ixn.Permissions[0].HTTP.PathPrefix)
	})

	t.Run("every intention has a distinct key", func(t *testing.T) {
		keys := map[string]bool{}
		for _, ixn := range intentions {
			id := intentionID(ixn)
			assert.False(t, keys[id], "duplicate cache key %q", id)
			keys[id] = true
		}
	})

	// the listing carries no timestamps, so the schema must report them as null
	// rather than as a date in year one
	t.Run("absent timestamps stay null", func(t *testing.T) {
		for _, ixn := range intentions {
			assert.Nil(t, nullableTime(ixn.CreatedAt))
			assert.Nil(t, nullableTime(ixn.UpdatedAt))
		}
	})
}
