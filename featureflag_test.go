package cnquery

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFeatureFlags(t *testing.T) {
	f := Features{byte(MassQueries)}
	assert.True(t, f.IsActive(MassQueries))

	parsed, err := DecodeFeatures(f.Encode())
	require.NoError(t, err)
	assert.Equal(t, f, parsed)

	f = Features{}
	assert.False(t, f.IsActive(MassQueries))

	parsed, err = DecodeFeatures(f.Encode())
	require.NoError(t, err)
	assert.Equal(t, f, parsed)
}
