// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
	"sigs.k8s.io/yaml"
)

const defaultClaudeCodeConfigDir = ".claude"

func initClaudeCode(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["configPath"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, fmt.Errorf("wrong type for 'configPath' in claude.code initialization, it must be a string")
		}
		if path == "" {
			delete(args, "configPath")
		}
	}

	if _, ok := args["configPath"]; !ok {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, nil, fmt.Errorf("cannot determine user home directory: %w", err)
		}
		args["configPath"] = llx.StringData(filepath.Join(home, defaultClaudeCodeConfigDir))
	}

	return args, nil, nil
}

func (r *mqlClaudeCode) id() (string, error) {
	return "claude.code/" + r.ConfigPath.Data, nil
}

// configDir returns the configPath for this resource instance.
func (r *mqlClaudeCode) configDir() string {
	return r.ConfigPath.Data
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
	err := claudeReadJSONFile(r.configDir(), "settings.json", &settings)
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
	err := claudeReadJSONFile(r.configDir(), "settings.json", &settings)
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
	var installedPlugins struct {
		Version int                               `json:"version"`
		Plugins map[string][]installedPluginEntry `json:"plugins"`
	}
	err := claudeReadJSONFile(r.configDir(), "plugins/installed_plugins.json", &installedPlugins)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var settings struct {
		EnabledPlugins map[string]bool `json:"enabledPlugins"`
	}
	_ = claudeReadJSONFile(r.configDir(), "settings.json", &settings)

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
	skillsDir := filepath.Join(r.configDir(), "skills")

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []interface{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
		data, err := os.ReadFile(skillPath)
		if err != nil {
			continue
		}

		skill := parseSkillMd(entry.Name(), skillPath, string(data))

		allowedToolsAny := make([]interface{}, len(skill.allowedTools))
		for i, t := range skill.allowedTools {
			allowedToolsAny[i] = t
		}

		res, err := NewResource(r.MqlRuntime, "claude.code.skill", map[string]*llx.RawData{
			"__id":         llx.StringData("claude.code.skill/" + entry.Name()),
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
	state, err := r.loadBackupState()
	if err != nil {
		return nil, err
	}

	projectsDir := filepath.Join(r.configDir(), "projects")
	var result []interface{}
	for projectPath, dirName := range state.projectDirMap() {
		memoryDir := filepath.Join(projectsDir, dirName, "memory")
		hasMemory := dirHasFiles(memoryDir)

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
	err := claudeReadJSONFile(r.configDir(), "mcp-needs-auth-cache.json", &cache)
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

type skillInfo struct {
	name         string
	description  string
	allowedTools []string
	argumentHint string
	source       string
	content      string
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

// backupStateOnce caches the backup state per resource instance,
// loaded at most once via sync.Once.
type backupStateOnce struct {
	once  sync.Once
	state *claudeBackupState
	err   error
}

var (
	backupStateInstances   = make(map[string]*backupStateOnce)
	backupStateInstancesMu sync.Mutex
)

func (r *mqlClaudeCode) loadBackupState() (*claudeBackupState, error) {
	dir := r.configDir()

	backupStateInstancesMu.Lock()
	bso, ok := backupStateInstances[dir]
	if !ok {
		bso = &backupStateOnce{}
		backupStateInstances[dir] = bso
	}
	backupStateInstancesMu.Unlock()

	bso.once.Do(func() {
		backupFile, err := findLatestBackup(dir)
		if err != nil {
			bso.err = err
			return
		}
		var state claudeBackupState
		if err := claudeReadJSONFile(dir, filepath.Join("backups", backupFile), &state); err != nil {
			bso.err = err
			return
		}
		bso.state = &state
	})

	return bso.state, bso.err
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

func findLatestBackup(configDir string) (string, error) {
	backupsDir := filepath.Join(configDir, "backups")
	entries, err := os.ReadDir(backupsDir)
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

// claudeReadJSONFile reads and unmarshals a JSON file relative to a base directory.
func claudeReadJSONFile(baseDir string, relPath string, v interface{}) error {
	data, err := os.ReadFile(filepath.Join(baseDir, relPath))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

type skillFrontmatter struct {
	Name         string `json:"name" yaml:"name"`
	Description  string `json:"description" yaml:"description"`
	AllowedTools string `json:"allowed-tools" yaml:"allowed-tools"`
	ArgumentHint string `json:"argument-hint" yaml:"argument-hint"`
}

func parseSkillMd(name, sourcePath, content string) skillInfo {
	info := skillInfo{
		name:    name,
		source:  sourcePath,
		content: content,
	}

	// Extract YAML frontmatter between --- delimiters
	if !strings.HasPrefix(content, "---\n") {
		return info
	}

	endIdx := strings.Index(content[4:], "\n---")
	if endIdx == -1 {
		return info
	}

	frontmatter := content[4 : 4+endIdx]
	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(frontmatter), &fm); err != nil {
		return info
	}

	if fm.Name != "" {
		info.name = fm.Name
	}
	info.description = fm.Description
	info.argumentHint = fm.ArgumentHint

	// allowed-tools is a comma-separated string in both Claude Code and Codex SKILL.md files
	if fm.AllowedTools != "" {
		for _, tool := range strings.Split(fm.AllowedTools, ",") {
			tool = strings.TrimSpace(tool)
			if tool != "" {
				info.allowedTools = append(info.allowedTools, tool)
			}
		}
	}

	return info
}

func dirHasFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() {
			return true
		}
	}
	return false
}

// Stub ID methods for child resources (they use __id set during creation)

func (r *mqlClaudeCodePlugin) id() (string, error) {
	return "claude.code.plugin/" + r.Name.Data + "/" + r.Scope.Data, nil
}

func (r *mqlClaudeCodeSkill) id() (string, error) {
	return "claude.code.skill/" + r.Name.Data, nil
}

func (r *mqlClaudeCodeProject) id() (string, error) {
	return "claude.code.project/" + r.Path.Data, nil
}

func (r *mqlClaudeCodeMcpServer) id() (string, error) {
	return "claude.code.mcpServer/" + r.Name.Data, nil
}
