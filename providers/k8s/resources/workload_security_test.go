// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/k8s/connection/manifest"
	"go.mondoo.com/mql/v13/providers/k8s/connection/shared"
	"go.mondoo.com/mql/v13/utils/syncx"
)

func workloadSecurityK8s(t *testing.T) *mqlK8s {
	t.Helper()
	conn, err := manifest.NewConnection(0, &inventory.Asset{
		Connections: []*inventory.Config{
			{Options: map[string]string{shared.OPTION_NAMESPACE: "default"}},
		},
	}, manifest.WithManifestFile("./testdata/workload-security.yaml"))
	require.NoError(t, err)

	runtime := &plugin.Runtime{Resources: &syncx.Map[plugin.Resource]{}}
	runtime.Connection = conn

	obj, err := NewResource(runtime, "k8s", nil)
	require.NoError(t, err)
	return obj.(*mqlK8s)
}

func deploymentByName(t *testing.T, k8s *mqlK8s, name string) *mqlK8sDeployment {
	t.Helper()
	deployments := k8s.GetDeployments()
	require.NoError(t, deployments.Error)
	for i := range deployments.Data {
		d := deployments.Data[i].(*mqlK8sDeployment)
		if d.GetName().Data == name {
			return d
		}
	}
	require.FailNowf(t, "deployment not found", "deployment %q not found", name)
	return nil
}

func TestWorkloadSecurityRollups(t *testing.T) {
	k8s := workloadSecurityK8s(t)

	t.Run("hardened workload", func(t *testing.T) {
		d := deploymentByName(t, k8s, "hardened")
		assert.False(t, d.GetRunsPrivileged().Data, "runsPrivileged")
		assert.False(t, d.GetAllowsPrivilegeEscalation().Data, "allowsPrivilegeEscalation")
		assert.False(t, d.GetRunsAsRoot().Data, "runsAsRoot")
		assert.False(t, d.GetHasWritableRootFilesystem().Data, "hasWritableRootFilesystem")
		assert.True(t, d.GetDropsAllCapabilities().Data, "dropsAllCapabilities")
		assert.Equal(t, []any{"NET_BIND_SERVICE"}, d.GetAddedCapabilities().Data, "addedCapabilities")
		assert.False(t, d.GetUsesHostNamespaces().Data, "usesHostNamespaces")
		assert.False(t, d.GetUsesHostPath().Data, "usesHostPath")
		assert.False(t, d.GetAutomountServiceAccountToken().Data, "automountServiceAccountToken")
		assert.False(t, d.GetHostNetwork().Data, "hostNetwork")
	})

	t.Run("risky workload", func(t *testing.T) {
		d := deploymentByName(t, k8s, "risky")
		assert.True(t, d.GetRunsPrivileged().Data, "runsPrivileged")
		assert.True(t, d.GetAllowsPrivilegeEscalation().Data, "allowsPrivilegeEscalation")
		assert.True(t, d.GetRunsAsRoot().Data, "runsAsRoot")
		assert.True(t, d.GetHasWritableRootFilesystem().Data, "hasWritableRootFilesystem")
		assert.False(t, d.GetDropsAllCapabilities().Data, "dropsAllCapabilities")
		// union across init + main containers, deduplicated
		assert.ElementsMatch(t, []any{"NET_RAW", "SYS_ADMIN"}, d.GetAddedCapabilities().Data, "addedCapabilities")
		assert.True(t, d.GetUsesHostNamespaces().Data, "usesHostNamespaces")
		assert.True(t, d.GetUsesHostPath().Data, "usesHostPath")
		assert.True(t, d.GetHostNetwork().Data, "hostNetwork")
		assert.True(t, d.GetHostPID().Data, "hostPID")
		assert.False(t, d.GetHostIPC().Data, "hostIPC")
		// automountServiceAccountToken defaults to true when unset
		assert.True(t, d.GetAutomountServiceAccountToken().Data, "automountServiceAccountToken")
	})

	t.Run("pod-level runAsNonRoot is folded into runsAsRoot", func(t *testing.T) {
		d := deploymentByName(t, k8s, "podlevelnonroot")
		assert.False(t, d.GetRunsAsRoot().Data, "runsAsRoot")
		assert.False(t, d.GetRunsPrivileged().Data, "runsPrivileged")
		assert.True(t, d.GetDropsAllCapabilities().Data, "dropsAllCapabilities")
	})
}
