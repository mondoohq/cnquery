// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewWorkloadPlatformId pins the workload platform-id construction, which
// is load-bearing for asset identity and caching: a regression silently merges
// or splits assets. It covers pluralization, the namespace-injection branch
// (which keys off whether the cluster identifier already contains "namespace"),
// and the namespace delegation.
func TestNewWorkloadPlatformId(t *testing.T) {
	const base = "//platformid.api.mondoo.app/runtime/k8s/uid/abc"

	t.Run("injects namespace when cluster id lacks one", func(t *testing.T) {
		got := NewWorkloadPlatformId(base, "/cluster/123", "deployment", "prod", "api", "uid-1")
		assert.Equal(t, "/cluster/123/namespace/prod/deployments/name/api", got)
	})

	t.Run("does not double-inject namespace when cluster id already has one", func(t *testing.T) {
		// When scanning with --namespace, the cluster identifier already
		// carries the namespace and must not get a second segment.
		clusterID := "/cluster/123/namespace/prod"
		got := NewWorkloadPlatformId(base, clusterID, "deployment", "prod", "api", "uid-1")
		assert.Equal(t, "/cluster/123/namespace/prod/deployments/name/api", got)
	})

	t.Run("omits namespace segment for empty namespace", func(t *testing.T) {
		got := NewWorkloadPlatformId(base, "/cluster/123", "node", "", "worker-1", "uid-2")
		assert.Equal(t, "/cluster/123/nodes/name/worker-1", got)
	})

	t.Run("namespace workload type delegates to namespace id", func(t *testing.T) {
		got := NewWorkloadPlatformId(base, "/cluster/123", "namespace", "", "prod", "uid-3")
		assert.Equal(t, NewNamespacePlatformId(base, "prod", "uid-3"), got)
	})
}

func TestNewNamespacePlatformId(t *testing.T) {
	const base = "//platformid.api.mondoo.app/runtime/k8s/uid/abc"
	got := NewNamespacePlatformId(base, "prod", "uid-3")
	assert.Equal(t, base+"uid-3/namespace/prod", got)
}
