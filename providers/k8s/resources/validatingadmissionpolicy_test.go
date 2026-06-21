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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newManifestRuntime(t *testing.T, manifestFile string) *plugin.Runtime {
	t.Helper()
	conn, err := manifest.NewConnection(0, &inventory.Asset{
		Connections: []*inventory.Config{{}},
	}, manifest.WithManifestFile(manifestFile))
	require.NoError(t, err)
	require.NotNil(t, conn)

	runtime := &plugin.Runtime{Resources: &syncx.Map[plugin.Resource]{}}
	runtime.Connection = conn
	return runtime
}

func TestNamespacePodSecurity(t *testing.T) {
	runtime := newManifestRuntime(t, "./testdata/podsecurity-vap.yaml")

	// The manifest connection synthesizes namespaces from other objects'
	// metadata and does not carry their labels, so construct the namespace
	// resource directly from a Namespace object to exercise the PSA accessors.
	obj, err := CreateResource(runtime, "k8s.namespace", map[string]*llx.RawData{
		"id":   llx.StringData("namespace:secured"),
		"name": llx.StringData("secured"),
	})
	require.NoError(t, err)
	ns := obj.(*mqlK8sNamespace)
	ns.obj = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "secured",
			Labels: map[string]string{
				psaEnforceLabel:        "restricted",
				psaEnforceVersionLabel: "v1.30",
				psaWarnLabel:           "baseline",
			},
		},
	}

	assert.Equal(t, "restricted", ns.GetPodSecurityEnforce().Data)
	assert.Equal(t, "v1.30", ns.GetPodSecurityEnforceVersion().Data)
	assert.Equal(t, "baseline", ns.GetPodSecurityWarn().Data)
	// audit label is not set, so it resolves to empty
	assert.Equal(t, "", ns.GetPodSecurityAudit().Data)
}

func TestManifest_ValidatingAdmissionPolicy(t *testing.T) {
	runtime := newManifestRuntime(t, "./testdata/podsecurity-vap.yaml")

	obj, err := NewResource(runtime, "k8s", nil)
	require.NoError(t, err)
	k8s := obj.(*mqlK8s)

	policies := k8s.GetValidatingAdmissionPolicies()
	require.NoError(t, policies.Error)
	require.Len(t, policies.Data, 1)

	vap := policies.Data[0].(*mqlK8sAdmissionValidatingadmissionpolicy)
	assert.Equal(t, "demo-policy", vap.GetName().Data)
	assert.Equal(t, "Fail", vap.GetFailurePolicy().Data)
	assert.Len(t, vap.GetValidations().Data, 1)

	// the policy resolves the bindings that reference it by name
	bindings := vap.GetBindings()
	require.NoError(t, bindings.Error)
	require.Len(t, bindings.Data, 1)
	binding := bindings.Data[0].(*mqlK8sAdmissionValidatingadmissionpolicybinding)
	assert.Equal(t, "demo-binding", binding.GetName().Data)
	assert.Equal(t, []any{"Deny", "Audit"}, binding.GetValidationActions().Data)

	// the binding resolves back to its typed policy
	policy := binding.GetPolicy()
	require.NoError(t, policy.Error)
	require.NotNil(t, policy.Data)
	assert.Equal(t, "demo-policy", policy.Data.GetName().Data)
}
