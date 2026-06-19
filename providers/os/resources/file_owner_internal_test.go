// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/mock"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/utils/syncx"
)

// A file that exists but is owned by a uid/gid with no passwd/group entry
// (common on minimal containers, or files left by a deleted user) must resolve
// file.user / file.group to null and fail cleanly, rather than erroring the
// whole check with "cannot find user for uid N".
func TestFileOwnership_UnknownUidGid(t *testing.T) {
	fixturePath, err := filepath.Abs("testdata/file_orphan_owner.toml")
	require.NoError(t, err)

	asset := &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "centos",
			Family: []string{"linux", "unix"},
		},
	}
	conn, err := mock.New(0, asset, mock.WithPath(fixturePath))
	require.NoError(t, err)

	runtime := &plugin.Runtime{
		Connection: conn,
		Resources:  &syncx.Map[plugin.Resource]{},
	}

	raw, err := CreateResource(runtime, "file", map[string]*llx.RawData{
		"path": llx.StringData("/orphan"),
	})
	require.NoError(t, err)

	file := raw.(*mqlFile)
	// Simulate an existing file owned by a uid/gid that is not in passwd/group.
	file.Exists = plugin.TValue[bool]{Data: true, State: plugin.StateIsSet}
	file.statInfo = &shared.FileInfoDetails{
		Mode: shared.FileModeDetails{FileMode: os.FileMode(0o644)},
		Size: 10,
		Uid:  4242,
		Gid:  4242,
	}

	user := file.GetUser()
	require.NoError(t, user.Error, "file.user must not error for an unknown uid")
	assert.True(t, user.State&plugin.StateIsNull != 0, "file.user should be null for an unknown uid")

	group := file.GetGroup()
	require.NoError(t, group.Error, "file.group must not error for an unknown gid")
	assert.True(t, group.State&plugin.StateIsNull != 0, "file.group should be null for an unknown gid")
}
