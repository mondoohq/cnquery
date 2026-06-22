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
	"go.mondoo.com/mql/v13/utils/syncx"
)

func exposureRuntime(t *testing.T) *plugin.Runtime {
	t.Helper()
	conn, err := manifest.NewConnection(0, &inventory.Asset{
		Connections: []*inventory.Config{{}},
	}, manifest.WithManifestFile("./testdata/exposure_targets.yaml"))
	require.NoError(t, err)

	runtime := &plugin.Runtime{Resources: &syncx.Map[plugin.Resource]{}}
	runtime.Connection = conn
	return runtime
}

func TestServicePods(t *testing.T) {
	runtime := exposureRuntime(t)

	svc, err := NewResource(runtime, "k8s.service", map[string]*llx.RawData{
		"name":      llx.StringData("web-svc"),
		"namespace": llx.StringData("prod"),
	})
	require.NoError(t, err)
	pods := svc.(*mqlK8sService).GetPods()
	require.NoError(t, pods.Error)
	// Selects app=web in prod only — excludes other-1 and the same-labeled pod
	// in the staging namespace.
	assert.Equal(t, []string{"web-1", "web-2"}, mqlResourceNames(t, pods.Data))

	// Selectorless service selects nothing.
	headless, err := NewResource(runtime, "k8s.service", map[string]*llx.RawData{
		"name":      llx.StringData("headless"),
		"namespace": llx.StringData("prod"),
	})
	require.NoError(t, err)
	hp := headless.(*mqlK8sService).GetPods()
	require.NoError(t, hp.Error)
	assert.Empty(t, hp.Data)
}

func TestPodServices(t *testing.T) {
	runtime := exposureRuntime(t)

	pod, err := NewResource(runtime, "k8s.pod", map[string]*llx.RawData{
		"name":      llx.StringData("web-1"),
		"namespace": llx.StringData("prod"),
	})
	require.NoError(t, err)
	svcs := pod.(*mqlK8sPod).GetServices()
	require.NoError(t, svcs.Error)
	assert.Equal(t, []string{"web-svc"}, mqlResourceNames(t, svcs.Data))

	// A pod no service selects resolves to no services.
	other, err := NewResource(runtime, "k8s.pod", map[string]*llx.RawData{
		"name":      llx.StringData("other-1"),
		"namespace": llx.StringData("prod"),
	})
	require.NoError(t, err)
	os := other.(*mqlK8sPod).GetServices()
	require.NoError(t, os.Error)
	assert.Empty(t, os.Data)
}

func TestIngressPods(t *testing.T) {
	runtime := exposureRuntime(t)

	ing, err := NewResource(runtime, "k8s.ingress", map[string]*llx.RawData{
		"name":      llx.StringData("web-ing"),
		"namespace": llx.StringData("prod"),
	})
	require.NoError(t, err)
	pods := ing.(*mqlK8sIngress).GetPods()
	require.NoError(t, pods.Error)
	assert.Equal(t, []string{"web-1", "web-2"}, mqlResourceNames(t, pods.Data))
}

func TestGatewayPods(t *testing.T) {
	runtime := exposureRuntime(t)

	gw, err := NewResource(runtime, "k8s.gateway", map[string]*llx.RawData{
		"name":      llx.StringData("gw"),
		"namespace": llx.StringData("prod"),
	})
	require.NoError(t, err)
	pods := gw.(*mqlK8sGateway).GetPods()
	require.NoError(t, pods.Error)
	// HTTPRoute and GRPCRoute both target web-svc; the backing pods dedup.
	assert.Equal(t, []string{"web-1", "web-2"}, mqlResourceNames(t, pods.Data))
}

func TestNetworkExposurePods(t *testing.T) {
	k8sObj, err := NewResource(exposureRuntime(t), "k8s", nil)
	require.NoError(t, err)
	k8s := k8sObj.(*mqlK8s)

	exps := k8s.GetNetworkExposures()
	require.NoError(t, exps.Error)

	var serviceExposure *mqlK8sNetworkExposure
	for i := range exps.Data {
		e := exps.Data[i].(*mqlK8sNetworkExposure)
		if e.SourceKind.Data == "Service" && e.Name.Data == "web-svc" {
			serviceExposure = e
			break
		}
	}
	require.NotNil(t, serviceExposure, "expected a Service-sourced exposure for web-svc")

	pods := serviceExposure.GetPods()
	require.NoError(t, pods.Error)
	assert.Equal(t, []string{"web-1", "web-2"}, mqlResourceNames(t, pods.Data))
}

func secretByName(t *testing.T, runtime *plugin.Runtime, name string) *mqlK8sSecret {
	t.Helper()
	s, err := NewResource(runtime, "k8s.secret", map[string]*llx.RawData{
		"name":      llx.StringData(name),
		"namespace": llx.StringData("prod"),
	})
	require.NoError(t, err)
	return s.(*mqlK8sSecret)
}

func TestSecretHygiene(t *testing.T) {
	runtime := exposureRuntime(t)

	tests := []struct {
		name              string
		serviceAccountTok bool
		imagePull         bool
		unused            bool
	}{
		// mounted by web-1
		{"used-secret", false, false, false},
		// image-pull secret referenced by web-1
		{"pull", false, true, false},
		// no pod and no service account reference it
		{"orphan", false, false, true},
		// referenced only by web-sa, so not orphaned despite no pod use
		{"sa-token", true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := secretByName(t, runtime, tt.name)

			sat := s.GetIsServiceAccountToken()
			require.NoError(t, sat.Error)
			assert.Equal(t, tt.serviceAccountTok, sat.Data, "isServiceAccountToken")

			ips := s.GetIsImagePullSecret()
			require.NoError(t, ips.Error)
			assert.Equal(t, tt.imagePull, ips.Data, "isImagePullSecret")

			unused := s.GetIsUnused()
			require.NoError(t, unused.Error)
			assert.Equal(t, tt.unused, unused.Data, "isUnused")
		})
	}
}
