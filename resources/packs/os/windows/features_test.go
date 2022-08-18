package windows

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowsFeatures(t *testing.T) {
	r, err := os.Open("./testdata/features.json")
	require.NoError(t, err)

	items, err := ParseWindowsFeatures(r)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(items))
}
