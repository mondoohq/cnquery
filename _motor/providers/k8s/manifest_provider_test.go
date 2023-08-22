// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package k8s

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s/resources"
)

type K8sObjectKindTest struct {
	kind string
}

func TestManifestFiles(t *testing.T) {
	tests := []K8sObjectKindTest{
		{kind: "cronjob"},
		{kind: "job"},
		{kind: "deployment"},
		{kind: "pod"},
		{kind: "statefulset"},
		{kind: "replicaset"},
		{kind: "daemonset"},
	}
	for _, testCase := range tests {
		t.Run("k8s "+testCase.kind, func(t *testing.T) {
			manifestFile := "./resources/testdata/" + testCase.kind + ".yaml"
			provider, err := newManifestProvider("", testCase.kind, WithManifestFile(manifestFile))
			require.NoError(t, err)
			require.NotNil(t, provider)
			res, err := provider.Resources(testCase.kind, "mondoo", "default")
			require.NoError(t, err)
			assert.Equal(t, "mondoo", res.Name)
			assert.Equal(t, testCase.kind, res.Kind)
			assert.Equal(t, "k8s-manifest", provider.PlatformInfo().Runtime)
			assert.Equal(t, 1, len(res.Resources))
			podSpec, err := resources.GetPodSpec(res.Resources[0])
			require.NoError(t, err)
			assert.NotNil(t, podSpec)
			containers, err := resources.GetContainers(res.Resources[0])
			require.NoError(t, err)
			assert.Equal(t, 1, len(containers))
			initContainers, err := resources.GetInitContainers(res.Resources[0])
			require.NoError(t, err)
			assert.Equal(t, 0, len(initContainers))
		})
	}
}

func TestManifestFile_CustomResource(t *testing.T) {
	manifestFile := "./resources/testdata/cr/tekton.yaml"
	provider, err := newManifestProvider("", "", WithManifestFile(manifestFile))
	require.NoError(t, err)
	require.NotNil(t, provider)

	name := "demo-pipeline"
	namespace := "default"
	kind := "pipeline.tekton.dev"
	res, err := provider.Resources(kind, name, namespace)
	require.NoError(t, err)
	assert.Equal(t, name, res.Name)
	assert.Equal(t, namespace, res.Namespace)
	assert.Equal(t, kind, res.Kind)
	assert.Equal(t, "k8s-manifest", provider.PlatformInfo().Runtime)
	assert.Equal(t, 1, len(res.Resources))
}

func TestManifestFileProvider(t *testing.T) {
	t.Run("k8s manifest provider with file", func(t *testing.T) {
		manifestFile := "./resources/testdata/pod.yaml"
		provider, err := newManifestProvider("", "", WithManifestFile(manifestFile))
		require.NoError(t, err)
		require.NotNil(t, provider)
		assert.Equal(t, "k8s-manifest", provider.PlatformInfo().Name)
		assert.Equal(t, "k8s-manifest", provider.PlatformInfo().Runtime)
		assert.Equal(t, providers.Kind_KIND_CODE, provider.PlatformInfo().Kind)
		assert.Contains(t, provider.PlatformInfo().Family, "k8s")
	})
}

func TestManifestContentProvider(t *testing.T) {
	t.Run("k8s manifest provider with content", func(t *testing.T) {
		manifestFile := "./resources/testdata/pod.yaml"
		data, err := os.ReadFile(manifestFile)
		require.NoError(t, err)

		provider, err := newManifestProvider("", "", WithManifestContent(data))
		require.NoError(t, err)
		require.NotNil(t, provider)
		name, err := provider.Name()
		require.NoError(t, err)
		assert.Equal(t, "K8s Manifest", name)
		assert.Equal(t, "k8s-manifest", provider.PlatformInfo().Name)
		assert.Equal(t, "k8s-manifest", provider.PlatformInfo().Runtime)
		assert.Equal(t, providers.Kind_KIND_CODE, provider.PlatformInfo().Kind)
		assert.Contains(t, provider.PlatformInfo().Family, "k8s")
	})
}

func TestLoadManifestDirRecursively(t *testing.T) {
	manifests, err := loadManifestFile("./resources/testdata/")
	require.NoError(t, err)

	manifestsAsString := string(manifests[:])
	// This is content from files of the root dir
	assert.Contains(t, manifestsAsString, "mondoo")
	assert.Contains(t, manifestsAsString, "RollingUpdate")

	// Files containing this should be skipped
	assert.NotContains(t, manifestsAsString, "AdmissionReview")
	assert.NotContains(t, manifestsAsString, "README")
	assert.NotContains(t, manifestsAsString, "operators.coreos.com")

	// This is from files in subdirs whicch should be included
	assert.Contains(t, manifestsAsString, "hello-1")
	assert.Contains(t, manifestsAsString, "hello-2")
	assert.Contains(t, manifestsAsString, "MondooAuditConfig")
}
