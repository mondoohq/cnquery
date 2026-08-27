// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"k8s.io/utils/ptr"
)

func TestFstabEntries(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		testdata := `# <device>                                <dir> <type> <options> <dump> <fsck>
UUID=0a3407de-014b-458b-b5c1-848e92a327a3 /     ext4   defaults  0      1
UUID=f9fe0b69-a280-415d-a03a-a32752370dee none  swap   defaults  0      0
UUID=b411dc99-f0a0-4c87-9e05-184977be8539 /home ext4   defaults  0      2`

		reader := strings.NewReader(testdata)
		entries, err := ParseFstab(reader)

		require.NoError(t, err)
		require.Len(t, entries, 3)

		require.Equal(t, FstabEntry{
			Device:     "UUID=0a3407de-014b-458b-b5c1-848e92a327a3",
			Mountpoint: "/",
			Fstype:     "ext4",
			Options:    []string{"defaults"},
			Dump:       ptr.To(0),
			Fsck:       ptr.To(1),
		}, entries[0])
		require.Equal(t, FstabEntry{
			Device:     "UUID=f9fe0b69-a280-415d-a03a-a32752370dee",
			Mountpoint: "none",
			Fstype:     "swap",
			Options:    []string{"defaults"},
			Dump:       ptr.To(0),
			Fsck:       ptr.To(0),
		}, entries[1])
		require.Equal(t, FstabEntry{
			Device:     "UUID=b411dc99-f0a0-4c87-9e05-184977be8539",
			Mountpoint: "/home",
			Fstype:     "ext4",
			Options:    []string{"defaults"},
			Dump:       ptr.To(0),
			Fsck:       ptr.To(2),
		}, entries[2])
	})

	t.Run("short", func(t *testing.T) {
		testdata := `# <device>                                <dir> <type> <options>
UUID=0a3407de-014b-458b-b5c1-848e92a327a3 /     ext4   defaults
UUID=f9fe0b69-a280-415d-a03a-a32752370dee none  swap   defaults
UUID=b411dc99-f0a0-4c87-9e05-184977be8539 /home ext4   defaults`

		reader := strings.NewReader(testdata)
		entries, err := ParseFstab(reader)

		require.NoError(t, err)
		require.Len(t, entries, 3)

		require.Equal(t, FstabEntry{
			Device:     "UUID=0a3407de-014b-458b-b5c1-848e92a327a3",
			Mountpoint: "/",
			Fstype:     "ext4",
			Options:    []string{"defaults"},
		}, entries[0])
		require.Equal(t, FstabEntry{
			Device:     "UUID=f9fe0b69-a280-415d-a03a-a32752370dee",
			Mountpoint: "none",
			Fstype:     "swap",
			Options:    []string{"defaults"},
		}, entries[1])
		require.Equal(t, FstabEntry{
			Device:     "UUID=b411dc99-f0a0-4c87-9e05-184977be8539",
			Mountpoint: "/home",
			Fstype:     "ext4",
			Options:    []string{"defaults"},
		}, entries[2])
	})

	t.Run("valid (with tabs)", func(t *testing.T) {
		testdata := `# <device>                                <dir> <type> <options> <dump> <fsck>
LABEL=cloudimg-rootfs	/	 ext4	discard,commit=30,errors=remount-ro	0 1
LABEL=BOOT	/boot	ext4	defaults	0 2
LABEL=UEFI	/boot/efi	vfat	umask=0077	0 1`

		reader := strings.NewReader(testdata)
		entries, err := ParseFstab(reader)

		require.NoError(t, err)
		require.Len(t, entries, 3)

		require.Equal(t, FstabEntry{
			Device:     "LABEL=cloudimg-rootfs",
			Mountpoint: "/",
			Fstype:     "ext4",
			Options:    []string{"discard", "commit=30", "errors=remount-ro"},
			Dump:       ptr.To(0),
			Fsck:       ptr.To(1),
		}, entries[0])
		require.Equal(t, FstabEntry{
			Device:     "LABEL=BOOT",
			Mountpoint: "/boot",
			Fstype:     "ext4",
			Options:    []string{"defaults"},
			Dump:       ptr.To(0),
			Fsck:       ptr.To(2),
		}, entries[1])
		require.Equal(t, FstabEntry{
			Device:     "LABEL=UEFI",
			Mountpoint: "/boot/efi",
			Fstype:     "vfat",
			Options:    []string{"umask=0077"},
			Dump:       ptr.To(0),
			Fsck:       ptr.To(1),
		}, entries[2])
	})

	t.Run("invalid (too short)", func(t *testing.T) {
		testdata := `# <device>                                <dir> <type>
UUID=0a3407de-014b-458b-b5c1-848e92a327a3 /     ext4
UUID=f9fe0b69-a280-415d-a03a-a32752370dee none  swap
UUID=b411dc99-f0a0-4c87-9e05-184977be8539 /home ext4`

		reader := strings.NewReader(testdata)
		entries, err := ParseFstab(reader)

		require.Error(t, err)
		require.Nil(t, entries)
	})

	t.Run("invalid (not numeric dump)", func(t *testing.T) {
		testdata := `# <device>                                <dir> <type> <options> <dump> <fsck>
UUID=0a3407de-014b-458b-b5c1-848e92a327a3 /     ext4   defaults  0      1
UUID=f9fe0b69-a280-415d-a03a-a32752370dee none  swap   defaults  0      0
UUID=b411dc99-f0a0-4c87-9e05-184977be8539 /home ext4   defaults  A      2` // note the 'A' here

		reader := strings.NewReader(testdata)
		entries, err := ParseFstab(reader)

		require.Error(t, err)
		require.Nil(t, entries)
	})

	t.Run("invalid (not numeric fsck)", func(t *testing.T) {
		testdata := `# <device>                                <dir> <type> <options> <dump> <fsck>
UUID=0a3407de-014b-458b-b5c1-848e92a327a3 /     ext4   defaults  0      1
UUID=f9fe0b69-a280-415d-a03a-a32752370dee none  swap   defaults  0      0
UUID=b411dc99-f0a0-4c87-9e05-184977be8539 /home ext4   defaults  0      A` // note the 'A' here

		reader := strings.NewReader(testdata)
		entries, err := ParseFstab(reader)

		require.Error(t, err)
		require.Nil(t, entries)
	})

	t.Run("indented comments and blank lines are skipped", func(t *testing.T) {
		testdata := "# leading comment\n" +
			"\t# indented comment with a tab\n" +
			"   # indented comment with spaces\n" +
			"   \t  \n" + // whitespace-only line
			"\n" + // empty line
			"  UUID=0a3407de-014b-458b-b5c1-848e92a327a3 / ext4 defaults 0 1\n"

		reader := strings.NewReader(testdata)
		entries, err := ParseFstab(reader)

		require.NoError(t, err)
		require.Len(t, entries, 1)
		require.Equal(t, FstabEntry{
			Device:     "UUID=0a3407de-014b-458b-b5c1-848e92a327a3",
			Mountpoint: "/",
			Fstype:     "ext4",
			Options:    []string{"defaults"},
			Dump:       ptr.To(0),
			Fsck:       ptr.To(1),
		}, entries[0])
	})
}

