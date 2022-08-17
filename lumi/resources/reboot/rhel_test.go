package reboot

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/providers/mock"
)

func TestRhelKernelLatest(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/redhat_kernel_reboot.toml")
	provider, err := mock.NewFromTomlFile(filepath)
	require.NoError(t, err)

	lb := RpmNewestKernel{provider: provider}
	required, err := lb.RebootPending()
	require.NoError(t, err)
	assert.Equal(t, true, required)
}

func TestAmznContainerWithoutKernel(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/amzn_kernel_container.toml")
	provider, err := mock.NewFromTomlFile(filepath)
	require.NoError(t, err)

	lb := RpmNewestKernel{provider: provider}
	required, err := lb.RebootPending()
	require.NoError(t, err)

	assert.Equal(t, false, required)
}

func TestAmznEc2Kernel(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/amzn_kernel_ec2.toml")
	provider, err := mock.NewFromTomlFile(filepath)
	require.NoError(t, err)

	lb := RpmNewestKernel{provider: provider}
	required, err := lb.RebootPending()
	require.NoError(t, err)

	assert.Equal(t, false, required)
}
