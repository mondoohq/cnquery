// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/os/connection/mock"
)

func TestParseFipsEnabled(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantEnabled bool
		wantOk      bool
	}{
		{"enabled", "1", true, true},
		{"enabled with newline", "1\n", true, true},
		{"enabled with whitespace", "  1  \n", true, true},
		{"disabled", "0", false, true},
		{"disabled with newline", "0\n", false, true},
		// An empty or unexpected file has not said whether FIPS is on. Reading
		// "off" out of it is the same invention as reading it out of a file
		// that was never there.
		{"empty is not a reading", "", false, false},
		{"whitespace only is not a reading", "   \n", false, false},
		{"unexpected value is not a reading", "2", false, false},
		{"text is not a reading", "enabled\n", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enabled, ok := parseFipsEnabled(tt.content)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.wantEnabled, enabled)
		})
	}
}

func TestNormalizeCryptoPolicy(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
		want   string
	}{
		{"default with newline", "DEFAULT\n", "DEFAULT"},
		{"future no newline", "FUTURE", "FUTURE"},
		{"subpolicy kept as-is", "FIPS:OSPP", "FIPS:OSPP"},
		{"legacy with whitespace", "  LEGACY  \n", "LEGACY"},
		{"empty", "", ""},
		{"whitespace only", "   \n", ""},
		{"multiline takes first line", "DEFAULT\nsome warning\n", "DEFAULT"},
		{"fips", "FIPS\n", "FIPS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeCryptoPolicy(tt.stdout))
		})
	}
}

// mockOs builds an os resource over a mock connection serving the given files
// and commands.
func mockOs(t *testing.T, data *mock.TomlData) *mqlOs {
	t.Helper()
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(data))
	require.NoError(t, err)
	return &mqlOs{MqlRuntime: &plugin.Runtime{Connection: conn}}
}

func fipsFile(content string) *mock.TomlData {
	return &mock.TomlData{
		Files: map[string]*mock.MockFileData{
			"/proc/sys/crypto/fips_enabled": {
				Path:    "/proc/sys/crypto/fips_enabled",
				Content: content,
			},
		},
	}
}

func TestOsFipsEnabled(t *testing.T) {
	t.Run("reports true when the sysctl says 1", func(t *testing.T) {
		os := mockOs(t, fipsFile("1\n"))

		enabled, err := os.fipsEnabled()

		require.NoError(t, err)
		assert.True(t, enabled)
		assert.False(t, os.FipsEnabled.IsNull(), "a real reading must not be null")
	})

	t.Run("reports false when the sysctl says 0", func(t *testing.T) {
		os := mockOs(t, fipsFile("0\n"))

		enabled, err := os.fipsEnabled()

		require.NoError(t, err)
		assert.False(t, enabled)
		assert.False(t, os.FipsEnabled.IsNull(), "a real reading must not be null")
	})

	// The case this whole file exists for. On a host with no procfs the file is
	// simply not there, and "FIPS is off" is not something that absence
	// establishes. Reporting false here would let a "FIPS must be enabled"
	// audit render a verdict about a host nobody read.
	t.Run("reports null when the sysctl is absent", func(t *testing.T) {
		os := mockOs(t, &mock.TomlData{})

		_, err := os.fipsEnabled()

		require.NoError(t, err)
		assert.True(t, os.FipsEnabled.IsNull(), "an unread host must be null, not false")
	})

	t.Run("reports null when the sysctl holds something else", func(t *testing.T) {
		os := mockOs(t, fipsFile("who knows\n"))

		_, err := os.fipsEnabled()

		require.NoError(t, err)
		assert.True(t, os.FipsEnabled.IsNull())
	})

	// A runtime with no OS connection behind it cannot have read anything.
	t.Run("reports null without an os connection", func(t *testing.T) {
		os := &mqlOs{MqlRuntime: &plugin.Runtime{}}

		_, err := os.fipsEnabled()

		require.NoError(t, err)
		assert.True(t, os.FipsEnabled.IsNull())
	})
}

func cryptoPolicyCmd(cmd *mock.Command) *mock.TomlData {
	return &mock.TomlData{
		Commands: map[string]*mock.Command{"update-crypto-policies --show": cmd},
	}
}

func TestOsCryptoPolicy(t *testing.T) {
	t.Run("reports the policy the tool prints", func(t *testing.T) {
		os := mockOs(t, cryptoPolicyCmd(&mock.Command{Stdout: "FIPS:OSPP\n"}))

		policy, err := os.cryptoPolicy()

		require.NoError(t, err)
		assert.Equal(t, "FIPS:OSPP", policy)
		assert.False(t, os.CryptoPolicy.IsNull(), "a real reading must not be null")
	})

	// Debian, Ubuntu, macOS and Windows ship no update-crypto-policies. Their
	// crypto policy is not the empty string, it is unknown to us, and a check
	// comparing against a policy name cannot tell those apart.
	t.Run("reports null when the tool is missing", func(t *testing.T) {
		os := mockOs(t, cryptoPolicyCmd(&mock.Command{
			Stderr:     "sh: update-crypto-policies: command not found",
			ExitStatus: 127,
		}))

		_, err := os.cryptoPolicy()

		require.NoError(t, err)
		assert.True(t, os.CryptoPolicy.IsNull(), "a failed command must be null, not \"\"")
	})

	t.Run("reports null when the command is not in the mock at all", func(t *testing.T) {
		os := mockOs(t, &mock.TomlData{})

		_, err := os.cryptoPolicy()

		require.NoError(t, err)
		assert.True(t, os.CryptoPolicy.IsNull())
	})

	t.Run("reports null when the tool prints nothing", func(t *testing.T) {
		os := mockOs(t, cryptoPolicyCmd(&mock.Command{Stdout: "\n"}))

		_, err := os.cryptoPolicy()

		require.NoError(t, err)
		assert.True(t, os.CryptoPolicy.IsNull())
	})

	t.Run("reports null without an os connection", func(t *testing.T) {
		os := &mqlOs{MqlRuntime: &plugin.Runtime{}}

		_, err := os.cryptoPolicy()

		require.NoError(t, err)
		assert.True(t, os.CryptoPolicy.IsNull())
	})
}
