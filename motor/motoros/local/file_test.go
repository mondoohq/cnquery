package local_test

import (
	"io/ioutil"
	"log"
	"testing"

	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/motoros/local"

	"github.com/stretchr/testify/assert"
)

func TestFileResource(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "test")
	if err != nil {
		log.Fatal(err)
	}
	tmpfile.Close()

	path := tmpfile.Name()

	trans, err := local.New()
	assert.Nil(t, err)

	fs := trans.FS()
	f, err := fs.Open(path)
	assert.Nil(t, err)

	afutil := afero.Afero{Fs: fs}

	content := "hello world"

	// create the file and set the content
	err = ioutil.WriteFile(path, []byte(content), 0666)
	assert.Nil(t, err)

	if assert.NotNil(t, f) {
		assert.Equal(t, path, f.Name(), "they should be equal")
		c, err := afutil.ReadFile(f.Name())
		assert.Nil(t, err)
		assert.Equal(t, content, string(c), "content should be equal")
	}
}
