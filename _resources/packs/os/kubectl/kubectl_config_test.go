// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package kubectl_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers/mock"
	"go.mondoo.com/cnquery/resources/packs/os/kubectl"
)

func TestKubectlConfigParser(t *testing.T) {
	r, err := os.Open("./testdata/kubeconfig_default.yml")
	if err != nil {
		t.Fatal(err)
	}

	config, err := kubectl.ParseKubectlConfig(r)
	require.NoError(t, err)
	assert.Equal(t, "Config", config.Kind)
	assert.Equal(t, "minikube", config.CurrentContext)
	assert.Equal(t, "default", config.CurrentNamespace())

	r, err = os.Open("./testdata/kubeconfig_with_namespace.yml")
	if err != nil {
		t.Fatal(err)
	}

	config, err = kubectl.ParseKubectlConfig(r)
	require.NoError(t, err)
	assert.Equal(t, "Config", config.Kind)
	assert.Equal(t, "minikube", config.CurrentContext)
	assert.Equal(t, "ggckad-s2", config.CurrentNamespace())
}

func TestKubectlExecuter(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/linux_kubeclt.toml")
	if err != nil {
		t.Fatal(err)
	}

	config, err := kubectl.LoadKubeConfig(mock)
	require.NoError(t, err)
	assert.Equal(t, "Config", config.Kind)
	assert.Equal(t, "minikube", config.CurrentContext)
}
