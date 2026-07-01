// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/k8s/connection/shared"
	sharedres "go.mondoo.com/mql/v13/providers/k8s/connection/shared/resources"
	"go.mondoo.com/mql/v13/utils/syncx"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
)

func TestDiscoverStagedDiscoveryWithCommaSeparatedNamespacesStartsClusterStage(t *testing.T) {
	cfg := &inventory.Config{
		Type: "k8s",
		Options: map[string]string{
			plugin.OptionStagedDiscovery: "",
			shared.OPTION_NAMESPACE:      "team-a,team-b",
		},
		Discover: &inventory.Discovery{
			Targets: []string{DiscoveryNamespaces},
		},
	}
	asset := &inventory.Asset{
		Name:        "K8s Cluster test",
		Connections: []*inventory.Config{cfg},
	}
	conn := &namespaceDiscoveryConnection{
		Connection: plugin.NewConnection(1, asset),
		asset:      asset,
		namespaces: []corev1.Namespace{
			newTestNamespace("team-a", "uid-team-a"),
			newTestNamespace("team-b", "uid-team-b"),
		},
	}
	pluginRuntime := &plugin.Runtime{
		Resources:  &syncx.Map[plugin.Resource]{},
		Connection: conn,
	}
	pluginRuntime.CreateResource = func(runtime *plugin.Runtime, name string, args map[string]*llx.RawData) (plugin.Resource, error) {
		return &mqlK8s{MqlRuntime: runtime}, nil
	}

	inv, err := Discover(pluginRuntime, mql.Features{})
	require.NoError(t, err)
	require.Len(t, inv.Spec.Assets, 2)
	require.Empty(t, conn.namespaceGetCalls)
	require.Equal(t, "team-a", inv.Spec.Assets[0].Connections[0].Options[shared.OPTION_NAMESPACE])
	require.Equal(t, "team-b", inv.Spec.Assets[1].Connections[0].Options[shared.OPTION_NAMESPACE])
}

func newTestNamespace(name, uid string) corev1.Namespace {
	return corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			UID:  types.UID(uid),
		},
	}
}

type namespaceDiscoveryConnection struct {
	plugin.Connection
	asset             *inventory.Asset
	namespaces        []corev1.Namespace
	namespaceGetCalls []string
}

func (c *namespaceDiscoveryConnection) Name() string {
	return c.asset.Name
}

func (c *namespaceDiscoveryConnection) Runtime() string {
	return "k8s-cluster"
}

func (c *namespaceDiscoveryConnection) Resources(kind string, name string, namespace string) (*shared.ResourceResult, error) {
	return nil, fmt.Errorf("unexpected resource lookup %s/%s/%s", kind, namespace, name)
}

func (c *namespaceDiscoveryConnection) ServerVersion() *version.Info {
	return nil
}

func (c *namespaceDiscoveryConnection) SupportedResourceTypes() (*sharedres.ApiResourceIndex, error) {
	return nil, nil
}

func (c *namespaceDiscoveryConnection) Platform() *inventory.Platform {
	return &inventory.Platform{
		Name:    "k8s-cluster",
		Family:  []string{"k8s"},
		Kind:    "api",
		Runtime: c.Runtime(),
		Title:   "Kubernetes Cluster",
	}
}

func (c *namespaceDiscoveryConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *namespaceDiscoveryConnection) AssetId() (string, error) {
	return shared.NewPlatformId("cluster-uid"), nil
}

func (c *namespaceDiscoveryConnection) BasePlatformId() (string, error) {
	return shared.IdPrefix, nil
}

func (c *namespaceDiscoveryConnection) AdmissionReviews() ([]admissionv1.AdmissionReview, error) {
	return []admissionv1.AdmissionReview{}, nil
}

func (c *namespaceDiscoveryConnection) Namespace(name string) (*corev1.Namespace, error) {
	c.namespaceGetCalls = append(c.namespaceGetCalls, name)
	for i := range c.namespaces {
		if c.namespaces[i].Name == name {
			return &c.namespaces[i], nil
		}
	}
	return nil, fmt.Errorf("namespace %q not found", name)
}

func (c *namespaceDiscoveryConnection) Namespaces() ([]corev1.Namespace, error) {
	return c.namespaces, nil
}

func (c *namespaceDiscoveryConnection) InventoryConfig() *inventory.Config {
	return c.asset.Connections[0]
}

var _ shared.Connection = (*namespaceDiscoveryConnection)(nil)
