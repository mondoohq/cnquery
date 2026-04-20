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

func createTestCopilotConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeTestFile(t, dir, "apps.json", `{
		"github.com:Iv23ctfURkiMfJ4xr5mv": {
			"oauth_token": "ghu_test123",
			"user": "testuser",
			"githubAppId": "Iv23ctfURkiMfJ4xr5mv"
		},
		"github.com:Ov23liV9UpD7Rnfnskm3": {
			"user": "testuser",
			"oauth_token": "gho_test456",
			"githubAppId": "Ov23liV9UpD7Rnfnskm3"
		}
	}`)

	mkdirAllTest(t, dir, "intellij")
	writeTestFile(t, dir, "intellij/mcp.json", `{
		"servers": {
			"my-server": {
				"type": "stdio",
				"command": "my-command",
				"args": ["--flag"]
			}
		}
	}`)

	return dir
}

func TestCopilotAppsParsing(t *testing.T) {
	afs := testAfero()
	dir := createTestCopilotConfig(t)

	data, err := afs.ReadFile(filepath.Join(dir, "apps.json"))
	require.NoError(t, err)

	var apps map[string]copilotApp
	err = json.Unmarshal(data, &apps)
	require.NoError(t, err)
	assert.Len(t, apps, 2)

	app := apps["github.com:Iv23ctfURkiMfJ4xr5mv"]
	assert.Equal(t, "testuser", app.User)
	assert.Equal(t, "Iv23ctfURkiMfJ4xr5mv", app.GitHubAppID)
	assert.Equal(t, "ghu_test123", app.OAuthToken)
}

func TestCopilotMCPParsing(t *testing.T) {
	afs := testAfero()
	dir := createTestCopilotConfig(t)

	data, err := afs.ReadFile(filepath.Join(dir, "intellij", "mcp.json"))
	require.NoError(t, err)

	var config copilotMCPConfig
	err = json.Unmarshal(data, &config)
	require.NoError(t, err)
	assert.Len(t, config.Servers, 1)

	server := config.Servers["my-server"]
	assert.Equal(t, "stdio", server.Type)
	assert.Equal(t, "my-command", server.Command)
	assert.Equal(t, []string{"--flag"}, server.Args)
}

func TestCopilotConfigMissing(t *testing.T) {
	afs := testAfero()
	dir := t.TempDir()

	_, err := afs.ReadFile(filepath.Join(dir, "apps.json"))
	assert.Error(t, err)
}
