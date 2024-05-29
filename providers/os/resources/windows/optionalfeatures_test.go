// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowsOptionalFeatures(t *testing.T) {
	r, err := os.Open("./testdata/optionalfeatures.json")
	require.NoError(t, err)

	items, err := ParseWindowsOptionalFeatures(r)
	assert.Nil(t, err)
	assert.Equal(t, 134, len(items))
	assert.Equal(t, "MicrosoftWindowsPowerShellV2", items[9].Name)
	assert.Equal(t, "Windows PowerShell 2.0 Engine", items[9].DisplayName)
	assert.True(t, items[9].Enabled)
	assert.Equal(t, int64(2), items[9].State)
	assert.Equal(t, "Adds or Removes Windows PowerShell 2.0 Engine", items[9].Description)
}
