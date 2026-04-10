// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testAfero returns an afero.Afero backed by the real OS filesystem,
// suitable for unit tests that create temp directories.
func testAfero() *afero.Afero {
	return &afero.Afero{Fs: afero.NewOsFs()}
}

// createTestClaudeConfig creates a temporary directory tree mimicking a Claude Code
// config directory and returns its path. The caller should defer os.RemoveAll(path).
func createTestClaudeConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// settings.json
	writeTestFile(t, dir, "settings.json", `{
		"enabledPlugins": {
			"gopls-lsp@official": true,
			"frontend-design@official": true
		}
	}`)

	// plugins/installed_plugins.json
	mkdirAllTest(t, dir, "plugins")
	writeTestFile(t, dir, "plugins/installed_plugins.json", `{
		"version": 2,
		"plugins": {
			"gopls-lsp@official": [{
				"scope": "user",
				"installPath": "/home/test/.claude/plugins/cache/gopls-lsp/1.0.0",
				"version": "1.0.0",
				"installedAt": "2026-01-01T00:00:00Z",
				"lastUpdated": "2026-01-01T00:00:00Z",
				"gitCommitSha": "abc123"
			}],
			"frontend-design@official": [{
				"scope": "user",
				"installPath": "/home/test/.claude/plugins/cache/frontend-design/2.0.0",
				"version": "2.0.0",
				"installedAt": "2026-02-01T00:00:00Z",
				"lastUpdated": "2026-02-01T00:00:00Z",
				"gitCommitSha": "def456"
			}]
		}
	}`)

	// backups with account info
	mkdirAllTest(t, dir, "backups")
	writeTestFile(t, dir, "backups/.claude.json.backup.1000000000000", `{
		"oauthAccount": {
			"accountUuid": "test-uuid-1234",
			"emailAddress": "test@example.com",
			"organizationUuid": "org-uuid-5678",
			"billingType": "team",
			"organizationRole": "owner",
			"organizationName": "TestOrg"
		},
		"projects": {
			"/home/test/project-one": {},
			"/home/test/go/src/github.com/user/repo": {}
		}
	}`)

	// skills
	mkdirAllTest(t, dir, "skills/review-pr")
	writeTestFile(t, dir, "skills/review-pr/SKILL.md", `---
name: review-pr
description: Review a pull request.
allowed-tools: Bash(gh *), Read, Grep
argument-hint: "<pr-number>"
---

# Review PR
`)

	// projects directories (matching the backup paths)
	mkdirAllTest(t, dir, "projects/-home-test-project-one")
	mkdirAllTest(t, dir, "projects/-home-test-go-src-github-com-user-repo/memory")
	writeTestFile(t, dir, "projects/-home-test-go-src-github-com-user-repo/memory/note.md", "test memory")

	// mcp-needs-auth-cache.json
	writeTestFile(t, dir, "mcp-needs-auth-cache.json", `{
		"claude.ai HubSpot": {"timestamp": 1700000000000},
		"claude.ai Gmail": {"timestamp": 1700000001000}
	}`)

	return dir
}

func mkdirAllTest(t *testing.T, base string, rel string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(base, rel), 0o755))
}

func writeTestFile(t *testing.T, base string, rel string, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(base, rel), []byte(content), 0o644))
}

func TestParseSkillMd(t *testing.T) {
	content := `---
name: review-pr
description: Review a pull request or branch using a structured checklist.
allowed-tools: Bash(gh *), Bash(git *), Read, Grep, Glob
argument-hint: "<pr-number-or-branch> [--focus area]"
---

# Pull Request Review

Review a PR or branch diff.
`

	skill := parseSkillMd("review-pr", "/home/user/.claude/skills/review-pr/SKILL.md", content)

	assert.Equal(t, "review-pr", skill.name)
	assert.Equal(t, "Review a pull request or branch using a structured checklist.", skill.description)
	require.Len(t, skill.allowedTools, 5)
	assert.Equal(t, "Bash(gh *)", skill.allowedTools[0])
	assert.Equal(t, "Bash(git *)", skill.allowedTools[1])
	assert.Equal(t, "Read", skill.allowedTools[2])
	assert.Equal(t, "Grep", skill.allowedTools[3])
	assert.Equal(t, "Glob", skill.allowedTools[4])
	assert.Equal(t, "<pr-number-or-branch> [--focus area]", skill.argumentHint)
	assert.Equal(t, "/home/user/.claude/skills/review-pr/SKILL.md", skill.source)
	assert.Equal(t, content, skill.content)
}

func TestParseSkillMdNoFrontmatter(t *testing.T) {
	content := "# Just a markdown file\n\nNo frontmatter here."
	skill := parseSkillMd("test", "/path/to/skill", content)

	assert.Equal(t, "test", skill.name)
	assert.Equal(t, "", skill.description)
	assert.Nil(t, skill.allowedTools)
}

