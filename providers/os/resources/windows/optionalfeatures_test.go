// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"os"
	"strings"
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

// a single-feature lookup makes ConvertTo-Json emit a bare object, not an array
func TestWindowsOptionalFeatures_SingleObject(t *testing.T) {
	input := `{
    "FeatureName": "TelnetClient",
    "DisplayName": "Telnet Client",
    "Description": "Telnet Client uses the Telnet protocol",
    "State": 2
}`

	items, err := ParseWindowsOptionalFeatures(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "TelnetClient", items[0].Name)
	assert.Equal(t, "Telnet Client", items[0].DisplayName)
	assert.Equal(t, "Telnet Client uses the Telnet protocol", items[0].Description)
	assert.Equal(t, int64(2), items[0].State)
	assert.True(t, items[0].Enabled)
}

func TestWindowsOptionalFeatures_Empty(t *testing.T) {
	items, err := ParseWindowsOptionalFeatures(strings.NewReader("   \n  "))
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestOptionalFeatureQuery(t *testing.T) {
	assert.Equal(t,
		"Get-WindowsOptionalFeature -Online -FeatureName 'TelnetClient' | Select-Object -Property FeatureName,DisplayName,Description,State | ConvertTo-Json",
		OptionalFeatureQuery("TelnetClient"))

	// single quotes must be doubled so the name stays a literal string
	assert.Equal(t,
		"Get-WindowsOptionalFeature -Online -FeatureName 'O''Brien' | Select-Object -Property FeatureName,DisplayName,Description,State | ConvertTo-Json",
		OptionalFeatureQuery("O'Brien"))
}
