// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"cloud.google.com/go/container/apiv1/containerpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeTaintID(t *testing.T) {
	// Kubernetes taints are uniquely identified by (key, value, effect). A node
	// pool may carry siblings that share any one or two of those — collisions
	// here silently drop taints from the runtime cache.
	cases := []struct {
		name string
		want string
		got  string
	}{
		{
			name: "happy path",
			want: "pool1/taints/dedicated/team-a/NoSchedule",
			got:  nodeTaintID("pool1", "dedicated", "team-a", "NoSchedule"),
		},
		{
			name: "same key, different value",
			want: "pool1/taints/dedicated/team-b/NoSchedule",
			got:  nodeTaintID("pool1", "dedicated", "team-b", "NoSchedule"),
		},
		{
			name: "same key & value, different effect",
			want: "pool1/taints/dedicated/team-a/PreferNoSchedule",
			got:  nodeTaintID("pool1", "dedicated", "team-a", "PreferNoSchedule"),
		},
	}
	seen := make(map[string]string, len(cases))
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.got)
			if prev, exists := seen[tc.got]; exists {
				t.Fatalf("id collision: %q produced by both %q and %q", tc.got, prev, tc.name)
			}
			seen[tc.got] = tc.name
		})
	}
}

func TestBuildGKENotificationConfig(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result, err := buildGKENotificationConfig(nil, "parent", nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("nil pubsub returns nil", func(t *testing.T) {
		nc := &containerpb.NotificationConfig{Pubsub: nil}
		result, err := buildGKENotificationConfig(nil, "parent", nc)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}
