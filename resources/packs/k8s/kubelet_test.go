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

	t.Run("kubelet process command", func(t *testing.T) {
		res := x.TestQuery(t, "k8s.kubelet.process.executable")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		assert.Equal(t, "/var/lib/minikube/binaries/v1.24.3/kubelet", res[0].Data.Value)
	})

	t.Run("kubelet config file flag", func(t *testing.T) {
		res := x.TestQuery(t, "k8s.kubelet.options[\"config\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "/var/lib/kubelet/config.yaml", res[0].Data.Value)
	})

	t.Run("check for default value", func(t *testing.T) {
		res := x.TestQuery(t, "k8s.kubelet.options[\"volumePluginDir\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/", res[0].Data.Value)
	})

	t.Run("check for config file param", func(t *testing.T) {
		res := x.TestQuery(t, "k8s.kubelet.options[\"healthzBindAddress\"]")
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

		res = x.TestQuery(t, "k8s.kubelet.options[\"runtimeRequestTimeout\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "15m", res[0].Data.Value)
	})
}
