// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/k8s/connection/manifest"
	"go.mondoo.com/mql/v13/providers/k8s/connection/shared"
	"go.mondoo.com/mql/v13/utils/syncx"
)

func TestContainerSecurityContextFields(t *testing.T) {
	manifestFile := "../connection/shared/resources/testdata/pod-securitycontext.yaml"
	conn, err := manifest.NewConnection(0, &inventory.Asset{
		Connections: []*inventory.Config{
			{Options: map[string]string{shared.OPTION_NAMESPACE: "default"}},
		},
	}, manifest.WithManifestFile(manifestFile))
	require.NoError(t, err)

	runtime := &plugin.Runtime{Resources: &syncx.Map[plugin.Resource]{}}
	runtime.Connection = conn

	obj, err := NewResource(runtime, "k8s.pod", map[string]*llx.RawData{
		"name":      llx.StringData("secctx"),
		"namespace": llx.StringData("default"),
	})
	require.NoError(t, err)
	pod := obj.(*mqlK8sPod)

	containers := pod.GetContainers()
	require.NoError(t, containers.Error)
	require.Len(t, containers.Data, 2)

	t.Run("fully specified security context", func(t *testing.T) {
		c := containers.Data[0].(*mqlK8sContainer)
		assert.Equal(t, "app", c.GetName().Data)
		assert.Equal(t, false, c.GetPrivileged().Data)
		assert.Equal(t, false, c.GetAllowPrivilegeEscalation().Data)
		assert.Equal(t, true, c.GetRunAsNonRoot().Data)
		assert.Equal(t, int64(1000), c.GetRunAsUser().Data)
		assert.Equal(t, int64(3000), c.GetRunAsGroup().Data)
		assert.Equal(t, true, c.GetReadOnlyRootFilesystem().Data)
		assert.Equal(t, []any{"NET_BIND_SERVICE"}, c.GetAddedCapabilities().Data)
		assert.Equal(t, []any{"ALL"}, c.GetDroppedCapabilities().Data)
		assert.Equal(t, "RuntimeDefault", c.GetSeccompProfileType().Data)
	})

	t.Run("absent security context leaves pointer fields null", func(t *testing.T) {
		c := containers.Data[1].(*mqlK8sContainer)
		assert.Equal(t, "bare", c.GetName().Data)
		assert.NotEqual(t, 0, c.GetPrivileged().State&plugin.StateIsNull)
		assert.NotEqual(t, 0, c.GetRunAsUser().State&plugin.StateIsNull)
		assert.Empty(t, c.GetAddedCapabilities().Data)
		assert.Empty(t, c.GetDroppedCapabilities().Data)
		assert.Equal(t, "", c.GetSeccompProfileType().Data)
	})
}
