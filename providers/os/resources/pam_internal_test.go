// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/mock"
	"go.mondoo.com/mql/v13/utils/syncx"
)

// newPamRuntime builds a runtime backed by a mock filesystem containing the
// given files, so pam.conf.exists can be exercised for both the present and
// absent cases without disturbing the shared LinuxMock recording.
func newPamRuntime(t *testing.T, files ...string) *plugin.Runtime {
	t.Helper()

	fileData := map[string]*mock.MockFileData{}
	for _, f := range files {
		fileData[f] = &mock.MockFileData{Path: f}
	}

	conn, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{Name: "arch", Family: []string{"arch", "linux"}},
	}, mock.WithData(&mock.TomlData{Files: fileData}))
	require.NoError(t, err)

	return &plugin.Runtime{
		Connection: conn,
		Resources:  &syncx.Map[plugin.Resource]{},
	}
}

func TestPamConfExists(t *testing.T) {
	t.Run("true when /etc/pam.conf is present", func(t *testing.T) {
		pam := &mqlPamConf{MqlRuntime: newPamRuntime(t, "/etc/pam.conf")}
		got, err := pam.exists()
		require.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("true when the /etc/pam.d directory is present", func(t *testing.T) {
		pam := &mqlPamConf{MqlRuntime: newPamRuntime(t, "/etc/pam.d")}
		got, err := pam.exists()
		require.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("false without erroring when no PAM config is present", func(t *testing.T) {
		pam := &mqlPamConf{MqlRuntime: newPamRuntime(t)}
		got, err := pam.exists()
		require.NoError(t, err)
		assert.False(t, got)
	})
}

func TestPamConfServiceEntryParams(t *testing.T) {
	se := &mqlPamConfServiceEntry{}

	t.Run("key=value and bare flags", func(t *testing.T) {
		got, err := se.params([]any{"use_uid", "group=wheel"})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"use_uid": "", "group": "wheel"}, got)
	})

	t.Run("duplicate keys: last occurrence wins", func(t *testing.T) {
		got, err := se.params([]any{"group=wheel", "group=admin"})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"group": "admin"}, got)
	})

	t.Run("no options yields an empty map", func(t *testing.T) {
		got, err := se.params([]any{})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{}, got)
	})
}
