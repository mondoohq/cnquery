package local

import (
	"io/ioutil"
	"testing"

	"go.mondoo.io/mondoo/motor/motorutil"

	"github.com/stretchr/testify/assert"
)

func TestFileResource(t *testing.T) {

	// create the file and set the content
	f := &File{filePath: "/tmp/test"}

	err := ioutil.WriteFile(f.filePath, []byte("hello world"), 0666)
	assert.Nil(t, err)

	if assert.NotNil(t, f) {
		assert.Equal(t, "/tmp/test", f.Name(), "they should be equal")

		c, err := motorutil.ReadFile(f)
		assert.Nil(t, err)
		assert.Equal(t, "hello world", string(c), "content should be equal")
	}
}
