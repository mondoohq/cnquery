package fsutil_test

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
	"go.mondoo.io/mondoo/motor/transports/local"
)

func TestFileResource(t *testing.T) {
	path := "/tmp/test_hash"

	trans, err := local.New()
	assert.Nil(t, err)

	fs := trans.FS()
	afutil := afero.Afero{Fs: fs}

	// create the file and set the content
	err = afutil.WriteFile(path, []byte("hello world"), 0666)
	assert.Nil(t, err)

	f, err := fs.Open(path)
	assert.Nil(t, err)
	if assert.NotNil(t, f) {
		assert.Equal(t, path, f.Name(), "they should be equal")

		md5, err := fsutil.Md5(f)
		assert.Nil(t, err)
		assert.Equal(t, "5eb63bbbe01eeed093cb22bb8f5acdc3", md5)

		sha256, err := fsutil.Sha256(f)
		assert.Nil(t, err)
		assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", sha256)
	}
}
