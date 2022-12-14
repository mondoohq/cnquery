package local_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers/local"
)

func TestFileResource(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "test")
	require.NoError(t, err)
	tmpfile.Close()

	path := tmpfile.Name()
	defer os.Remove(path)

	p, err := local.New()
	require.NoError(t, err)
	fs := p.FS()
	f, err := fs.Open(path)
	require.NoError(t, err)

	afutil := afero.Afero{Fs: fs}

	content := "hello world"

	// create the file and set the content
	err = ioutil.WriteFile(path, []byte(content), 0o666)
	require.NoError(t, err)

	if assert.NotNil(t, f) {
		assert.Equal(t, path, f.Name(), "they should be equal")
		c, err := afutil.ReadFile(f.Name())
		assert.Nil(t, err)
		assert.Equal(t, content, string(c), "content should be equal")
	}
}

func TestFilePermissions666(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "test")
	require.NoError(t, err)
	tmpfile.Close()

	path := tmpfile.Name()
	defer os.Remove(path)

	err = ioutil.WriteFile(path, []byte("hello"), 0o666)
	require.NoError(t, err)

	// ensure permissions
	err = os.Chmod(path, 0o666)
	require.NoError(t, err)

	p, err := local.New()
	require.NoError(t, err)

	details, err := p.FileInfo(path)
	require.NoError(t, err)
	assert.Equal(t, int64(os.Getuid()), details.Uid)
	assert.Equal(t, int64(os.Getgid()), details.Gid)
	assert.True(t, details.Size >= 0)
	assert.Equal(t, false, details.Mode.IsDir())
	assert.Equal(t, true, details.Mode.IsRegular())
	assert.Equal(t, "-rw-rw-rw-", details.Mode.String())
	assert.True(t, details.Mode.UserReadable())
	assert.True(t, details.Mode.UserWriteable())
	assert.False(t, details.Mode.UserExecutable())
	assert.True(t, details.Mode.GroupReadable())
	assert.True(t, details.Mode.GroupWriteable())
	assert.False(t, details.Mode.GroupExecutable())
	assert.True(t, details.Mode.OtherReadable())
	assert.True(t, details.Mode.OtherWriteable())
	assert.False(t, details.Mode.OtherExecutable())
	assert.False(t, details.Mode.Suid())
	assert.False(t, details.Mode.Sgid())
	assert.False(t, details.Mode.Sticky())
}

func TestFilePermissions755(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "test")
	require.NoError(t, err)
	tmpfile.Close()

	path := tmpfile.Name()
	defer os.Remove(path)

	err = os.WriteFile(path, []byte("hello"), 0o755)
	require.NoError(t, err)

	// ensure permissions
	err = os.Chmod(path, 0o755)
	require.NoError(t, err)

	p, err := local.New()
	require.NoError(t, err)

	details, err := p.FileInfo(path)
	require.NoError(t, err)
	assert.Equal(t, int64(os.Getuid()), details.Uid)
	assert.Equal(t, int64(os.Getgid()), details.Gid)
	assert.True(t, details.Size >= 0)
	assert.Equal(t, false, details.Mode.IsDir())
	assert.Equal(t, true, details.Mode.IsRegular())
	assert.Equal(t, "-rwxr-xr-x", details.Mode.String())
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
