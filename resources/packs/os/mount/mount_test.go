package mount_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/resources/packs/os/mount"
	"go.mondoo.com/cnquery/motor/providers/mock"
)

func TestMountLinuxParser(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/debian.toml")
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
	mock, err := mock.NewFromTomlFile("./testdata/osx.toml")
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
	mock, err := mock.NewFromTomlFile("./testdata/freebsd12.toml")
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
	mock, err := mock.NewFromTomlFile("./testdata/debian.toml")
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

var fstabExample = `
# 
# /etc/fstab: static file system information
#
# <file system>	<dir>	<type>	<options>	<dump>	<pass>
# /dev/sdc2
UUID=6c44ec5a-4727-47d4-b485-81cff72b207e	/         	ext4      	rw,relatime,data=ordered	0 1

# /dev/sdc1
UUID=0EC7-F4C1      	/boot     	vfat      	rw,relatime,fmask=0022,dmask=0022,iocharset=iso8859-1	0 2

UUID=6060df9a-7e53-439c-9189-ba9657161fd4       /data           btrfs           rw,nofail              0 2
`

func TestFstab(t *testing.T) {
	r := strings.NewReader(fstabExample)

	entries, err := mount.ParseFstab(r)
	require.NoError(t, err)

	// /dev/sda1 on / type ext4 (rw,relatime,data=ordered)
	expected := []mount.MountPoint{
		{
			Device:     "UUID=6c44ec5a-4727-47d4-b485-81cff72b207e",
			MountPoint: "/",
			FSType:     "ext4",
			Options: map[string]string{
				"rw":       "",
				"relatime": "",
				"data":     "ordered",
			},
		},
		{
			Device:     "UUID=0EC7-F4C1",
			MountPoint: "/boot",
			FSType:     "vfat",
			Options: map[string]string{
				"rw":        "",
				"relatime":  "",
				"fmask":     "0022",
				"dmask":     "0022",
				"iocharset": "iso8859-1",
			},
		},
		{
			Device:     "UUID=6060df9a-7e53-439c-9189-ba9657161fd4",
			MountPoint: "/data",
			FSType:     "btrfs",
			Options: map[string]string{
				"rw":     "",
				"nofail": "",
			},
		},
	}

	assert.Equal(t, expected, entries)
}
