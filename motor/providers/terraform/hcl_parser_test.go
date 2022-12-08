package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"go.mondoo.com/cnquery/motor/providers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadHclBlocks(t *testing.T) {
	path := "./testdata/"
	tc := &providers.Config{
		Options: map[string]string{
			"path": path,
		},
	}
	tf, err := New(tc)
	require.NoError(t, err)
	require.NotNil(t, tf.parsed)
}

func TestLoadTfvars(t *testing.T) {
	path := "./testdata/hcl/sample.tfvars"
	variables := make(map[string]*hcl.Attribute)
	err := ReadTfVarsFromFile(path, variables)
	require.NoError(t, err)
	assert.Equal(t, 2, len(variables))
}
