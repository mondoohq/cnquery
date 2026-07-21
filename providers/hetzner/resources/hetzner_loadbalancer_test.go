// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/stretchr/testify/assert"
)

func TestTargetServerIDs(t *testing.T) {
	srv := func(id int64) hcloud.LoadBalancerTarget {
		return hcloud.LoadBalancerTarget{
			Type:   hcloud.LoadBalancerTargetTypeServer,
			Server: &hcloud.LoadBalancerTargetServer{Server: &hcloud.Server{ID: id}},
		}
	}

	t.Run("empty", func(t *testing.T) {
		assert.Empty(t, targetServerIDs(nil))
	})

	t.Run("extracts and dedupes server ids", func(t *testing.T) {
		got := targetServerIDs([]hcloud.LoadBalancerTarget{srv(10), srv(20), srv(10)})
		assert.Equal(t, []int64{10, 20}, got)
	})

	t.Run("skips entries without a resolved server", func(t *testing.T) {
		targets := []hcloud.LoadBalancerTarget{
			srv(10),
			{Type: hcloud.LoadBalancerTargetTypeServer, Server: nil},
			{Type: hcloud.LoadBalancerTargetTypeServer, Server: &hcloud.LoadBalancerTargetServer{Server: nil}},
			srv(30),
		}
		assert.Equal(t, []int64{10, 30}, targetServerIDs(targets))
	})
}
