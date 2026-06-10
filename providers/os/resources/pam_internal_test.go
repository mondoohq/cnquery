// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/mock"
	"go.mondoo.com/mql/v13/providers/os/resources/filesfind"
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

func TestPamConfPrefersPamDirOverPamConf(t *testing.T) {
	// When /etc/pam.d exists, Linux-PAM ignores /etc/pam.conf entirely. Our
	// parsing must do the same: only the pam.d files are read, and a service
	// defined only in /etc/pam.conf must not appear.
	findCmd := filesfind.BuildFilesFindCmd(defaultPamDir, false, "file", "", 0, "", nil)
	conn, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{Name: "arch", Family: []string{"arch", "linux", "unix"}},
	}, mock.WithData(&mock.TomlData{
		Files: map[string]*mock.MockFileData{
			defaultPamDir:         {Path: defaultPamDir, StatData: mock.FileInfo{Mode: os.ModeDir | 0o755}},
			defaultPamDir + "/su": {Path: defaultPamDir + "/su", Content: "auth required pam_wheel.so use_uid\n"},
			defaultPamConf:        {Path: defaultPamConf, Content: "login auth required pam_unix.so\n"},
		},
		Commands: map[string]*mock.Command{
			findCmd: {Stdout: defaultPamDir + "/su\n"},
		},
	}))
	require.NoError(t, err)
	rt := &plugin.Runtime{Connection: conn, Resources: &syncx.Map[plugin.Resource]{}}

	pam := &mqlPamConf{MqlRuntime: rt}

	files := pam.GetFiles()
	require.NoError(t, files.Error)
	paths := make([]string, 0, len(files.Data))
	for _, f := range files.Data {
		paths = append(paths, f.(*mqlFile).Path.Data)
	}
	assert.Equal(t, []string{defaultPamDir + "/su"}, paths,
		"only /etc/pam.d files are read; /etc/pam.conf is ignored")

	entries := pam.GetEntries()
	require.NoError(t, entries.Error)
	_, hasPamDirService := entries.Data[defaultPamDir+"/su"]
	assert.True(t, hasPamDirService, "the pam.d service is parsed")
	_, hasPamConfService := entries.Data["login"]
	assert.False(t, hasPamConfService,
		"the /etc/pam.conf service must not be parsed while /etc/pam.d exists")
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
