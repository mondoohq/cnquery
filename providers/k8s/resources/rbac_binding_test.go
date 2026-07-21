// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clusterRoleBindingByName(t *testing.T, k8s *mqlK8s, name string) *mqlK8sRbacClusterrolebinding {
	t.Helper()
	bindings := k8s.GetClusterrolebindings()
	require.NoError(t, bindings.Error)
	for i := range bindings.Data {
		b := bindings.Data[i].(*mqlK8sRbacClusterrolebinding)
		if b.GetName().Data == name {
			return b
		}
	}
	require.FailNowf(t, "clusterrolebinding not found", "clusterrolebinding %q not found", name)
	return nil
}

func roleBindingByName(t *testing.T, k8s *mqlK8s, name string) *mqlK8sRbacRolebinding {
	t.Helper()
	bindings := k8s.GetRolebindings()
	require.NoError(t, bindings.Error)
	for i := range bindings.Data {
		b := bindings.Data[i].(*mqlK8sRbacRolebinding)
		if b.GetName().Data == name {
			return b
		}
	}
	require.FailNowf(t, "rolebinding not found", "rolebinding %q not found", name)
	return nil
}

func TestRbacRoleGrantsClusterAdmin(t *testing.T) {
	k8sObj, err := NewResource(rbacRuntime(t), "k8s", nil)
	require.NoError(t, err)
	k8s := k8sObj.(*mqlK8s)

	assert.True(t, clusterRoleByName(t, k8s, "wildcard-admin").GetGrantsClusterAdmin().Data, "wildcard-admin")
	assert.False(t, clusterRoleByName(t, k8s, "secret-reader").GetGrantsClusterAdmin().Data, "secret-reader")
	// A role that escalates but isn't a full wildcard is not cluster-admin.
	assert.False(t, clusterRoleByName(t, k8s, "escalator").GetGrantsClusterAdmin().Data, "escalator")
	assert.False(t, roleByName(t, k8s, "binder").GetGrantsClusterAdmin().Data, "binder")
}

// TestRbacClusterRoleBindingRollups checks that a ClusterRoleBinding surfaces
// the danger of the ClusterRole it grants.
func TestRbacClusterRoleBindingRollups(t *testing.T) {
	k8sObj, err := NewResource(rbacRuntime(t), "k8s", nil)
	require.NoError(t, err)
	k8s := k8sObj.(*mqlK8s)

	b := clusterRoleBindingByName(t, k8s, "admin-binding")
	assert.True(t, b.GetGrantsClusterAdmin().Data, "grantsClusterAdmin")
	assert.True(t, b.GetHasWildcardRule().Data, "hasWildcardRule")
	assert.True(t, b.GetAllowsPrivilegeEscalation().Data, "allowsPrivilegeEscalation")
	assert.True(t, b.GetCanReadSecrets().Data, "canReadSecrets")
}

// TestRbacRoleBindingRollups checks the rollups for RoleBindings that reference
// a namespaced Role and ones that reference a ClusterRole.
func TestRbacRoleBindingRollups(t *testing.T) {
	k8sObj, err := NewResource(rbacRuntime(t), "k8s", nil)
	require.NoError(t, err)
	k8s := k8sObj.(*mqlK8s)

	t.Run("benign role binding", func(t *testing.T) {
		b := roleBindingByName(t, k8s, "reader-binding")
		assert.False(t, b.GetGrantsClusterAdmin().Data, "grantsClusterAdmin")
		assert.False(t, b.GetHasWildcardRule().Data, "hasWildcardRule")
		assert.False(t, b.GetAllowsPrivilegeEscalation().Data, "allowsPrivilegeEscalation")
		assert.False(t, b.GetCanReadSecrets().Data, "canReadSecrets")
	})

	t.Run("role binding to escalating role", func(t *testing.T) {
		b := roleBindingByName(t, k8s, "binder-binding")
		assert.True(t, b.GetAllowsPrivilegeEscalation().Data, "allowsPrivilegeEscalation")
		assert.False(t, b.GetGrantsClusterAdmin().Data, "grantsClusterAdmin")
		assert.False(t, b.GetCanReadSecrets().Data, "canReadSecrets")
	})

	t.Run("role binding referencing a cluster role", func(t *testing.T) {
		// secret-binding -> ClusterRole secret-reader, resolved via the
		// clusterRole() path.
		b := roleBindingByName(t, k8s, "secret-binding")
		assert.True(t, b.GetCanReadSecrets().Data, "canReadSecrets")
		assert.False(t, b.GetGrantsClusterAdmin().Data, "grantsClusterAdmin")
		assert.False(t, b.GetAllowsPrivilegeEscalation().Data, "allowsPrivilegeEscalation")
	})
}
