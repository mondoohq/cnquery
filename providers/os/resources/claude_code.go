// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

const defaultClaudeCodeConfigDir = ".claude"

func initClaudeCode(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "claude.code", defaultClaudeCodeConfigDir)
}

// mqlClaudeCodeInternal caches the backup state for the lifetime of
// this resource instance, avoiding the global map that leaked across assets.
type mqlClaudeCodeInternal struct {
	backupOnce  sync.Once
	backupState *claudeBackupState
	backupErr   error
}

func (r *mqlClaudeCode) id() (string, error) {
	return "claude.code/" + r.ConfigPath.Data, nil
}

// configDir returns the configPath for this resource instance.
func (r *mqlClaudeCode) configDir() string {
	return r.ConfigPath.Data
}

// afs returns an afero.Afero wrapping the connection's filesystem.
func (r *mqlClaudeCode) afs() *afero.Afero {
	return connectionAfs(r.MqlRuntime)
}

func (r *mqlClaudeCode) email() (string, error) {
	acct, err := r.loadOAuthAccount()
	if err != nil {
		return "", err
	}
	return acct.EmailAddress, nil
}

func (r *mqlClaudeCode) organization() (string, error) {
	acct, err := r.loadOAuthAccount()
	if err != nil {
		return "", err
	}
	return acct.OrganizationName, nil
}

func (r *mqlClaudeCode) role() (string, error) {
	acct, err := r.loadOAuthAccount()
	if err != nil {
		return "", err
	}
	return acct.OrganizationRole, nil
}

func (r *mqlClaudeCode) subscription() (string, error) {
	acct, err := r.loadOAuthAccount()
	if err != nil {
		return "", err
	}
	return acct.BillingType, nil
}

func (r *mqlClaudeCode) userId() (string, error) {
	acct, err := r.loadOAuthAccount()
	if err != nil {
		return "", err
	}
	return acct.AccountUuid, nil
}

func (r *mqlClaudeCode) organizationId() (string, error) {
	acct, err := r.loadOAuthAccount()
	if err != nil {
		return "", err
	}
	return acct.OrganizationUuid, nil
}

func (r *mqlClaudeCode) settings() (interface{}, error) {
	var settings map[string]interface{}
	err := readJSONFileAfero(r.afs(), r.configDir(), "settings.json", &settings)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}
	return settings, nil
}

func (r *mqlClaudeCode) enabledPlugins() ([]interface{}, error) {
	var settings struct {
		EnabledPlugins map[string]bool `json:"enabledPlugins"`
	}
	err := readJSONFileAfero(r.afs(), r.configDir(), "settings.json", &settings)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var result []interface{}
	for name, enabled := range settings.EnabledPlugins {
		if enabled {
			result = append(result, name)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].(string) < result[j].(string)
	})
	return result, nil
}

func (r *mqlClaudeCode) plugins() ([]interface{}, error) {
	afs := r.afs()

	var installedPlugins struct {
		Version int                               `json:"version"`
		Plugins map[string][]installedPluginEntry `json:"plugins"`
	}
	err := readJSONFileAfero(afs, r.configDir(), "plugins/installed_plugins.json", &installedPlugins)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var settings struct {
		EnabledPlugins map[string]bool `json:"enabledPlugins"`
	}
	_ = readJSONFileAfero(afs, r.configDir(), "settings.json", &settings)

	var result []interface{}
	for name, entries := range installedPlugins.Plugins {
		for _, entry := range entries {
			enabled := false
			if settings.EnabledPlugins != nil {
				enabled = settings.EnabledPlugins[name]
			}

			pluginID := "claude.code.plugin/" + name + "/" + entry.Scope
			res, err := NewResource(r.MqlRuntime, "claude.code.plugin", map[string]*llx.RawData{
				"__id":         llx.StringData(pluginID),
				"name":         llx.StringData(name),
				"version":      llx.StringData(entry.Version),
				"scope":        llx.StringData(entry.Scope),
				"installPath":  llx.StringData(entry.InstallPath),
				"installedAt":  llx.StringData(entry.InstalledAt),
				"lastUpdated":  llx.StringData(entry.LastUpdated),
				"gitCommitSha": llx.StringData(entry.GitCommitSha),
				"enabled":      llx.BoolData(enabled),
			})
			if err != nil {
				return nil, err
			}
			result = append(result, res)
		}
	}
	return result, nil
}

