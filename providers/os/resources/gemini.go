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

const defaultGeminiConfigDir = ".gemini"

func initGemini(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "gemini", defaultGeminiConfigDir)
}

func (r *mqlGemini) id() (string, error) {
	return "gemini/" + r.ConfigPath.Data, nil
}

func (r *mqlGemini) authType() (string, error) {
	afs := connectionAfs(r.MqlRuntime)
	var settings geminiSettings
	err := readJSONFileAfero(afs, r.ConfigPath.Data, "settings.json", &settings)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return settings.SelectedAuthType, nil
}

func (r *mqlGemini) settings() (interface{}, error) {
	afs := connectionAfs(r.MqlRuntime)
	var settings map[string]interface{}
	err := readJSONFileAfero(afs, r.ConfigPath.Data, "settings.json", &settings)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}
	return settings, nil
}

func (r *mqlGemini) mcpServers() ([]interface{}, error) {
	afs := connectionAfs(r.MqlRuntime)
	configDir := r.ConfigPath.Data

	// Gemini stores MCP config in antigravity/mcp_config.json
	data, err := afs.ReadFile(filepath.Join(configDir, "antigravity", "mcp_config.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var config geminiMCPConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse gemini mcp_config.json: %w", err)
	}

	var result []interface{}
	for name, server := range config.McpServers {
		argsAny := make([]interface{}, len(server.Args))
		for i, a := range server.Args {
			argsAny[i] = a
		}

		res, err := NewResource(r.MqlRuntime, "gemini.mcpServer", map[string]*llx.RawData{
			"__id":    llx.StringData("gemini.mcpServer/" + name),
			"name":    llx.StringData(name),
			"command": llx.StringData(server.Command),
			"args":    llx.ArrayData(argsAny, types.String),
			"hasEnv":  llx.BoolData(len(server.Env) > 0),
		})
		if err != nil {
			return nil, err
		}
		result = append(result, res)
	}
	return result, nil
}

func (r *mqlGemini) skills() ([]interface{}, error) {
	afs := connectionAfs(r.MqlRuntime)
	skillsDir := filepath.Join(r.ConfigPath.Data, "skills")

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

		res, err := NewResource(r.MqlRuntime, "gemini.skill", map[string]*llx.RawData{
			"__id":         llx.StringData("gemini.skill/" + dir.name),
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

// Child resource ID methods

func (r *mqlGeminiMcpServer) id() (string, error) {
	return "gemini.mcpServer/" + r.Name.Data, nil
}

func (r *mqlGeminiSkill) id() (string, error) {
	return "gemini.skill/" + r.Name.Data, nil
}

func (r *mqlGeminiSkill) sha256() (string, error) {
	return contentSHA256(r.Content.Data), nil
}

// Helper types

type geminiSettings struct {
	Theme            string `json:"theme"`
	SelectedAuthType string `json:"selectedAuthType"`
}

type geminiMCPConfig struct {
	McpServers map[string]geminiMCPServer `json:"mcpServers"`
}

type geminiMCPServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}
