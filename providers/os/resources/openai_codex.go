// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

const defaultCodexConfigDir = ".codex"

func initOpenaiCodex(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["configPath"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, fmt.Errorf("wrong type for 'configPath' in openai.codex initialization, it must be a string")
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
		args["configPath"] = llx.StringData(filepath.Join(home, defaultCodexConfigDir))
	}

	return args, nil, nil
}

func (r *mqlOpenaiCodex) id() (string, error) {
	return "openai.codex/" + r.ConfigPath.Data, nil
}

func (r *mqlOpenaiCodex) codexDir() string {
	return r.ConfigPath.Data
}

func (r *mqlOpenaiCodex) authMode() (string, error) {
	auth, err := r.loadAuth()
	if err != nil {
		return "", err
	}
	return auth.AuthMode, nil
}

func (r *mqlOpenaiCodex) accountId() (string, error) {
	auth, err := r.loadAuth()
	if err != nil {
		return "", err
	}
	if auth.Tokens == nil {
		return "", nil
	}
	return auth.Tokens.AccountID, nil
}

func (r *mqlOpenaiCodex) version() (string, error) {
	var ver struct {
		LatestVersion string `json:"latest_version"`
	}
	err := claudeReadJSONFile(r.codexDir(), "version.json", &ver)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return ver.LatestVersion, nil
}

func (r *mqlOpenaiCodex) lastRefresh() (string, error) {
	auth, err := r.loadAuth()
	if err != nil {
		return "", err
	}
	return auth.LastRefresh, nil
}

func (r *mqlOpenaiCodex) plugins() ([]interface{}, error) {
	pluginsDir := filepath.Join(r.codexDir(), ".tmp", "plugins", "plugins")

	entries, err := os.ReadDir(pluginsDir)
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

		pluginName := entry.Name()
		pluginDir := filepath.Join(pluginsDir, pluginName)

		p := codexPluginInfo{name: pluginName}

		// Read plugin.json
		var pj codexPluginJSON
		if err := claudeReadJSONFile(pluginDir, ".codex-plugin/plugin.json", &pj); err == nil {
			p.version = pj.Version
			p.description = pj.Description
			if pj.Author != nil {
				p.author = pj.Author.Name
			}
			if pj.Interface != nil {
				p.category = pj.Interface.Category
				p.capabilities = pj.Interface.Capabilities
			}
		}

		// Collect skill names from skills directory
		skillsDir := filepath.Join(pluginDir, "skills")
		if skillEntries, err := os.ReadDir(skillsDir); err == nil {
			for _, se := range skillEntries {
				if se.IsDir() {
					p.skillNames = append(p.skillNames, se.Name())
				}
			}
		}

		// Check for MCP config
		_, mcpErr := os.Stat(filepath.Join(pluginDir, ".mcp.json"))
		p.hasMcp = mcpErr == nil

		// Check for hooks
		_, hooksErr := os.Stat(filepath.Join(pluginDir, "hooks.json"))
		p.hasHooks = hooksErr == nil

		capAny := make([]interface{}, len(p.capabilities))
		for i, c := range p.capabilities {
			capAny[i] = c
		}
		skillAny := make([]interface{}, len(p.skillNames))
		for i, s := range p.skillNames {
			skillAny[i] = s
		}

		res, err := NewResource(r.MqlRuntime, "openai.codex.plugin", map[string]*llx.RawData{
			"__id":         llx.StringData("openai.codex.plugin/" + pluginName),
			"name":         llx.StringData(pluginName),
			"version":      llx.StringData(p.version),
			"description":  llx.StringData(p.description),
			"author":       llx.StringData(p.author),
			"category":     llx.StringData(p.category),
			"capabilities": llx.ArrayData(capAny, types.String),
			"skillNames":   llx.ArrayData(skillAny, types.String),
			"hasMcp":       llx.BoolData(p.hasMcp),
			"hasHooks":     llx.BoolData(p.hasHooks),
		})
		if err != nil {
			return nil, err
		}
		result = append(result, res)
	}
	return result, nil
}

func (r *mqlOpenaiCodex) skills() ([]interface{}, error) {
	var result []interface{}

	// Collect system skills
	systemSkillsDir := filepath.Join(r.codexDir(), "skills", ".system")
	if entries, err := os.ReadDir(systemSkillsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillPath := filepath.Join(systemSkillsDir, entry.Name(), "SKILL.md")
			data, err := os.ReadFile(skillPath)
			if err != nil {
				continue
			}
			skill := parseSkillMd(entry.Name(), skillPath, string(data))
			res, err := newCodexSkillResource(r.MqlRuntime, skill, "system")
			if err != nil {
				return nil, err
			}
			result = append(result, res)
		}
	}

	// Collect plugin skills
	pluginsDir := filepath.Join(r.codexDir(), ".tmp", "plugins", "plugins")
	if pluginEntries, err := os.ReadDir(pluginsDir); err == nil {
		for _, pe := range pluginEntries {
			if !pe.IsDir() {
				continue
			}
			pluginName := pe.Name()
			skillsDir := filepath.Join(pluginsDir, pluginName, "skills")
			skillEntries, err := os.ReadDir(skillsDir)
			if err != nil {
				continue
			}
			for _, se := range skillEntries {
				if !se.IsDir() {
					continue
				}
				skillPath := filepath.Join(skillsDir, se.Name(), "SKILL.md")
				data, err := os.ReadFile(skillPath)
				if err != nil {
					continue
				}
				skill := parseSkillMd(se.Name(), skillPath, string(data))
				res, err := newCodexSkillResource(r.MqlRuntime, skill, pluginName)
				if err != nil {
					return nil, err
				}
				result = append(result, res)
			}
		}
	}

	return result, nil
}

