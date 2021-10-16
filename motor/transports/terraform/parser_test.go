package terraform

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadHclBlocks(t *testing.T) {
	path := "./testdata"
	fileList, err := ioutil.ReadDir(path)

	parsed, err := ParseHclDirectory(path, fileList)
	require.NoError(t, err)
	assert.Equal(t, 1, len(parsed.Files()))
}
