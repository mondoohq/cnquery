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

	t.Run("securityContext dict surfaces the pod-level context", func(t *testing.T) {
		d := deploymentByName(t, k8s, "podlevelnonroot")
		sc := d.GetSecurityContext()
		require.NoError(t, sc.Error)
		scMap, ok := sc.Data.(map[string]any)
		require.True(t, ok, "securityContext is a dict")
		assert.Equal(t, true, scMap["runAsNonRoot"], "securityContext.runAsNonRoot")
	})
}

// TestWorkloadSecurityRollups_Pod exercises the pod accessor path, which uses
// podSpecTyped() rather than the controller securitySpec() helper.
func TestWorkloadSecurityRollups_Pod(t *testing.T) {
	k8s := workloadSecurityK8s(t)

	pods := k8s.GetPods()
	require.NoError(t, pods.Error)
	var pod *mqlK8sPod
	for i := range pods.Data {
		p := pods.Data[i].(*mqlK8sPod)
		if p.GetName().Data == "risky-pod" {
			pod = p
			break
		}
	}
	require.NotNil(t, pod, "risky-pod not found")

	assert.True(t, pod.GetRunsPrivileged().Data, "runsPrivileged")
	assert.True(t, pod.GetAllowsPrivilegeEscalation().Data, "allowsPrivilegeEscalation")
	assert.True(t, pod.GetRunsAsRoot().Data, "runsAsRoot")
	assert.True(t, pod.GetHasWritableRootFilesystem().Data, "hasWritableRootFilesystem")
	assert.False(t, pod.GetDropsAllCapabilities().Data, "dropsAllCapabilities")
	assert.Equal(t, []any{"SYS_ADMIN"}, pod.GetAddedCapabilities().Data, "addedCapabilities")
	assert.True(t, pod.GetUsesHostNamespaces().Data, "usesHostNamespaces (hostIPC)")
	assert.True(t, pod.GetUsesHostPath().Data, "usesHostPath")
}

// rollupReader is the common accessor surface every workload-bearing resource
// implements; it lets one table verify per-kind wiring.
type rollupReader interface {
	GetName() *plugin.TValue[string]
	GetRunsPrivileged() *plugin.TValue[bool]
	GetSecurityContext() *plugin.TValue[any]
}

// TestWorkloadSecurityRollups_AllKinds confirms every controller workload wires
// the predicates and the securityContext dict to its own object getter.
func TestWorkloadSecurityRollups_AllKinds(t *testing.T) {
	k8s := workloadSecurityK8s(t)

	find := func(t *testing.T, list *plugin.TValue[[]any], name string) rollupReader {
		t.Helper()
		require.NoError(t, list.Error)
		for i := range list.Data {
			r := list.Data[i].(rollupReader)
			if r.GetName().Data == name {
				return r
			}
		}
		require.FailNowf(t, "workload not found", "%q not found", name)
		return nil
	}

	cases := []struct {
		name string
		list *plugin.TValue[[]any]
	}{
		{"ds-privileged", k8s.GetDaemonsets()},
		{"ss-privileged", k8s.GetStatefulsets()},
		{"rs-privileged", k8s.GetReplicasets()},
		{"job-privileged", k8s.GetJobs()},
		{"cronjob-privileged", k8s.GetCronjobs()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := find(t, tc.list, tc.name)
			assert.True(t, r.GetRunsPrivileged().Data, "runsPrivileged")
			// securityContext resolves without error even when unset
			assert.NoError(t, r.GetSecurityContext().Error, "securityContext")
		})
	}
}
