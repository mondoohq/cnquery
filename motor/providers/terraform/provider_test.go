package terraform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers"
)

func TestTerraform(t *testing.T) {
	p, err := New(&providers.Config{
		Options: map[string]string{
			"path": "./testdata/",
		},
	})
	require.NoError(t, err)

	files := p.Parser().Files()
	assert.Equal(t, len(files), 2)
}
