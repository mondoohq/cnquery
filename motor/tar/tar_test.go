package tar_test

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1/mutate"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/tar"
	"go.mondoo.io/mondoo/motor/types"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func TestTarCommand(t *testing.T) {
	err := cacheImageToTar()
	assert.Equal(t, nil, err, "should create tar without error")
	if err != nil {
		return
	}

	filepath, _ := filepath.Abs("./alpine-container.tar")
	tarTransport, err := tar.New(&types.Endpoint{Backend: "tar", Path: filepath})
	assert.Equal(t, nil, err, "should create tar without error")

	cmd, err := tarTransport.RunCommand("ls /")
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

	filepath, _ := filepath.Abs("./alpine-container.tar")
	tarTransport, err := tar.New(&types.Endpoint{Backend: "tar", Path: filepath})
	assert.Equal(t, nil, err, "should create tar without error")

	f, err := tarTransport.File("/bin/cat")

	if assert.NotNil(t, f) {
		assert.Equal(t, nil, err, "should execute without error")

		p := f.Name()
		assert.Equal(t, "/bin/cat", p, "path should be correct")

		stat, err := f.Stat()
		assert.Equal(t, nil, err, "should stat without error")
		assert.Equal(t, int64(796240), stat.Size(), "should read file size")

		reader, err := f.Open()
		assert.Equal(t, nil, err, "should open without error")
		content, err := ioutil.ReadAll(reader)
		assert.Equal(t, nil, err, "should execute without error")
		assert.Equal(t, 796240, len(content), "should read the full content")

		// ensure the same works with tar()
		content, err = ioutil.ReadAll(f)
		assert.Equal(t, nil, err, "should read without error")
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
// 	tarTransport, err := tar.New(&types.Endpoint{Backend: "tar", Path: filepath})
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
	assert.Equal(t, nil, err, "should create tar without error")
	if err != nil {
		return
	}

	filepath, _ := filepath.Abs("./alpine-container.tar")
	tarTransport, err := tar.New(&types.Endpoint{Backend: "tar", Path: filepath})
	assert.Equal(t, nil, err, "should create tar without error")

	f, err := tarTransport.File("/etc/alpine-release")

	if assert.NotNil(t, f) {
		assert.Equal(t, nil, err, "should execute without error")

		p := f.Name()
		assert.Equal(t, "/etc/alpine-release", p, "path should be correct")

		stat, err := f.Stat()
		assert.Equal(t, int64(6), stat.Size(), "should read file size")
		assert.Equal(t, nil, err, "should execute without error")

		reader, err := f.Open()
		content, err := ioutil.ReadAll(reader)
		assert.Equal(t, nil, err, "should execute without error")
		assert.Equal(t, 6, len(content), "should read the full content")
	}
}

func cacheImageToTar() error {

	source := "alpine:3.9"
	filename := "./alpine-container.tar"

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
