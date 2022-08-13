package terraform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/providers"
)

func TestTerraform(t *testing.T) {
	trans, err := New(&providers.TransportConfig{
		Options: map[string]string{
			"path": "./testdata/",
		},
	})
	require.NoError(t, err)

	files := trans.Parser().Files()
	assert.Equal(t, len(files), 2)
}
