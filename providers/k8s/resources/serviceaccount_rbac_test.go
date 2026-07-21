// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func serviceAccountByName(t *testing.T, k8s *mqlK8s, namespace, name string) *mqlK8sServiceaccount {
	t.Helper()
	sas := k8s.GetServiceaccounts()
	require.NoError(t, sas.Error)
	for i := range sas.Data {
		sa := sas.Data[i].(*mqlK8sServiceaccount)
		if sa.GetName().Data == name && sa.GetNamespace().Data == namespace {
			return sa
		}
	}
	require.FailNowf(t, "serviceaccount not found", "serviceaccount %s/%s not found", namespace, name)
	return nil
}

func TestServiceAccountRbacRollups(t *testing.T) {
	k8sObj, err := NewResource(rbacRuntime(t), "k8s", nil)
	require.NoError(t, err)
	k8s := k8sObj.(*mqlK8s)

	t.Run("cluster-admin service account", func(t *testing.T) {
		sa := serviceAccountByName(t, k8s, "default", "admin-sa")
		assert.True(t, sa.GetIsClusterAdmin().Data, "isClusterAdmin")
		assert.True(t, sa.GetCanEscalatePrivileges().Data, "canEscalatePrivileges")
		assert.True(t, sa.GetCanReadSecrets().Data, "canReadSecrets")
		assert.True(t, sa.GetHasWildcardPermissions().Data, "hasWildcardPermissions")

		// bound via a ClusterRoleBinding, not a RoleBinding
		crbs := sa.GetClusterRoleBindings()
		require.NoError(t, crbs.Error)
		assert.Len(t, crbs.Data, 1, "clusterRoleBindings")
		rbs := sa.GetRoleBindings()
		require.NoError(t, rbs.Error)
		assert.Empty(t, rbs.Data, "roleBindings")
	})

	t.Run("benign service account", func(t *testing.T) {
		sa := serviceAccountByName(t, k8s, "default", "reader-sa")
		assert.False(t, sa.GetIsClusterAdmin().Data, "isClusterAdmin")
		assert.False(t, sa.GetCanEscalatePrivileges().Data, "canEscalatePrivileges")
		assert.False(t, sa.GetCanReadSecrets().Data, "canReadSecrets")
		assert.False(t, sa.GetHasWildcardPermissions().Data, "hasWildcardPermissions")

		rbs := sa.GetRoleBindings()
		require.NoError(t, rbs.Error)
		assert.Len(t, rbs.Data, 1, "roleBindings")
	})

	t.Run("secret-reading service account via cluster role", func(t *testing.T) {
		// secret-sa is bound (via a RoleBinding) to the ClusterRole secret-reader.
		sa := serviceAccountByName(t, k8s, "default", "secret-sa")
		assert.True(t, sa.GetCanReadSecrets().Data, "canReadSecrets")
		assert.False(t, sa.GetIsClusterAdmin().Data, "isClusterAdmin")
		assert.False(t, sa.GetCanEscalatePrivileges().Data, "canEscalatePrivileges")
	})

	t.Run("privilege-escalating service account", func(t *testing.T) {
		sa := serviceAccountByName(t, k8s, "default", "binder-sa")
		assert.True(t, sa.GetCanEscalatePrivileges().Data, "canEscalatePrivileges")
		assert.False(t, sa.GetIsClusterAdmin().Data, "isClusterAdmin")
		assert.False(t, sa.GetCanReadSecrets().Data, "canReadSecrets")
	})

	t.Run("namespaced-only binding has no cluster bindings", func(t *testing.T) {
		// binder-sa is bound only via a RoleBinding, so its clusterRoleBindings
		// list is empty.
		sa := serviceAccountByName(t, k8s, "default", "binder-sa")
		crbs := sa.GetClusterRoleBindings()
		require.NoError(t, crbs.Error)
		assert.Empty(t, crbs.Data, "clusterRoleBindings")
	})
}
