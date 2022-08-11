package tfstate

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTfstate(t *testing.T) {
	data, err := os.ReadFile("testdata/state_simple.json")
	require.NoError(t, err)

	var state State
	err = json.Unmarshal(data, &state)
	require.NoError(t, err)
	assert.NotNil(t, state)
}
