package windows

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowsComputerInfo(t *testing.T) {
	r, err := os.Open("./testdata/computer-info.json")
	require.NoError(t, err)

	items, err := ParseComputerInfo(r)
	assert.Nil(t, err)
	assert.Equal(t, 43, len(items))
}
