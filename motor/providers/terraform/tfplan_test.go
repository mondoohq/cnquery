package terraform

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTfplan(t *testing.T) {
	data, err := os.ReadFile("testdata/gcp/plan_simple.json")
	require.NoError(t, err)

	var plan Plan
	err = json.Unmarshal(data, &plan)
	require.NoError(t, err)
	assert.NotNil(t, plan)
}
