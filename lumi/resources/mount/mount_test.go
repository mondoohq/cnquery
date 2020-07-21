package mount_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/lumi/resources/mount"
	"go.mondoo.io/mondoo/motor/motorapi"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestMountLinuxParser(t *testing.T) {
	mock, err := mock.NewFromToml(&motorapi.TransportConfig{Backend: motorapi.TransportBackend_CONNECTION_MOCK, Path: "./testdata/debian.toml"})
	require.NoError(t, err)

	f, err := mock.RunCommand("mount")
	require.NoError(t, err)

	entries := mount.ParseLinuxMountCmd(f.Stdout)
	assert.Equal(t, 25, len(entries))

	// /dev/sda1 on / type ext4 (rw,relatime,data=ordered)
	expected := &mount.MountPoint{
		Device:     "/dev/sda1",
		MountPoint: "/",
		FSType:     "ext4",
		Options: map[string]string{
			"rw":       "",
			"relatime": "",
			"data":     "ordered",
		},
	}
	found := findMountpoint(entries, "/")
	assert.True(t, cmp.Equal(expected, found), cmp.Diff(expected, found))
}

func TestMountMacosParser(t *testing.T) {
	mock, err := mock.NewFromToml(&motorapi.TransportConfig{Backend: motorapi.TransportBackend_CONNECTION_MOCK, Path: "./testdata/osx.toml"})
	require.NoError(t, err)

	f, err := mock.RunCommand("mount")
	require.NoError(t, err)

	entries := mount.ParseUnixMountCmd(f.Stdout)
	// NOTE: we do not handle `map auto_home` yet
	assert.Equal(t, 4, len(entries))

	expected := &mount.MountPoint{
		Device:     "/dev/disk1s5",
		MountPoint: "/",
		FSType:     "apfs",
		Options: map[string]string{
			"apfs":      "",
			"local":     "",
			"read-only": "",
			"journaled": "",
		},
	}
	found := findMountpoint(entries, "/")
	assert.True(t, cmp.Equal(expected, found), cmp.Diff(expected, found))
}

func TestMountFreeBsdParser(t *testing.T) {
	mock, err := mock.NewFromToml(&motorapi.TransportConfig{Backend: motorapi.TransportBackend_CONNECTION_MOCK, Path: "./testdata/freebsd12.toml"})
	require.NoError(t, err)

	f, err := mock.RunCommand("mount")
	require.NoError(t, err)

	entries := mount.ParseUnixMountCmd(f.Stdout)
	assert.Equal(t, 2, len(entries))

	expected := &mount.MountPoint{
		Device:     "/dev/gpt/rootfs",
		MountPoint: "/",
		FSType:     "ufs",
		Options: map[string]string{
			"ufs":          "",
			"local":        "",
			"soft-updates": "",
		},
	}
	found := findMountpoint(entries, "/")
	assert.True(t, cmp.Equal(expected, found), cmp.Diff(expected, found))
}

func TestProcModulesParser(t *testing.T) {
	mock, err := mock.NewFromToml(&motorapi.TransportConfig{Backend: motorapi.TransportBackend_CONNECTION_MOCK, Path: "./testdata/debian.toml"})
	require.NoError(t, err)

	f, err := mock.FS().Open("/proc/mounts")
	require.NoError(t, err)
	defer f.Close()

	entries := mount.ParseLinuxProcMount(f)
	assert.Equal(t, 25, len(entries))

	// /dev/sda1 on / type ext4 (rw,relatime,data=ordered)
	expected := &mount.MountPoint{
		Device:     "/dev/sda1",
		MountPoint: "/",
		FSType:     "ext4",
		Options: map[string]string{
			"rw":       "",
			"relatime": "",
			"data":     "ordered",
		},
	}
	found := findMountpoint(entries, "/")
	assert.True(t, cmp.Equal(expected, found), cmp.Diff(expected, found))
}

func findMountpoint(mounts []mount.MountPoint, name string) *mount.MountPoint {
	for i := range mounts {
		if mounts[i].MountPoint == name {
			return &mounts[i]
		}
	}
	return nil
}