func fstabEntry(device, mountpoint string) *mqlFstabEntry {
	return &mqlFstabEntry{
		Device:     plugin.TValue[string]{Data: device, State: plugin.StateIsSet},
		Mountpoint: plugin.TValue[string]{Data: mountpoint, State: plugin.StateIsSet},
	}
}

// Two rows sharing a device is ordinary in /etc/fstab. When their __id
// collides the runtime serves the first for both, and the later rows drop out
// of fstab.entries without any error.
func TestFstabEntryIDIsUniquePerMountPoint(t *testing.T) {
	tests := []struct {
		name string
		a    *mqlFstabEntry
		b    *mqlFstabEntry
	}{
		{
			// The reported case, and the stock layout on many distros.
			name: "two tmpfs mounts",
			a:    fstabEntry("tmpfs", "/tmp"),
			b:    fstabEntry("tmpfs", "/dev/shm"),
		},
		{
			name: "two swap devices both mounting at none",
			a:    fstabEntry("/dev/sda2", "none"),
			b:    fstabEntry("/dev/sda3", "none"),
		},
		{
			name: "two none-device pseudo filesystems",
			a:    fstabEntry("none", "/proc/sys/fs/binfmt_misc"),
			b:    fstabEntry("none", "/sys/kernel/debug"),
		},
		{
			name: "same mount point, different device",
			a:    fstabEntry("/dev/sdb1", "/data"),
			b:    fstabEntry("/dev/sdc1", "/data"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ida, err := tc.a.id()
			assert.NoError(t, err)
			idb, err := tc.b.id()
			assert.NoError(t, err)
			assert.NotEqual(t, ida, idb,
				"distinct fstab rows must not share an __id, or the later row is dropped")
		})
	}
}

func TestFstabEntryIDIsStable(t *testing.T) {
	e := fstabEntry("tmpfs", "/dev/shm")
	first, err := e.id()
	assert.NoError(t, err)
	second, err := e.id()
	assert.NoError(t, err)
	assert.Equal(t, first, second, "__id must be stable across calls")
	assert.Contains(t, first, "tmpfs")
	assert.Contains(t, first, "/dev/shm")
}

// The fstab resource is selected by path, so its __id has to carry that path.
// Without an id() every fstab shares the empty cache key and the second
// fstab(...) in a query resolves to the first one's file.
func TestFstabIDIsPerFile(t *testing.T) {
	mk := func(path string) *mqlFstab {
		return &mqlFstab{Path: plugin.TValue[string]{Data: path, State: plugin.StateIsSet}}
	}

	etc, err := mk("/etc/fstab").id()
	assert.NoError(t, err)
	alt, err := mk("/tmp/fstab.alt").id()
	assert.NoError(t, err)

	assert.NotEqual(t, etc, alt,
		"two fstab files must not share an __id, or one silently serves the other")
	assert.Contains(t, etc, "/etc/fstab")

	again, err := mk("/etc/fstab").id()
	assert.NoError(t, err)
	assert.Equal(t, etc, again, "__id must be stable across calls")
}

// initFstab is what supplies the default path. os.linux.fstab has to go
// through it (NewResource), not around it (CreateResource).
func TestInitFstabDefaultsToEtcFstab(t *testing.T) {
	args, res, err := initFstab(nil, map[string]*llx.RawData{})
	assert.NoError(t, err)
	assert.Nil(t, res)
	assert.Equal(t, "/etc/fstab", args["path"].Value)
}