func TestPathToProjectDir(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/pub/go/src/go.mondoo.com/mql", "-pub-go-src-go-mondoo-com-mql"},
		{"/home/user/projects", "-home-user-projects"},
		{"/pub/git/setup", "-pub-git-setup"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := pathToProjectDir(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDirHasFilesAfero(t *testing.T) {
	afs := testAfero()

	assert.False(t, dirHasFilesAfero(afs, "/nonexistent/path"))

	dir := t.TempDir()
	assert.False(t, dirHasFilesAfero(afs, dir))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hi"), 0o644))
	assert.True(t, dirHasFilesAfero(afs, dir))
}

func TestReadJSONFileAfero(t *testing.T) {
	afs := testAfero()
	dir := t.TempDir()
	writeTestFile(t, dir, "test.json", `{"key": "value"}`)

	var result map[string]string
	err := readJSONFileAfero(afs, dir, "test.json", &result)
	require.NoError(t, err)
	assert.Equal(t, "value", result["key"])

	err = readJSONFileAfero(afs, dir, "nonexistent.json", &result)
	assert.True(t, os.IsNotExist(err))
}

func TestFindLatestBackupAfero(t *testing.T) {
	afs := testAfero()
	dir := t.TempDir()
	mkdirAllTest(t, dir, "backups")

	writeTestFile(t, dir, "backups/.claude.json.backup.1000000000000", "{}")
	writeTestFile(t, dir, "backups/.claude.json.backup.2000000000000", "{}")
	writeTestFile(t, dir, "backups/.claude.json.backup.1500000000000", "{}")

	latest, err := findLatestBackupAfero(afs, dir)
	require.NoError(t, err)
	assert.Equal(t, ".claude.json.backup.2000000000000", latest)
}

func TestFindLatestBackupAferoEmpty(t *testing.T) {
	afs := testAfero()
	dir := t.TempDir()
	mkdirAllTest(t, dir, "backups")

	_, err := findLatestBackupAfero(afs, dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no backup files found")
}

func TestClaudeConfigIntegration(t *testing.T) {
	afs := testAfero()
	dir := createTestClaudeConfig(t)

	// Test settings reading
	var settings struct {
		EnabledPlugins map[string]bool `json:"enabledPlugins"`
	}
	err := readJSONFileAfero(afs, dir, "settings.json", &settings)
	require.NoError(t, err)
	assert.True(t, settings.EnabledPlugins["gopls-lsp@official"])
	assert.True(t, settings.EnabledPlugins["frontend-design@official"])

	// Test backup state loading
	latest, err := findLatestBackupAfero(afs, dir)
	require.NoError(t, err)

	var state claudeBackupState
	err = readJSONFileAfero(afs, dir, filepath.Join("backups", latest), &state)
	require.NoError(t, err)
	require.NotNil(t, state.OAuthAccount)
	assert.Equal(t, "test@example.com", state.OAuthAccount.EmailAddress)
	assert.Equal(t, "TestOrg", state.OAuthAccount.OrganizationName)
	assert.Equal(t, "owner", state.OAuthAccount.OrganizationRole)
	assert.Equal(t, "team", state.OAuthAccount.BillingType)
	assert.Equal(t, "test-uuid-1234", state.OAuthAccount.AccountUuid)
	assert.Equal(t, "org-uuid-5678", state.OAuthAccount.OrganizationUuid)

	// Test project dir mapping
	dirMap := state.projectDirMap()
	assert.Equal(t, "-home-test-project-one", dirMap["/home/test/project-one"])
	assert.Equal(t, "-home-test-go-src-github-com-user-repo", dirMap["/home/test/go/src/github.com/user/repo"])

	// Test memory detection
	assert.False(t, dirHasFilesAfero(afs, filepath.Join(dir, "projects", "-home-test-project-one", "memory")))
	assert.True(t, dirHasFilesAfero(afs, filepath.Join(dir, "projects", "-home-test-go-src-github-com-user-repo", "memory")))

	// Test skill parsing from fixture
	skillPath := filepath.Join(dir, "skills", "review-pr", "SKILL.md")
	data, err := afs.ReadFile(skillPath)
	require.NoError(t, err)
	skill := parseSkillMd("review-pr", skillPath, string(data))
	assert.Equal(t, "review-pr", skill.name)
	assert.Equal(t, "Review a pull request.", skill.description)
	require.Len(t, skill.allowedTools, 3)
	assert.Equal(t, "Bash(gh *)", skill.allowedTools[0])
	assert.Equal(t, "<pr-number>", skill.argumentHint)

	// Test MCP auth cache reading
	var cache map[string]struct {
		Timestamp int64 `json:"timestamp"`
	}
	err = readJSONFileAfero(afs, dir, "mcp-needs-auth-cache.json", &cache)
	require.NoError(t, err)
	assert.Len(t, cache, 2)
	assert.Equal(t, int64(1700000000000), cache["claude.ai HubSpot"].Timestamp)
}
