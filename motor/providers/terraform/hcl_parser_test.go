package terraform

import (
	"os"
	"testing"

	"github.com/hashicorp/hcl/v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadHclBlocks(t *testing.T) {
	path := "./testdata/hcl"
	fileList, err := os.ReadDir(path)
	require.NoError(t, err)

	loader := NewHCLFileLoader()
	err = loader.ParseHclDirectory(path, fileList)
	require.NoError(t, err)
	assert.Equal(t, 2, len(loader.GetParser().Files()))
}

func TestLoadTfvars(t *testing.T) {
	path := "./testdata/hcl"
	fileList, err := os.ReadDir(path)
	require.NoError(t, err)

	variables := make(map[string]*hcl.Attribute)
	err = ReadTfVarsFromDir(path, fileList, variables)
	require.NoError(t, err)
	assert.Equal(t, 2, len(variables))
}
