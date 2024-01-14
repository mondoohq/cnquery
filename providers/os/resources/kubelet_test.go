// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/testutils"
)

func TestResource_K8sKubelet(t *testing.T) {
	x := testutils.InitTester(testutils.KubeletMock())

	t.Run("kubelet resource", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("kubelet configFile path", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet.configFile.path")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("kubelet process executable", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet.process.executable")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		assert.Equal(t, "/var/lib/minikube/binaries/v1.28.3/kubelet", res[0].Data.Value)
	})

	t.Run("kubelet config file flag", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet.configuration[\"config\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "/var/lib/kubelet/config.yaml", res[0].Data.Value)
	})

	t.Run("check for default value", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet.configuration[\"volumePluginDir\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/", res[0].Data.Value)
	})

	t.Run("check for config file param", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet.configuration[\"healthzBindAddress\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "127.0.0.1", res[0].Data.Value)
	})

	t.Run("check for cli flag overwrite", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet.process.flags[\"runtime-request-timeout\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "7m", res[0].Data.Value)

		res = x.TestQuery(t, "kubelet.configFile.content")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Contains(t, res[0].Data.Value, "runtimeRequestTimeout: 15m0s")

		res = x.TestQuery(t, "kubelet.configuration[\"runtimeRequestTimeout\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "7m", res[0].Data.Value)
	})

	t.Run("kubelet config clientCAFile", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet.configuration[\"authentication\"][\"x509\"][\"clientCAFile\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Contains(t, "/var/lib/minikube/certs/ca.crt", res[0].Data.Value)
	})
}

func TestResource_K8sKubeletAKS(t *testing.T) {
	// AKS is special in that regard, that it does not have a kubelet config file
	// everything is configured via the kubelet process flags
	x := testutils.InitTester(testutils.KubeletAKSMock())

	t.Run("kubelet resource", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("kubelet configFile path", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet.configFile")
		assert.NotEmpty(t, res)
		assert.Error(t, res[0].Data.Error)
	})

	t.Run("kubelet configFile exists", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet.configFile.exists")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		assert.False(t, res[0].Data.Value.(bool))
	})

	t.Run("kubelet process executable", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet.process.executable")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		assert.Equal(t, "/var/lib/minikube/binaries/v1.28.3/kubelet", res[0].Data.Value)
	})

	t.Run("kubelet config file flag", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet.configuration[\"config\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, nil, res[0].Data.Value)
	})

	t.Run("kubelet flag anonymous-auth", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet.configuration[\"authentication\"][\"anonymous\"][\"enabled\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "false", res[0].Data.Value)
	})

	t.Run("kubelet flag tls-cipher-suites", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet.configuration[\"tlsCipherSuites\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, 8, len(res[0].Data.Value.([]interface{})))
		assert.Contains(t, res[0].Data.Value.([]interface{}), "TLS_RSA_WITH_AES_128_GCM_SHA256")
	})

	t.Run("kubelet flag eviction-hard", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet.configuration[\"evictionHard\"][\"memory.available\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "750Mi", res[0].Data.Value)
	})

	t.Run("check for cli flag overwrite", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet.process.flags[\"read-only-port\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "0", res[0].Data.Value)

		// default is 10250
		res = x.TestQuery(t, "kubelet.configuration[\"readOnlyPort\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "0", res[0].Data.Value)
	})
}

func TestResource_K8sKubeletEKS(t *testing.T) {
	// EKS is differetn becasue it uses a JSON config file
	// and set's the read-only-port to 0
	x := testutils.InitTester(testutils.KubeletEKSMock())

	t.Run("kubelet resource", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("kubelet configFile path", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet.configFile")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("kubelet config readOnlyPort", func(t *testing.T) {
		res := x.TestQuery(t, "kubelet.configuration[\"readOnlyPort\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, 0.0, res[0].Data.Value)
	})
}
