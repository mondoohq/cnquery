// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tar_test

import (
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/tar"
)

const (
	alpineImage         = "alpine:3.9"
	alpineContainerPath = "./alpine-container.tar"

	centosImage         = "centos:7"
	centosContainerPath = "./centos-container.tar"
)

func TestTarCommand(t *testing.T) {
	err := cacheAlpine()
	require.NoError(t, err, "should create tar without error")

	c, err := tar.NewTarConnection(0, &inventory.Config{
		Type: "tar",
		Options: map[string]string{
			tar.OPTION_FILE: alpineContainerPath,
		},
	}, &inventory.Asset{})
	assert.Equal(t, nil, err, "should create tar without error")

	cmd, err := c.RunCommand("ls /")
	assert.Nil(t, err)
	if assert.NotNil(t, cmd) {
		assert.Equal(t, nil, err, "should execute without error")
		assert.Equal(t, -1, cmd.ExitStatus, "command should not be executed")
		stdoutContent, _ := io.ReadAll(cmd.Stdout)
		assert.Equal(t, "", string(stdoutContent), "output should be correct")
		stderrContent, _ := io.ReadAll(cmd.Stdout)
		assert.Equal(t, "", string(stderrContent), "output should be correct")
	}
}

func TestPlatformIdentifier(t *testing.T) {
	err := cacheAlpine()
	require.NoError(t, err, "should create tar without error")

	conn, err := tar.NewTarConnection(0, &inventory.Config{
		Type: "tar",
		Options: map[string]string{
			tar.OPTION_FILE: alpineContainerPath,
		},
	}, &inventory.Asset{})
	require.NoError(t, err)
	platformId, err := conn.Identifier()
	require.NoError(t, err)
	assert.True(t, len(platformId) > 0)
	assert.True(t, strings.HasPrefix(platformId, "//platformid.api.mondoo.app/runtime/tar/hash/"))
}

func TestTarSymlinkFile(t *testing.T) {
	err := cacheAlpine()
	require.NoError(t, err, "should create tar without error")

	c, err := tar.NewTarConnection(0, &inventory.Config{
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
		assert.Equal(t, int64(796240), stat.Size(), "should read file size")

		content, err := io.ReadAll(f)
		assert.Equal(t, nil, err, "should execute without error")
		assert.Equal(t, 796240, len(content), "should read the full content")
	}
}

// deactivate test for now for speedier testing
// in contrast to alpine, the symlink on centos is pointing to a relative target and not an absolute one
func TestTarRelativeSymlinkFileCentos(t *testing.T) {
	err := cacheCentos()
	require.NoError(t, err, "should create tar without error")

	c, err := tar.NewTarConnection(0, &inventory.Config{
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

func TestTarFile(t *testing.T) {
	err := cacheAlpine()
	require.NoError(t, err, "should create tar without error")

	c, err := tar.NewTarConnection(0, &inventory.Config{
		Type: "tar",
		Options: map[string]string{
			tar.OPTION_FILE: alpineContainerPath,
		},
	}, &inventory.Asset{})
	assert.Equal(t, nil, err, "should create tar without error")

	f, err := c.FileSystem().Open("/etc/alpine-release")
	assert.Nil(t, err)
	if assert.NotNil(t, f) {
		assert.Equal(t, nil, err, "should execute without error")

		p := f.Name()
		assert.Equal(t, "/etc/alpine-release", p, "path should be correct")

		stat, err := f.Stat()
		assert.Equal(t, int64(6), stat.Size(), "should read file size")
		assert.Equal(t, nil, err, "should execute without error")

		content, err := io.ReadAll(f)
		assert.Equal(t, nil, err, "should execute without error")
		assert.Equal(t, 6, len(content), "should read the full content")
	}
}

func TestFilePermissions(t *testing.T) {
	err := cacheAlpine()
	require.NoError(t, err, "should create tar without error")

	c, err := tar.NewTarConnection(0, &inventory.Config{
		Type: "tar",
		Options: map[string]string{
			tar.OPTION_FILE: alpineContainerPath,
		},
	}, &inventory.Asset{})
	require.NoError(t, err)

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
}

func TestTarFileFind(t *testing.T) {
	err := cacheAlpine()
	require.NoError(t, err, "should create tar without error")

	c, err := tar.NewTarConnection(0, &inventory.Config{
		Type: "tar",
		Options: map[string]string{
			tar.OPTION_FILE: alpineContainerPath,
		},
	}, &inventory.Asset{})
	assert.Equal(t, nil, err, "should create tar without error")

	fs := c.FileSystem()

	fSearch := fs.(*tar.FS)

	infos, err := fSearch.Find("/", regexp.MustCompile(`alpine-release`), "file")
	require.NoError(t, err)

	assert.Equal(t, 1, len(infos))
}

func cacheAlpine() error {
	return cacheImageToTar(alpineImage, alpineContainerPath)
}

func cacheCentos() error {
	return cacheImageToTar(centosImage, centosContainerPath)
}

func cacheImageToTar(source string, filename string) error {
	// check if the cache is already there
	_, err := os.Stat(filename)
	if err == nil {
		return nil
	}

	tag, err := name.NewTag(source, name.WeakValidation)
	if err != nil {
		return err
	}

	auth, err := authn.DefaultKeychain.Resolve(tag.Registry)
	if err != nil {
		return err
	}

	img, err := remote.Image(tag, remote.WithAuth(auth), remote.WithTransport(http.DefaultTransport))
	if err != nil {
		return err
	}

	// it is important that we extract the image here, since tar does not understand the OCI image
	// format and its layers
	w, err := os.Create(filename)
	if err != nil {
		return err
	}

	return tar.StreamToTmpFile(mutate.Extract(img), w)
}