func (r *mqlOpenaiCodex) mcpServers() ([]interface{}, error) {
	pluginsDir := filepath.Join(r.codexDir(), ".tmp", "plugins", "plugins")

	entries, err := os.ReadDir(pluginsDir)
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
		pluginName := entry.Name()
		mcpPath := filepath.Join(pluginsDir, pluginName, ".mcp.json")

		var mcpConfig struct {
			McpServers map[string]codexMcpServerEntry `json:"mcpServers"`
		}
		data, err := os.ReadFile(mcpPath)
		if err != nil {
			continue
		}
		if err := json.Unmarshal(data, &mcpConfig); err != nil {
			continue
		}

		for name, srv := range mcpConfig.McpServers {
			res, err := NewResource(r.MqlRuntime, "openai.codex.mcpServer", map[string]*llx.RawData{
				"__id":   llx.StringData("openai.codex.mcpServer/" + pluginName + "/" + name),
				"name":   llx.StringData(name),
				"type":   llx.StringData(srv.Type),
				"url":    llx.StringData(srv.URL),
				"note":   llx.StringData(srv.Note),
				"plugin": llx.StringData(pluginName),
			})
			if err != nil {
				return nil, err
			}
			result = append(result, res)
		}
	}
	return result, nil
}

func (r *mqlOpenaiCodex) connectors() ([]interface{}, error) {
	pluginsDir := filepath.Join(r.codexDir(), ".tmp", "plugins", "plugins")

	entries, err := os.ReadDir(pluginsDir)
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
		pluginName := entry.Name()

		var appConfig struct {
			Apps map[string]struct {
				ID string `json:"id"`
			} `json:"apps"`
		}
		if err := claudeReadJSONFile(filepath.Join(pluginsDir, pluginName), ".app.json", &appConfig); err != nil {
			continue
		}

		for connName, app := range appConfig.Apps {
			res, err := NewResource(r.MqlRuntime, "openai.codex.connector", map[string]*llx.RawData{
				"__id":   llx.StringData("openai.codex.connector/" + pluginName + "/" + connName),
				"name":   llx.StringData(connName),
				"id":     llx.StringData(app.ID),
				"plugin": llx.StringData(pluginName),
			})
			if err != nil {
				return nil, err
			}
			result = append(result, res)
		}
	}
	return result, nil
}

// Helper types

type codexAuthJSON struct {
	AuthMode    string           `json:"auth_mode"`
	Tokens      *codexTokensJSON `json:"tokens"`
	LastRefresh string           `json:"last_refresh"`
}

type codexTokensJSON struct {
	AccountID string `json:"account_id"`
}

type codexPluginJSON struct {
	Version     string              `json:"version"`
	Description string              `json:"description"`
	Author      *codexAuthorJSON    `json:"author"`
	Interface   *codexInterfaceJSON `json:"interface"`
}

type codexAuthorJSON struct {
	Name string `json:"name"`
}

type codexInterfaceJSON struct {
	Category     string   `json:"category"`
	Capabilities []string `json:"capabilities"`
}

type codexMcpServerEntry struct {
	Type string `json:"type"`
	URL  string `json:"url"`
	Note string `json:"note"`
}

type codexPluginInfo struct {
	name         string
	version      string
	description  string
	author       string
	category     string
	capabilities []string
	skillNames   []string
	hasMcp       bool
	hasHooks     bool
}

func (r *mqlOpenaiCodex) loadAuth() (*codexAuthJSON, error) {
	var auth codexAuthJSON
	err := claudeReadJSONFile(r.codexDir(), "auth.json", &auth)
	if err != nil {
		if os.IsNotExist(err) {
			return &codexAuthJSON{}, nil
		}
		return nil, err
	}
	return &auth, nil
}

func newCodexSkillResource(runtime *plugin.Runtime, skill skillInfo, pluginName string) (*mqlOpenaiCodexSkill, error) {
	res, err := NewResource(runtime, "openai.codex.skill", map[string]*llx.RawData{
		"__id":        llx.StringData("openai.codex.skill/" + pluginName + "/" + skill.name),
		"name":        llx.StringData(skill.name),
		"description": llx.StringData(skill.description),
		"source":      llx.StringData(skill.source),
		"plugin":      llx.StringData(pluginName),
		"content":     llx.StringData(skill.content),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenaiCodexSkill), nil
}

// Stub ID methods for child resources

func (r *mqlOpenaiCodexPlugin) id() (string, error) {
	return "openai.codex.plugin/" + r.Name.Data, nil
}

func (r *mqlOpenaiCodexSkill) id() (string, error) {
	return "openai.codex.skill/" + r.Plugin.Data + "/" + r.Name.Data, nil
}

func (r *mqlOpenaiCodexMcpServer) id() (string, error) {
	return "openai.codex.mcpServer/" + r.Plugin.Data + "/" + r.Name.Data, nil
}

func (r *mqlOpenaiCodexConnector) id() (string, error) {
	return "openai.codex.connector/" + r.Plugin.Data + "/" + r.Name.Data, nil
}
