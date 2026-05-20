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
	"go.mondoo.com/mql/v13/providers/os/connection/mock"
	"go.mondoo.com/mql/v13/utils/syncx"
)

func TestSshdConfigEffectiveAlgorithms(t *testing.T) {
	runtime := sshdEffectiveConfigMockRuntime(t, map[string]*mock.Command{
		sshdEffectiveConfigCommand: {
			Command: sshdEffectiveConfigCommand,
			Stdout: `ciphers chacha20-poly1305@openssh.com,aes256-gcm@openssh.com,aes128-ctr
macs hmac-sha2-512-etm@openssh.com,hmac-sha2-256
kexalgorithms sntrup761x25519-sha512,mlkem768x25519-sha256,curve25519-sha256
`,
			ExitStatus: 0,
		},
	})

	raw, err := CreateResource(runtime, ResourceSshdConfig, nil)
	require.NoError(t, err)

	config := raw.(*mqlSshdConfig)
	ciphers := config.GetEffectiveCiphers()
	require.NoError(t, ciphers.Error)
	assert.Equal(t, []any{"chacha20-poly1305@openssh.com", "aes256-gcm@openssh.com", "aes128-ctr"}, ciphers.Data)

	macs := config.GetEffectiveMacs()
	require.NoError(t, macs.Error)
	assert.Equal(t, []any{"hmac-sha2-512-etm@openssh.com", "hmac-sha2-256"}, macs.Data)

	kexs := config.GetEffectiveKexs()
	require.NoError(t, kexs.Error)
	assert.Equal(t, []any{"sntrup761x25519-sha512", "mlkem768x25519-sha256", "curve25519-sha256"}, kexs.Data)
}

func TestSshdConfigEffectiveAlgorithmsCustomPath(t *testing.T) {
	command := sshdEffectiveConfigCommand + " -f '/tmp/sshd config'"
	runtime := sshdEffectiveConfigMockRuntime(t, map[string]*mock.Command{
		command: {
			Command:    command,
			Stdout:     "ciphers aes256-gcm@openssh.com,aes128-gcm@openssh.com\n",
			ExitStatus: 0,
		},
	})

	raw, err := NewResource(runtime, ResourceSshdConfig, map[string]*llx.RawData{
		"path": llx.StringData("/tmp/sshd config"),
	})
	require.NoError(t, err)

	config := raw.(*mqlSshdConfig)
	ciphers := config.GetEffectiveCiphers()
	require.NoError(t, ciphers.Error)
	assert.Equal(t, []any{"aes256-gcm@openssh.com", "aes128-gcm@openssh.com"}, ciphers.Data)
}

func TestSshdConfigEffectiveAlgorithmsCommandFailure(t *testing.T) {
	runtime := sshdEffectiveConfigMockRuntime(t, map[string]*mock.Command{
		sshdEffectiveConfigCommand: {
			Command:    sshdEffectiveConfigCommand,
			Stderr:     "bad sshd configuration",
			ExitStatus: 255,
		},
	})

	raw, err := CreateResource(runtime, ResourceSshdConfig, nil)
	require.NoError(t, err)

	config := raw.(*mqlSshdConfig)
	ciphers := config.GetEffectiveCiphers()
	require.ErrorContains(t, ciphers.Error, "sshd -T failed (exit 255): bad sshd configuration")
}

func sshdEffectiveConfigMockRuntime(t *testing.T, commands map[string]*mock.Command) *plugin.Runtime {
	t.Helper()

	asset := &inventory.Asset{
		Platform: &inventory.Platform{
			Name:    "linux",
			Family:  []string{"linux", "unix", "os"},
			Version: "test",
		},
	}
	conn, err := mock.New(0, asset, mock.WithData(&mock.TomlData{
		Commands: commands,
		Files:    map[string]*mock.MockFileData{},
	}))
	require.NoError(t, err)

	return &plugin.Runtime{
		Connection: conn,
		Resources:  &syncx.Map[plugin.Resource]{},
	}
}
