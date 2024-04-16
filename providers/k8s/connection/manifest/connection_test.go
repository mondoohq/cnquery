// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package manifest_test

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/providers"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/testutils"
	k8s_conf "go.mondoo.com/cnquery/v11/providers/k8s/config"
	"go.mondoo.com/cnquery/v11/providers/k8s/connection/manifest"
	"go.mondoo.com/cnquery/v11/providers/k8s/connection/shared"
	k8s_provider "go.mondoo.com/cnquery/v11/providers/k8s/provider"
	"go.mondoo.com/cnquery/v11/providers/k8s/resources"
	"go.mondoo.com/cnquery/v11/utils/syncx"
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
				Discover: &inventory.Discovery{
					Targets: []string{"auto"},
				},
			}},
		},
	})
	require.NoError(t, err)

	h := sha256.New()
	absPath, err := filepath.Abs(path)
	require.NoError(t, err)
	h.Write([]byte(absPath))
	manifestHash := hex.EncodeToString(h.Sum(nil))
	require.NoError(t, err)
	// verify that the asset object gets the platform id
	require.Equal(t, "//platformid.api.mondoo.app/runtime/k8s/uid/"+manifestHash, runtime.Provider.Connection.Inventory.Spec.Assets[0].PlatformIds[0])
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
		Resources:      &syncx.Map[plugin.Resource]{},
		Connection:     conn,
		HasRecording:   false,
		CreateResource: resources.CreateResource,
	}
	inv, err := resources.Discover(pluginRuntime, cnquery.Features{})
	require.NoError(t, err)
	require.Len(t, inv.Spec.Assets, 2)

	conn.InventoryConfig().Discover.Targets = []string{"all"}
	pluginRuntime = &plugin.Runtime{
		Resources:      &syncx.Map[plugin.Resource]{},
		Connection:     conn,
		HasRecording:   false,
		CreateResource: resources.CreateResource,
	}
	inv, err = resources.Discover(pluginRuntime, cnquery.Features{})
	require.NoError(t, err)
	require.Len(t, inv.Spec.Assets, 2)

	conn.InventoryConfig().Discover.Targets = []string{"deployments"}
	pluginRuntime = &plugin.Runtime{
		Resources:      &syncx.Map[plugin.Resource]{},
		Connection:     conn,
		HasRecording:   false,
		CreateResource: resources.CreateResource,
	}
	inv, err = resources.Discover(pluginRuntime, cnquery.Features{})
	require.NoError(t, err)
	require.Len(t, inv.Spec.Assets, 1)
}

func TestOperatorManifest(t *testing.T) {
	path := "./testdata/mondoo-operator-manifests.yaml"

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
		Resources:      &syncx.Map[plugin.Resource]{},
		Connection:     conn,
		HasRecording:   false,
		CreateResource: resources.CreateResource,
	}
	inv, err := resources.Discover(pluginRuntime, cnquery.Features{})
	require.NoError(t, err)
	require.Len(t, inv.Spec.Assets, 2)

	require.Len(t, inv.Spec.Assets[1].PlatformIds, 1)

	for i := range inv.Spec.Assets {
		asset := inv.Spec.Assets[i]
		err = runtime.Connect(&plugin.ConnectReq{
			Asset: asset,
		})
		require.NoError(t, err)
		require.NotEmpty(t, asset.PlatformIds[0])
	}

	h := sha256.New()
	absPath, err := filepath.Abs(path)
	require.NoError(t, err)
	h.Write([]byte(absPath))
	manifestHash := hex.EncodeToString(h.Sum(nil))
	require.NoError(t, err)

	require.NotEqual(t, inv.Spec.Assets[0].PlatformIds[0], inv.Spec.Assets[1].PlatformIds[0])
	require.Equal(t, "//platformid.api.mondoo.app/runtime/k8s/uid/"+manifestHash, inv.Spec.Assets[0].PlatformIds[0])
	require.Equal(t, "//platformid.api.mondoo.app/runtime/k8s/uid/"+manifestHash+"/namespace/mondoo-operator/deployments/name/mondoo-operator-controller-manager", inv.Spec.Assets[1].PlatformIds[0])
}

func TestOperatorManifestWithNamespaceFilter(t *testing.T) {
	path := "./testdata/mondoo-operator-manifests.yaml"

	runtime := K8s()
	rootAsset := &inventory.Asset{
		Connections: []*inventory.Config{{
			Type: "k8s",
			Options: map[string]string{
				shared.OPTION_MANIFEST:  path,
				shared.OPTION_NAMESPACE: "mondoo-operator",
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
		Resources:      &syncx.Map[plugin.Resource]{},
		Connection:     conn,
		HasRecording:   false,
		CreateResource: resources.CreateResource,
	}
	inv, err := resources.Discover(pluginRuntime, cnquery.Features{})
	require.NoError(t, err)
	require.Len(t, inv.Spec.Assets, 2)

	require.Len(t, inv.Spec.Assets[1].PlatformIds, 1)

	for i := range inv.Spec.Assets {
		asset := inv.Spec.Assets[i]
		err = runtime.Connect(&plugin.ConnectReq{
			Asset: asset,
		})
		require.NoError(t, err)
		require.NotEmpty(t, asset.PlatformIds[0])
	}
	require.NotEqual(t, inv.Spec.Assets[0].PlatformIds[0], inv.Spec.Assets[1].PlatformIds[0])
	require.Equal(t, "//platformid.api.mondoo.app/runtime/k8s/uid/namespace/mondoo-operator", inv.Spec.Assets[0].PlatformIds[0])
	require.Equal(t, "//platformid.api.mondoo.app/runtime/k8s/uid/namespace/mondoo-operator/deployments/name/mondoo-operator-controller-manager", inv.Spec.Assets[1].PlatformIds[0])
}

func TestManifestNoObjects(t *testing.T) {
	path := "./testdata/no-discovered-objects.yaml"

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
		Resources:      &syncx.Map[plugin.Resource]{},
		Connection:     conn,
		HasRecording:   false,
		CreateResource: resources.CreateResource,
	}
	inv, err := resources.Discover(pluginRuntime, cnquery.Features{})
	require.NoError(t, err)
	require.Len(t, inv.Spec.Assets, 1)

	require.Len(t, inv.Spec.Assets[0].PlatformIds, 1)

	for i := range inv.Spec.Assets {
		asset := inv.Spec.Assets[i]
		err = runtime.Connect(&plugin.ConnectReq{
			Asset: asset,
		})
		require.NoError(t, err)
		require.NotEmpty(t, asset.PlatformIds[0])
	}
	require.NotEmpty(t, inv.Spec.Assets[0].PlatformIds[0])
}

func TestManifestDir(t *testing.T) {
	path := "./testdata/"

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
		Resources:      &syncx.Map[plugin.Resource]{},
		Connection:     conn,
		HasRecording:   false,
		CreateResource: resources.CreateResource,
	}
	inv, err := resources.Discover(pluginRuntime, cnquery.Features{})
	require.NoError(t, err)
	require.Len(t, inv.Spec.Assets, 3)

	for i := range inv.Spec.Assets {
		asset := inv.Spec.Assets[i]
		err = runtime.Connect(&plugin.ConnectReq{
			Asset: asset,
		})
		require.NoError(t, err)
		require.NotEmpty(t, asset.PlatformIds[0])
	}
	require.NotEmpty(t, inv.Spec.Assets[0].PlatformIds[0])
	// we have the operator deployment twice
	require.Equal(t, inv.Spec.Assets[1].PlatformIds[0], inv.Spec.Assets[2].PlatformIds[0])
}
