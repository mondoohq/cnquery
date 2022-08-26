package reboot

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers/mock"
)

func TestRebootLinux(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/ubuntu_reboot.toml")
	provider, err := mock.NewFromTomlFile(filepath)
	require.NoError(t, err)

	lb := DebianReboot{provider: provider}
	required, err := lb.RebootPending()
	require.NoError(t, err)
	assert.Equal(t, true, required)
}

func TestNoRebootLinux(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/ubuntu_noreboot.toml")
	provider, err := mock.NewFromTomlFile(filepath)
	require.NoError(t, err)

	lb := DebianReboot{provider: provider}
	required, err := lb.RebootPending()
	require.NoError(t, err)
	assert.Equal(t, false, required)
}
