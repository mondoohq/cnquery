// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTfplan(t *testing.T) {
	data, err := os.ReadFile("./testdata/gcp/plan_simple.json")
	require.NoError(t, err)

	var plan Plan
	err = json.Unmarshal(data, &plan)
	require.NoError(t, err)
	assert.NotNil(t, plan)
}

func TestTfWithDynamicBlocksAndVariables(t *testing.T) {
	data, err := os.ReadFile("./testdata/dynamic_block/tfplan.json")
	require.NoError(t, err)

	var plan Plan
	err = json.Unmarshal(data, &plan)
	require.NoError(t, err)
	assert.NotNil(t, plan)
	_, ok := plan.Variables["environment"] // this exist in the testdata
	assert.True(t, ok)
	assert.True(t, plan.Applyable)
	assert.False(t, plan.Errored)
}
