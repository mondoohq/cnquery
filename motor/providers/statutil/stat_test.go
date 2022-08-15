package statutil

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/mock"
)

func TestLinuxStatCmd(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/linux.toml")
	p, err := mock.NewFromTomlFile(filepath)
	require.NoError(t, err)

	statHelper := New(p)

	// get file stats
	fi, err := statHelper.Stat("/etc/ssh/sshd_config")
	require.NoError(t, err)

	assert.Equal(t, int64(4317), fi.Size())
	assert.Equal(t, false, fi.IsDir())
	assert.Equal(t, os.FileMode(0x180), fi.Mode())
	assert.Equal(t, time.Unix(1590420240, 0), fi.ModTime())
	require.NoError(t, err)
	mode := fi.Mode()
	assert.Zero(t, mode&fs.ModeSetuid)
	assert.Equal(t, "sshd_config", fi.Name())

	fi, err = statHelper.Stat("/usr/bin/su")
	require.NoError(t, err)
	mode = fi.Mode()
	assert.NotZero(t, mode&fs.ModeSetuid)
	assert.Zero(t, mode&fs.ModeSetgid)
}

func TestOpenbsdStatCmd(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/openbsd.toml")
	p, err := mock.NewFromTomlFile(filepath)
	require.NoError(t, err)

	statHelper := New(p)

	// get file stats
	fi, err := statHelper.Stat("/etc/ssh/sshd_config")
	require.NoError(t, err)

	assert.Equal(t, int64(2259), fi.Size())
	assert.Equal(t, false, fi.IsDir())
	assert.Equal(t, "-rw-r--r--", fi.Mode().String())
	assert.Equal(t, time.Unix(1592996018, 0), fi.ModTime())
	assert.Equal(t, int64(0), fi.Sys().(*providers.FileInfo).Uid)
	assert.Equal(t, int64(0), fi.Sys().(*providers.FileInfo).Gid)
	assert.Equal(t, "sshd_config", fi.Name())
}

func TestToFileMode(t *testing.T) {
	t.Run("directory and setgid", func(t *testing.T) {
		m := toFileMode(0o040000 | 0o002000 | 0o755)
		assert.True(t, m.IsDir())
		assert.True(t, (m&fs.ModeSetgid) > 0)
		assert.False(t, (m&fs.ModeSetuid) > 0)
		assert.False(t, (m&fs.ModeSticky) > 0)
		assert.Equal(t, fs.FileMode(0o755), (m & 0o777))
	})

	t.Run("directory and setuid", func(t *testing.T) {
		m := toFileMode(0o040000 | 0o004000 | 0o755)
		assert.True(t, m.IsDir())
		assert.False(t, (m&fs.ModeSetgid) > 0)
		assert.True(t, (m&fs.ModeSetuid) > 0)
		assert.False(t, (m&fs.ModeSticky) > 0)
		assert.Equal(t, fs.FileMode(0o755), (m & 0o777))
	})

	t.Run("directory and setuid and sticky", func(t *testing.T) {
		m := toFileMode(0o040000 | 0o004000 | 0o001000 | 0o755)
		assert.True(t, m.IsDir())
		assert.False(t, (m&fs.ModeSetgid) > 0)
		assert.True(t, (m&fs.ModeSetuid) > 0)
		assert.True(t, (m&fs.ModeSticky) > 0)
		assert.Equal(t, fs.FileMode(0o755), (m & 0o777))
	})

	t.Run("file and setuid", func(t *testing.T) {
		m := toFileMode(0o170000 | 0o100000 | 0o004000 | 0o755)
		assert.False(t, m.IsDir())
		assert.False(t, (m&fs.ModeSetgid) > 0)
		assert.True(t, (m&fs.ModeSetuid) > 0)
		assert.False(t, (m&fs.ModeSticky) > 0)
		assert.Equal(t, fs.FileMode(0o755), (m & 0o777))
	})

	t.Run("file and setuid and setgid", func(t *testing.T) {
		m := toFileMode(0o170000 | 0o100000 | 0o004000 | 0o002000 | 0o755)
		assert.False(t, m.IsDir())
		assert.True(t, (m&fs.ModeSetgid) > 0)
		assert.True(t, (m&fs.ModeSetuid) > 0)
		assert.False(t, (m&fs.ModeSticky) > 0)
		assert.Equal(t, fs.FileMode(0o755), (m & 0o777))
	})
}
