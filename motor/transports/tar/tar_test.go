package tar_test

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1/mutate"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/motorapi"
	"go.mondoo.io/mondoo/motor/transports/tar"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

const alpineContainerPath = "./alpine-container.tar"

func TestTarCommand(t *testing.T) {
	err := cacheImageToTar()
	assert.Equal(t, nil, err, "should create tar without error")
	if err != nil {
		return
	}

	tarTransport, err := tar.New(&motorapi.Endpoint{Backend: "tar", Path: alpineContainerPath})
	assert.Equal(t, nil, err, "should create tar without error")

	cmd, err := tarTransport.RunCommand("ls /")
	assert.Nil(t, err)
	if assert.NotNil(t, cmd) {
		assert.Equal(t, nil, err, "should execute without error")
		assert.Equal(t, -1, cmd.ExitStatus, "command should not be executed")
		stdoutContent, _ := ioutil.ReadAll(cmd.Stdout)
		assert.Equal(t, "", string(stdoutContent), "output should be correct")
		stderrContent, _ := ioutil.ReadAll(cmd.Stdout)
		assert.Equal(t, "", string(stderrContent), "output should be correct")
	}
}
func TestTarSymlinkFile(t *testing.T) {
	err := cacheImageToTar()
	assert.Equal(t, nil, err, "should create tar without error")
	if err != nil {
		return
	}

	tarTransport, err := tar.New(&motorapi.Endpoint{Backend: "tar", Path: alpineContainerPath})
	assert.Equal(t, nil, err, "should create tar without error")

	f, err := tarTransport.FS().Open("/bin/cat")
	assert.Nil(t, err)
	if assert.NotNil(t, f) {
		assert.Equal(t, nil, err, "should execute without error")

		p := f.Name()
		assert.Equal(t, "/bin/cat", p, "path should be correct")

		stat, err := f.Stat()
		assert.Equal(t, nil, err, "should stat without error")
		assert.Equal(t, int64(796240), stat.Size(), "should read file size")

		content, err := ioutil.ReadAll(f)
		assert.Equal(t, nil, err, "should execute without error")
		assert.Equal(t, 796240, len(content), "should read the full content")
	}
}

// deactivate test for now for speedier testing
// in contrast to alpine, the symlink on centos is pointinng to a relative target
// and not an absolute one
//
// func TestTarSymlinkFileCentos(t *testing.T) {
// 	err := cacheImageToTar()
// 	assert.Equal(t, nil, err, "should create tar without error")
// 	if err != nil {
// 		return
// 	}

// 	filepath, _ := filepath.Abs("./centos-container.tar")
// 	tarTransport, err := tar.New(&motorapi.Endpoint{Backend: "tar", Path: filepath})
// 	assert.Equal(t, nil, err, "should create tar without error")

// 	f, err := tarTransport.File("/etc/redhat-release")

// 	if assert.NotNil(t, f) {
// 		assert.Equal(t, nil, err, "should execute without error")

// 		p := f.Name()
// 		assert.Equal(t, "/etc/redhat-release", p, "path should be correct")

// 		stat, err := f.Stat()
// 		assert.Equal(t, nil, err, "should stat without error")
// 		assert.Equal(t, int64(38), stat.Size(), "should read file size")

// 		reader, err := f.Open()
// 		assert.Equal(t, nil, err, "should open without error")
// 		content, err := ioutil.ReadAll(reader)
// 		assert.Equal(t, nil, err, "should execute without error")
// 		assert.Equal(t, 38, len(content), "should read the full content")

// 		// ensure the same works with tar()
// 		content, err = motorutil.ReadFile(f)
// 		assert.Equal(t, nil, err, "should read without error")
// 		assert.Equal(t, 38, len(content), "should read the full content")
// 	}
// }
func TestTarFile(t *testing.T) {
	err := cacheImageToTar()
	require.NoError(t, err)

	tarTransport, err := tar.New(&motorapi.Endpoint{Backend: "tar", Path: alpineContainerPath})
	assert.Equal(t, nil, err, "should create tar without error")

	f, err := tarTransport.FS().Open("/etc/alpine-release")
	assert.Nil(t, err)
	if assert.NotNil(t, f) {
		assert.Equal(t, nil, err, "should execute without error")

		p := f.Name()
		assert.Equal(t, "/etc/alpine-release", p, "path should be correct")

		stat, err := f.Stat()
		assert.Equal(t, int64(6), stat.Size(), "should read file size")
		assert.Equal(t, nil, err, "should execute without error")

		content, err := ioutil.ReadAll(f)
		assert.Equal(t, nil, err, "should execute without error")
		assert.Equal(t, 6, len(content), "should read the full content")
	}
}

func TestFilePermissions(t *testing.T) {
	err := cacheImageToTar()
	require.NoError(t, err)

	trans, err := tar.New(&motorapi.Endpoint{Backend: "tar", Path: alpineContainerPath})
	require.NoError(t, err)

	path := "/etc/alpine-release"
	details, err := trans.FileInfo(path)
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
	details, err = trans.FileInfo(path)
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

func cacheImageToTar() error {

	source := "alpine:3.9"
	filename := alpineContainerPath

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

	// convert multi-layer image into a flatten container tar
	rc := mutate.Extract(img)
	defer rc.Close()

	// write content to file
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc)

	return err
}

func TestTarFileFind(t *testing.T) {
	err := cacheImageToTar()
	require.NoError(t, err)

	trans, err := tar.New(&motorapi.Endpoint{Backend: "tar", Path: alpineContainerPath})
	assert.Equal(t, nil, err, "should create tar without error")

	fs := trans.FS()

	fSearch := fs.(*tar.FS)

	infos, err := fSearch.Find("/", regexp.MustCompile(`alpine-release`), "file")
	require.NoError(t, err)

	assert.Equal(t, 1, len(infos))
}
