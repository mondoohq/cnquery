// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/os/connection/mock"
	"go.mondoo.com/mql/utils/syncx"
)

func TestExpandSshdGlob(t *testing.T) {
	fs := afero.NewMemMapFs()
	files := []string{
		"/etc/ssh/sshd_config",
		"/etc/ssh/decoy.conf",
		"/etc/ssh/sshd_config.d/10-a.conf",
		"/etc/ssh/sshd_config.d/20-b.conf",
		"/etc/ssh/sshd_config.d/nested/deep.conf",
	}
	for _, f := range files {
		require.NoError(t, afero.WriteFile(fs, f, []byte("Port 22\n"), 0o644))
	}
	afs := &afero.Afero{Fs: fs}

	tests := []struct {
		name string
		glob string
		want []string
	}{
		{
			name: "non-glob absolute path is returned as-is",
			glob: "/etc/ssh/sshd_config",
			want: []string{"/etc/ssh/sshd_config"},
		},
		{
			name: "non-glob relative path resolves from /etc/ssh",
			glob: "sshd_config",
			want: []string{"/etc/ssh/sshd_config"},
		},
		{
			name: "absolute glob in a subdirectory",
			glob: "/etc/ssh/sshd_config.d/*.conf",
			want: []string{"/etc/ssh/sshd_config.d/10-a.conf", "/etc/ssh/sshd_config.d/20-b.conf"},
		},
		{
			// Regression: a relative Include glob with a subdirectory must not
			// drop the subdirectory segment and glob one level too shallow.
			name: "relative glob in a subdirectory",
			glob: "sshd_config.d/*.conf",
			want: []string{"/etc/ssh/sshd_config.d/10-a.conf", "/etc/ssh/sshd_config.d/20-b.conf"},
		},
		{
			// Regression: a single-segment relative glob must expand within
			// /etc/ssh, not return the directory itself.
			name: "relative single-segment glob",
			glob: "*.conf",
			want: []string{"/etc/ssh/decoy.conf"},
		},
		{
			name: "glob does not descend into subdirectories",
			glob: "/etc/ssh/*.conf",
			want: []string{"/etc/ssh/decoy.conf"},
		},
		{
			name: "glob against a missing directory yields no matches",
			glob: "/etc/does-not-exist/*.conf",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expandSshdGlob(afs, tt.glob)
			require.NoError(t, err)
			require.ElementsMatch(t, tt.want, got)
		})
	}
}

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
