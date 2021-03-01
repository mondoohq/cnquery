package fs

import (
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
	"testing"
)

func TestOsDetection(t *testing.T) {
	trans := &FsTransport{
		mountedDir: "./testdata/centos8",
	}

	m, err := motor.New(trans)
	require.NoError(t, err)

	pf, err := m.Platform()
	require.NoError(t, err)

	assert.Equal(t, "centos", pf.Name)
	assert.Equal(t, "8.2.2004", pf.Release)
}

func TestMountedDirectoryFile(t *testing.T) {
	trans := &FsTransport{
		mountedDir: "./testdata/centos8",
	}

	f, err := trans.FS().Open("/etc/os-release")
	assert.Nil(t, err, "should open without error")
	assert.NotNil(t, f)
	defer f.Close()

	afutil := afero.Afero{Fs: trans.FS()}
	afutil.Exists(f.Name())

	p := f.Name()
	assert.Equal(t, "/etc/os-release", p, "path should be correct")

	stat, err := f.Stat()
	assert.Equal(t, int64(417), stat.Size(), "should read file size")
	assert.Nil(t, err, "should execute without error")

	content, err := afutil.ReadFile(f.Name())
	assert.Equal(t, nil, err, "should execute without error")
	assert.Equal(t, 417, len(content), "should read the full content")

	// reset reader
	f.Seek(0, 0)
	sha, err := fsutil.Sha256(f)
	assert.Equal(t, "1d272eeae89e45470abf750cdc037eb72b216686cf8c105e5b9925df21ec1043", sha, "sha256 output should be correct")
	assert.Nil(t, err, "should execute without error")

	// reset reader
	f.Seek(0, 0)
	md5, err := fsutil.Md5(f)
	assert.Equal(t, "f5a898d54907811ccc54cd35dcb991d1", md5, "md5 output should be correct")
	assert.Nil(t, err, "should execute without error")
}
