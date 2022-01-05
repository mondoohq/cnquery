package cat_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports/mock"
	"go.mondoo.io/mondoo/motor/transports/winrm/cat"
)

func TestCatFs(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/winrm.toml")
	trans, err := mock.NewFromTomlFile(filepath)
	require.NoError(t, err)

	catfs := cat.New(trans)

	// fetch file content
	f, err := catfs.Open("C:\\test.txt")
	require.NoError(t, err)

	data, err := ioutil.ReadAll(f)
	require.NoError(t, err)

	expected := "hi\n"
	assert.Equal(t, expected, string(data))

	// get file stats
	fi, err := catfs.Stat("C:\\test.txt")
	require.NoError(t, err)

	assert.Equal(t, int64(2), fi.Size())
	assert.Equal(t, false, fi.IsDir())
	assert.Equal(t, int64(1603529613), fi.ModTime().Unix())
}
