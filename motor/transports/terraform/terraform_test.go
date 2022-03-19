package terraform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports"
)

func TestTerraform(t *testing.T) {
	trans, err := New(&transports.TransportConfig{
		Options: map[string]string{
			"path": "./testdata/",
		},
	})
	require.NoError(t, err)

	files := trans.Parser().Files()
	assert.Equal(t, len(files), 2)
}
