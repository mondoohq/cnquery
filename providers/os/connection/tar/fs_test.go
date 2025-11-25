// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tar_test

import (
	"io"
	"regexp"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers/os/connection/tar"
)

// deactivate test for now for speedier testing
// in contrast to alpine, the symlink on centos is pointing to a relative target and not an absolute one
func TestTarRelativeSymlinkFileCentos(t *testing.T) {
	err := cacheCentos()
	require.NoError(t, err, "should create tar without error")

	c, err := tar.NewConnection(0, &inventory.Config{
		Type: "tar",
		Options: map[string]string{
			tar.OPTION_FILE: centosContainerPath,
		},
	}, &inventory.Asset{})
	assert.Equal(t, nil, err, "should create tar without error")

	f, err := c.FileSystem().Open("/etc/redhat-release")
	require.NoError(t, err)

	if assert.NotNil(t, f) {
		assert.Equal(t, nil, err, "should execute without error")

		p := f.Name()
		assert.Equal(t, "/etc/redhat-release", p, "path should be correct")

		stat, err := f.Stat()
		assert.Equal(t, nil, err, "should stat without error")
		assert.Equal(t, int64(37), stat.Size(), "should read file size")

		content, err := io.ReadAll(f)
		assert.Equal(t, nil, err, "should execute without error")
		assert.Equal(t, 37, len(content), "should read the full content")
	}
}

func TestTarFileAlpine(t *testing.T) {
	err := cacheAlpine()
	require.NoError(t, err, "should create tar without error")

	c, err := tar.NewConnection(0, &inventory.Config{
		Type: "tar",
		Options: map[string]string{
			tar.OPTION_FILE: alpineContainerPath,
		},
	}, &inventory.Asset{})
	assert.Equal(t, nil, err, "should create tar without error")

	t.Run("test file content", func(t *testing.T) {
		f, err := c.FileSystem().Open("/etc/alpine-release")
		assert.Nil(t, err)
		if assert.NotNil(t, f) {
			assert.Equal(t, nil, err, "should execute without error")

			p := f.Name()
			assert.Equal(t, "/etc/alpine-release", p, "path should be correct")

			stat, err := f.Stat()
			assert.Equal(t, int64(8), stat.Size(), "should read file size")
			assert.Equal(t, nil, err, "should execute without error")

			content, err := io.ReadAll(f)
			assert.Equal(t, nil, err, "should execute without error")
			assert.Equal(t, 8, len(content), "should read the full content")
		}
	})

	t.Run("test file permissions", func(t *testing.T) {
		path := "/etc/alpine-release"
		details, err := c.FileInfo(path)
		require.NoError(t, err)
		assert.Equal(t, int64(0), details.Uid)
		assert.Equal(t, int64(0), details.Gid)
		assert.True(t, details.Size >= 0)
		assert.Equal(t, false, details.Mode.IsDir())
		assert.Equal(t, true, details.Mode.IsRegular())
		assert.Equal(t, "-rw-r--r--", details.Mode.String())
		assert.True(t, details.Mode.UserReadable())
		assert.True(t, details.Mode.UserWriteable())
		assert.False(t, details.Mode.UserExecutable())
		assert.True(t, details.Mode.GroupReadable())
		assert.False(t, details.Mode.GroupWriteable())
		assert.False(t, details.Mode.GroupExecutable())
		assert.True(t, details.Mode.OtherReadable())
		assert.False(t, details.Mode.OtherWriteable())
		assert.False(t, details.Mode.OtherExecutable())
		assert.False(t, details.Mode.Suid())
		assert.False(t, details.Mode.Sgid())
		assert.False(t, details.Mode.Sticky())

		path = "/etc"
		details, err = c.FileInfo(path)
		require.NoError(t, err)
		assert.Equal(t, int64(0), details.Uid)
		assert.Equal(t, int64(0), details.Gid)
		assert.True(t, details.Size >= 0)
		assert.True(t, details.Mode.IsDir())
		assert.False(t, details.Mode.IsRegular())
		assert.Equal(t, "drwxr-xr-x", details.Mode.String())
		assert.True(t, details.Mode.UserReadable())
		assert.True(t, details.Mode.UserWriteable())
		assert.True(t, details.Mode.UserExecutable())
		assert.True(t, details.Mode.GroupReadable())
		assert.False(t, details.Mode.GroupWriteable())
		assert.True(t, details.Mode.GroupExecutable())
		assert.True(t, details.Mode.OtherReadable())
		assert.False(t, details.Mode.OtherWriteable())
		assert.True(t, details.Mode.OtherExecutable())
		assert.False(t, details.Mode.Suid())
		assert.False(t, details.Mode.Sgid())
		assert.False(t, details.Mode.Sticky())
	})

	t.Run("test symlink", func(t *testing.T) {
		c, err := tar.NewConnection(0, &inventory.Config{
			Type: "tar",
			Options: map[string]string{
				tar.OPTION_FILE: alpineContainerPath,
			},
		}, &inventory.Asset{})
		assert.Equal(t, nil, err, "should create tar without error")

		f, err := c.FileSystem().Open("/bin/cat")
		assert.Nil(t, err)
		if assert.NotNil(t, f) {
			assert.Equal(t, nil, err, "should execute without error")

			p := f.Name()
			assert.Equal(t, "/bin/cat", p, "path should be correct")

			stat, err := f.Stat()
			assert.Equal(t, nil, err, "should stat without error")
			assert.Equal(t, int64(829000), stat.Size(), "should read file size")

			content, err := io.ReadAll(f)
			assert.Equal(t, nil, err, "should execute without error")
			assert.Equal(t, 829000, len(content), "should read the full content")
		}
	})

	t.Run("test file search", func(t *testing.T) {
		fs := c.FileSystem()
		fSearch := fs.(*tar.FS)
		infos, err := fSearch.Find("/", regexp.MustCompile(`alpine-release`), "file", nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 1, len(infos))
	})

	t.Run("test file readdirnames", func(t *testing.T) {
		fs := c.FileSystem()
		tarFs := fs.(*tar.FS)
		d, err := tarFs.Open("/etc/apk")
		require.NoError(t, err)
		defer d.Close()
		names, err := d.Readdirnames(-1)
		require.NoError(t, err)
		assert.Equal(t, 5, len(names))
		sort.Strings(names)
		assert.Equal(t, []string{"arch", "keys", "protected_paths.d", "repositories", "world"}, names)
	})
}