func (r *mqlClaudeCode) skills() ([]interface{}, error) {
	afs := r.afs()
	skillsDir := filepath.Join(r.configDir(), "skills")

	subdirs, err := listSubdirsAfero(afs, skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []interface{}
	for _, dir := range subdirs {
		skillPath := filepath.Join(dir.path, "SKILL.md")
		data, err := afs.ReadFile(skillPath)
		if err != nil {
			continue
		}

		skill := parseSkillMd(dir.name, skillPath, string(data))

		allowedToolsAny := make([]interface{}, len(skill.allowedTools))
		for i, t := range skill.allowedTools {
			allowedToolsAny[i] = t
		}

		res, err := NewResource(r.MqlRuntime, "claude.code.skill", map[string]*llx.RawData{
			"__id":         llx.StringData("claude.code.skill/" + dir.name),
			"name":         llx.StringData(skill.name),
			"description":  llx.StringData(skill.description),
			"allowedTools": llx.ArrayData(allowedToolsAny, types.String),
			"argumentHint": llx.StringData(skill.argumentHint),
			"source":       llx.StringData(skill.source),
			"content":      llx.StringData(skill.content),
		})
		if err != nil {
			return nil, err
		}
		result = append(result, res)
	}
	return result, nil
}

func (r *mqlClaudeCode) projects() ([]interface{}, error) {
	afs := r.afs()
	state, err := r.loadBackupState()
	if err != nil {
		return nil, err
	}

	projectsDir := filepath.Join(r.configDir(), "projects")
	var result []interface{}
	for projectPath, dirName := range state.projectDirMap() {
		memoryDir := filepath.Join(projectsDir, dirName, "memory")
		hasMemory := dirHasFilesAfero(afs, memoryDir)

		res, err := NewResource(r.MqlRuntime, "claude.code.project", map[string]*llx.RawData{
			"__id":      llx.StringData("claude.code.project/" + projectPath),
			"path":      llx.StringData(projectPath),
			"hasMemory": llx.BoolData(hasMemory),
		})
		if err != nil {
			return nil, err
		}
		result = append(result, res)
	}
	return result, nil
}

func (r *mqlClaudeCode) mcpServers() ([]interface{}, error) {
	var cache map[string]struct {
		Timestamp int64 `json:"timestamp"`
	}
	err := readJSONFileAfero(r.afs(), r.configDir(), "mcp-needs-auth-cache.json", &cache)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []interface{}
	for name, entry := range cache {
		lastChecked := ""
		if entry.Timestamp > 0 {
			lastChecked = time.UnixMilli(entry.Timestamp).UTC().Format(time.RFC3339)
		}

		// Presence in mcp-needs-auth-cache.json means the server requires
		// authentication; servers that don't need auth are not listed.
		res, err := NewResource(r.MqlRuntime, "claude.code.mcpServer", map[string]*llx.RawData{
			"__id":        llx.StringData("claude.code.mcpServer/" + name),
			"name":        llx.StringData(name),
			"needsAuth":   llx.BoolData(true),
			"lastChecked": llx.StringData(lastChecked),
		})
		if err != nil {
			return nil, err
		}
		result = append(result, res)
	}
	return result, nil
}

// Helper types and functions

type oauthAccount struct {
	AccountUuid      string `json:"accountUuid"`
	EmailAddress     string `json:"emailAddress"`
	OrganizationUuid string `json:"organizationUuid"`
	BillingType      string `json:"billingType"`
	OrganizationRole string `json:"organizationRole"`
	OrganizationName string `json:"organizationName"`
}

type installedPluginEntry struct {
	Scope        string `json:"scope"`
	InstallPath  string `json:"installPath"`
	Version      string `json:"version"`
	InstalledAt  string `json:"installedAt"`
	LastUpdated  string `json:"lastUpdated"`
	GitCommitSha string `json:"gitCommitSha"`
}

type claudeBackupState struct {
	OAuthAccount *oauthAccount          `json:"oauthAccount"`
	Projects     map[string]interface{} `json:"projects"`
}

// projectDirMap returns a map from original project path to encoded directory name.
func (s *claudeBackupState) projectDirMap() map[string]string {
	result := make(map[string]string)
	for path := range s.Projects {
		encoded := pathToProjectDir(path)
		result[path] = encoded
	}
	return result
}

// pathToProjectDir encodes a filesystem path the same way Claude Code does:
// replace all "/" and "." with "-" and prepend "-".
func pathToProjectDir(path string) string {
	s := strings.TrimPrefix(path, "/")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, ".", "-")
	return "-" + s
}

func (r *mqlClaudeCode) loadBackupState() (*claudeBackupState, error) {
	r.backupOnce.Do(func() {
		afs := r.afs()
		dir := r.configDir()

		backupFile, err := findLatestBackupAfero(afs, dir)
		if err != nil {
			r.backupErr = err
			return
		}
		var state claudeBackupState
		if err := readJSONFileAfero(afs, dir, filepath.Join("backups", backupFile), &state); err != nil {
			r.backupErr = err
			return
		}
		r.backupState = &state
	})

	return r.backupState, r.backupErr
}

func (r *mqlClaudeCode) loadOAuthAccount() (*oauthAccount, error) {
	state, err := r.loadBackupState()
	if err != nil {
		return nil, err
	}
	if state.OAuthAccount == nil {
		return &oauthAccount{}, nil
	}
	return state.OAuthAccount, nil
}

func findLatestBackupAfero(afs *afero.Afero, configDir string) (string, error) {
	backupsDir := filepath.Join(configDir, "backups")
	entries, err := afs.ReadDir(backupsDir)
	if err != nil {
		return "", fmt.Errorf("cannot read backups directory: %w", err)
	}

	const prefix = ".claude.json.backup."
	var latestBackup string
	var latestTimestamp int64
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		tsStr := strings.TrimPrefix(entry.Name(), prefix)
		ts, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			continue
		}
		if ts > latestTimestamp {
			latestTimestamp = ts
			latestBackup = entry.Name()
		}
	}

	if latestBackup == "" {
		return "", fmt.Errorf("no backup files found in %s", backupsDir)
	}
	return latestBackup, nil
}

// Stub ID methods for child resources (they use __id set during creation)

func (r *mqlClaudeCodePlugin) id() (string, error) {
	return "claude.code.plugin/" + r.Name.Data + "/" + r.Scope.Data, nil
}

func (r *mqlClaudeCodeSkill) id() (string, error) {
	return "claude.code.skill/" + r.Name.Data, nil
}

func (r *mqlClaudeCodeSkill) sha256() (string, error) {
	return contentSHA256(r.Content.Data), nil
}

func (r *mqlClaudeCodeProject) id() (string, error) {
	return "claude.code.project/" + r.Path.Data, nil
}

func (r *mqlClaudeCodeMcpServer) id() (string, error) {
	return "claude.code.mcpServer/" + r.Name.Data, nil
}
