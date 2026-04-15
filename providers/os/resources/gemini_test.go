// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestGeminiConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeTestFile(t, dir, "settings.json", `{
		"theme": "GitHub",
		"selectedAuthType": "oauth-personal"
	}`)

	mkdirAllTest(t, dir, "antigravity")
	writeTestFile(t, dir, "antigravity/mcp_config.json", `{
		"mcpServers": {
			"filesystem": {
				"command": "npx",
				"args": ["-y", "@modelcontextprotocol/server-filesystem"],
				"env": {"HOME": "/tmp"}
			}
		}
	}`)

	return dir
}

func TestGeminiSettingsParsing(t *testing.T) {
	afs := testAfero()
	dir := createTestGeminiConfig(t)

	var settings geminiSettings
	err := readJSONFileAfero(afs, dir, "settings.json", &settings)
	require.NoError(t, err)
	assert.Equal(t, "oauth-personal", settings.SelectedAuthType)
	assert.Equal(t, "GitHub", settings.Theme)
}

func TestGeminiMCPParsing(t *testing.T) {
	afs := testAfero()
	dir := createTestGeminiConfig(t)

	data, err := afs.ReadFile(filepath.Join(dir, "antigravity", "mcp_config.json"))
	require.NoError(t, err)

	var config geminiMCPConfig
	err = json.Unmarshal(data, &config)
	require.NoError(t, err)
	assert.Len(t, config.McpServers, 1)

	fs := config.McpServers["filesystem"]
	assert.Equal(t, "npx", fs.Command)
	assert.Len(t, fs.Args, 2)
	assert.Len(t, fs.Env, 1)
}

func TestGeminiConfigMissing(t *testing.T) {
	afs := testAfero()
	dir := t.TempDir()

	var settings geminiSettings
	err := readJSONFileAfero(afs, dir, "settings.json", &settings)
	assert.Error(t, err)
}
