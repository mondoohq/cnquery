// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package manifest_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v9/providers"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/testutils"
	k8s_conf "go.mondoo.com/cnquery/v9/providers/k8s/config"
	"go.mondoo.com/cnquery/v9/providers/k8s/connection/manifest"
	"go.mondoo.com/cnquery/v9/providers/k8s/connection/shared"
	k8s_provider "go.mondoo.com/cnquery/v9/providers/k8s/provider"
	"go.mondoo.com/cnquery/v9/providers/k8s/resources"
)

func K8s() *providers.Runtime {
	k8sSchema := testutils.MustLoadSchema(testutils.SchemaProvider{Provider: "k8s"})
	runtime := providers.Coordinator.NewRuntime()
	provider := &providers.RunningProvider{
		Name:   k8s_conf.Config.Name,
		ID:     k8s_conf.Config.ID,
		Plugin: k8s_provider.Init(),
		Schema: k8sSchema,
	}
	runtime.Provider = &providers.ConnectedProvider{Instance: provider}
	runtime.AddConnectedProvider(runtime.Provider)
	return runtime
}

func TestPlatformIDDetectionManifest(t *testing.T) {
	path := "./testdata/deployment.yaml"

	runtime := K8s()
	err := runtime.Connect(&plugin.ConnectReq{
		Asset: &inventory.Asset{
			Connections: []*inventory.Config{{
				Type: "k8s",
				Options: map[string]string{
					shared.OPTION_MANIFEST: path,
				},
			}},
		},
	})
	require.NoError(t, err)
	// verify that the asset object gets the platform id
	require.Equal(t, "//platformid.api.mondoo.app/runtime/k8s/uid/5c44b3080881cb47faaedf5754099b8b670a85b69861f64692d6323550197b2d", runtime.Provider.Connection.Asset.PlatformIds[0])
}

func TestManifestDiscovery(t *testing.T) {
	path := "./testdata/deployment.yaml"

	runtime := K8s()
	rootAsset := &inventory.Asset{
		Connections: []*inventory.Config{{
			Type: "k8s",
			Options: map[string]string{
				shared.OPTION_MANIFEST: path,
			},
			Discover: &inventory.Discovery{
				Targets: []string{"auto"},
			},
		}},
	}
	conn, err := manifest.NewConnection(0, rootAsset, manifest.WithManifestFile(path))
	require.NoError(t, err)

	err = runtime.Connect(&plugin.ConnectReq{
		Asset: rootAsset,
	})
	require.NoError(t, err)

	pluginRuntime := &plugin.Runtime{
		Connection:     conn,
		HasRecording:   false,
		CreateResource: resources.CreateResource,
	}
	inv, err := resources.Discover(pluginRuntime)
	require.NoError(t, err)
	require.Len(t, inv.Spec.Assets, 2)

	conn.InventoryConfig().Discover.Targets = []string{"all"}
	pluginRuntime = &plugin.Runtime{
		Connection:     conn,
		HasRecording:   false,
		CreateResource: resources.CreateResource,
	}
	inv, err = resources.Discover(pluginRuntime)
	require.NoError(t, err)
	require.Len(t, inv.Spec.Assets, 2)

	conn.InventoryConfig().Discover.Targets = []string{"deployments"}
	pluginRuntime = &plugin.Runtime{
		Connection:     conn,
		HasRecording:   false,
		CreateResource: resources.CreateResource,
	}
	inv, err = resources.Discover(pluginRuntime)
	require.NoError(t, err)
	require.Len(t, inv.Spec.Assets, 1)
}
