// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/utils/syncx"
	kustomizeTypes "sigs.k8s.io/kustomize/api/types"
)

func newTestRuntime() *plugin.Runtime {
	return &plugin.Runtime{Resources: &syncx.Map[plugin.Resource]{}}
}

// TestNewMqlKustomizeImageUniqueID ensures two images that share a name (legal
// in kustomize — e.g. one overrides the tag, one the digest) get distinct
// __ids, so the second isn't dropped by a resource-cache collision.
func TestNewMqlKustomizeImageUniqueID(t *testing.T) {
	rt := newTestRuntime()

	a, err := newMqlKustomizeImage(rt, "kustomization.yaml", 0, kustomizeTypes.Image{Name: "nginx", NewTag: "1.0"})
	require.NoError(t, err)
	b, err := newMqlKustomizeImage(rt, "kustomization.yaml", 1, kustomizeTypes.Image{Name: "nginx", Digest: "sha256:abc"})
	require.NoError(t, err)

	assert.NotEqual(t, a.__id, b.__id, "same-name images must have distinct __ids")
}

// TestReplacementTargetsWithoutFieldPaths ensures a target that specifies a
// Select but omits fieldPaths still produces a target row (it was previously
// dropped because the only emission happened inside the fieldPaths loop).
func TestReplacementTargetsWithoutFieldPaths(t *testing.T) {
	r := &mqlKustomizeReplacement{MqlRuntime: newTestRuntime()}
	r.kustPath = "kustomization.yaml"
	r.replacementTargets = []*kustomizeTypes.TargetSelector{
		{Select: &kustomizeTypes.Selector{}, FieldPaths: nil},                          // no fieldPaths -> 1 row
		{Select: &kustomizeTypes.Selector{}, FieldPaths: []string{"spec.a", "spec.b"}}, // -> 2 rows
		nil, // a bare `- ` list entry -> skipped
	}

	targets, err := r.targets()
	require.NoError(t, err)
	assert.Len(t, targets, 3, "target with no fieldPaths must still emit one row")
}
