// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tailscale/hujson"
)

func createTestZedConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeTestFile(t, dir, "settings.json", `{
		"ui_font_size": 16,
		"buffer_font_size": 16,
		"theme": {
			"mode": "system",
			"light": "One Light",
			"dark": "One Dark"
		},
		"telemetry": {
			"diagnostics": false,
			"metrics": false
		}
	}`)

	mkdirAllTest(t, dir, "extensions/html")
	mkdirAllTest(t, dir, "extensions/toml")

	return dir
}

func TestZedSettingsParsing(t *testing.T) {
	afs := testAfero()
	dir := createTestZedConfig(t)

	var settings map[string]interface{}
	err := readJSONFileAfero(afs, dir, "settings.json", &settings)
	require.NoError(t, err)
	assert.Equal(t, float64(16), settings["ui_font_size"])

	theme := settings["theme"].(map[string]interface{})
	assert.Equal(t, "system", theme["mode"])

	telemetry := settings["telemetry"].(map[string]interface{})
	assert.Equal(t, false, telemetry["diagnostics"])
}

func TestZedExtensionsFromDir(t *testing.T) {
	afs := testAfero()
	dir := createTestZedConfig(t)

	entries, err := afs.ReadDir(dir + "/extensions")
	require.NoError(t, err)

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	assert.Contains(t, names, "html")
	assert.Contains(t, names, "toml")
}

func TestZedSettingsJSONC(t *testing.T) {
	dir := t.TempDir()
	// Realistic Zed JSONC: line comments, inline comments, block comments, trailing commas
	writeTestFile(t, dir, "settings.json", `// Zed Settings
//
// For information on how to configure Zed, see the Zed
// documentation: https://zed.dev/docs/configuring-zed
{
  "ui_font_size": 14, // default
  /* Use system theme */
  "theme": {
    "mode": "system",
  }
}`)

	afs := testAfero()
	data, err := afs.ReadFile(dir + "/settings.json")
	require.NoError(t, err)

	clean, err := hujson.Standardize(data)
	require.NoError(t, err)

	var settings map[string]interface{}
	require.NoError(t, json.Unmarshal(clean, &settings))
	assert.Equal(t, float64(14), settings["ui_font_size"])

	theme := settings["theme"].(map[string]interface{})
	assert.Equal(t, "system", theme["mode"])
}

func TestZedConfigMissing(t *testing.T) {
	afs := testAfero()
	dir := t.TempDir()

	var settings map[string]interface{}
	err := readJSONFileAfero(afs, dir, "settings.json", &settings)
	assert.Error(t, err)
}
