package reboot

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/providers/mock"
)

func TestRebootOnUbuntu(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/ubuntu_reboot.toml")
	provider, err := mock.NewFromTomlFile(filepath)
	require.NoError(t, err)

	m, err := motor.New(provider)
	require.NoError(t, err)

	lb, err := New(m)
	require.NoError(t, err)

	required, err := lb.RebootPending()
	require.NoError(t, err)
	assert.Equal(t, true, required)
}

func TestRebootOnRhel(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/redhat_kernel_reboot.toml")
	provider, err := mock.NewFromTomlFile(filepath)
	require.NoError(t, err)

	m, err := motor.New(provider)
	require.NoError(t, err)

	lb, err := New(m)
	require.NoError(t, err)

	required, err := lb.RebootPending()
	require.NoError(t, err)

	assert.Equal(t, true, required)
}

func TestRebootOnWindows(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/windows_reboot.toml")
	provider, err := mock.NewFromTomlFile(filepath)
	require.NoError(t, err)

	m, err := motor.New(provider)
	require.NoError(t, err)

	lb, err := New(m)
	require.NoError(t, err)

	required, err := lb.RebootPending()
	require.NoError(t, err)
	assert.Equal(t, true, required)
}
