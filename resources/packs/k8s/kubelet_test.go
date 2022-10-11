package k8s_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/resources/packs/k8s"
	"go.mondoo.com/cnquery/resources/packs/os"
	"go.mondoo.com/cnquery/resources/packs/testutils"
)

func TestResource_K8sKubelet(t *testing.T) {
	combinedRegistry := k8s.Registry
	combinedRegistry.Add(os.Registry)
	x := testutils.InitTester(testutils.KubeletMock(), combinedRegistry)

	t.Run("k8s.kubelet resource", func(t *testing.T) {
		res := x.TestQuery(t, "k8s.kubelet")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("kubelet configFile path", func(t *testing.T) {
		res := x.TestQuery(t, "k8s.kubelet.configFile.path")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("kubelet process executable", func(t *testing.T) {
		res := x.TestQuery(t, "k8s.kubelet.process.executable")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		assert.Equal(t, "/var/lib/minikube/binaries/v1.24.3/kubelet", res[0].Data.Value)
	})

	t.Run("kubelet config file flag", func(t *testing.T) {
		res := x.TestQuery(t, "k8s.kubelet.configuration[\"config\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "/var/lib/kubelet/config.yaml", res[0].Data.Value)
	})

	t.Run("check for default value", func(t *testing.T) {
		res := x.TestQuery(t, "k8s.kubelet.configuration[\"volumePluginDir\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/", res[0].Data.Value)
	})

	t.Run("check for config file param", func(t *testing.T) {
		res := x.TestQuery(t, "k8s.kubelet.configuration[\"healthzBindAddress\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "127.0.0.1", res[0].Data.Value)
	})

	t.Run("check for cli flag overwrite", func(t *testing.T) {
		res := x.TestQuery(t, "k8s.kubelet.process.flags[\"runtime-request-timeout\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "15m", res[0].Data.Value)

		res = x.TestQuery(t, "k8s.kubelet.configFile.content")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Contains(t, res[0].Data.Value, "runtimeRequestTimeout: 0s")

		res = x.TestQuery(t, "k8s.kubelet.configuration[\"runtimeRequestTimeout\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "15m", res[0].Data.Value)
	})
}

func TestResource_K8sKubeletAKS(t *testing.T) {
	// AKS is special in that regard, that it does not have a kubelet config file
	// everything is configured via the kubelet process flags
	combinedRegistry := k8s.Registry
	combinedRegistry.Add(os.Registry)
	x := testutils.InitTester(testutils.KubeletAKSMock(), combinedRegistry)

	t.Run("k8s.kubelet resource", func(t *testing.T) {
		res := x.TestQuery(t, "k8s.kubelet")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("kubelet configFile path", func(t *testing.T) {
		res := x.TestQuery(t, "k8s.kubelet.configFile")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		assert.Equal(t, nil, res[0].Data.Value)
	})

	t.Run("kubelet process executable", func(t *testing.T) {
		res := x.TestQuery(t, "k8s.kubelet.process.executable")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		assert.Equal(t, "/usr/local/bin/kubelet", res[0].Data.Value)
	})

	t.Run("kubelet config file flag", func(t *testing.T) {
		res := x.TestQuery(t, "k8s.kubelet.configuration[\"config\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, nil, res[0].Data.Value)
	})

	t.Run("kubelet flag anonymous-auth", func(t *testing.T) {
		res := x.TestQuery(t, "k8s.kubelet.configuration[\"authentication\"][\"anonymous\"][\"enabled\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "false", res[0].Data.Value)
	})

	t.Run("kubelet flag tls-cipher-suites", func(t *testing.T) {
		res := x.TestQuery(t, "k8s.kubelet.configuration[\"tlsCipherSuites\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, 8, len(res[0].Data.Value.([]interface{})))
		assert.Contains(t, res[0].Data.Value.([]interface{}), "TLS_RSA_WITH_AES_128_GCM_SHA256")
	})

	t.Run("kubelet flag eviction-hard", func(t *testing.T) {
		res := x.TestQuery(t, "k8s.kubelet.configuration[\"evictionHard\"][\"memory.available\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "750Mi", res[0].Data.Value)
	})

	t.Run("check for cli flag overwrite", func(t *testing.T) {
		res := x.TestQuery(t, "k8s.kubelet.process.flags[\"read-only-port\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "0", res[0].Data.Value)

		// default is 10250
		res = x.TestQuery(t, "k8s.kubelet.configuration[\"readOnlyPort\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "0", res[0].Data.Value)
	})
}
