// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
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

func TestReadZedWorkspacePaths(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "db.sqlite")

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE workspaces (workspace_id INTEGER PRIMARY KEY, local_paths BLOB, window_state TEXT)`)
	require.NoError(t, err)

	// bincode-ish blob: 8-byte length prefix (mostly non-printable) then the path.
	blob := []byte{0x1d, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	blob = append(blob, []byte("/pub/go/src/go.mondoo.com/mql")...)
	_, err = db.Exec(`INSERT INTO workspaces (local_paths, window_state) VALUES (?, ?)`, blob, `{"x":1}`)
	require.NoError(t, err)
	// a second workspace whose path is not a printable-leading absolute path
	_, err = db.Exec(`INSERT INTO workspaces (local_paths, window_state) VALUES (?, ?)`, []byte("no-paths-here"), "")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	paths := readZedWorkspacePaths(testAfero(), dbPath)
	assert.Contains(t, paths, "/pub/go/src/go.mondoo.com/mql")

	// missing db yields nil, not a panic
	assert.Nil(t, readZedWorkspacePaths(testAfero(), filepath.Join(dir, "missing.sqlite")))
}
