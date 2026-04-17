// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

const defaultCursorConfigDir = ".cursor"

func initCursor(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "cursor", defaultCursorConfigDir)
}

func (r *mqlCursor) id() (string, error) {
	return "cursor/" + r.ConfigPath.Data, nil
}

func (r *mqlCursor) mcpServers() ([]interface{}, error) {
	afs := connectionAfs(r.MqlRuntime)
	configDir := r.ConfigPath.Data

	// Cursor stores MCP config in mcp.json
	data, err := afs.ReadFile(filepath.Join(configDir, "mcp.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var mcpConfig cursorMCPConfig
	if err := json.Unmarshal(data, &mcpConfig); err != nil {
		return nil, fmt.Errorf("failed to parse cursor mcp.json: %w", err)
	}

	var result []interface{}
	for name, server := range mcpConfig.McpServers {
		argsAny := make([]interface{}, len(server.Args))
		for i, a := range server.Args {
			argsAny[i] = a
		}

		res, err := NewResource(r.MqlRuntime, "cursor.mcpServer", map[string]*llx.RawData{
			"__id":    llx.StringData("cursor.mcpServer/" + name),
			"name":    llx.StringData(name),
			"command": llx.StringData(server.Command),
			"url":     llx.StringData(server.URL),
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

func (r *mqlCursor) rules() ([]interface{}, error) {
	afs := connectionAfs(r.MqlRuntime)
	rulesDir := filepath.Join(r.ConfigPath.Data, "rules")

	entries, err := afs.ReadDir(rulesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []interface{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// Only process markdown and text rule files
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".md" && ext != ".txt" && ext != ".mdc" && ext != "" {
			continue
		}

		rulePath := filepath.Join(rulesDir, name)
		data, err := afs.ReadFile(rulePath)
		if err != nil {
			continue
		}

		ruleName := strings.TrimSuffix(name, filepath.Ext(name))
		res, err := NewResource(r.MqlRuntime, "cursor.rule", map[string]*llx.RawData{
			"__id":    llx.StringData("cursor.rule/" + name),
			"name":    llx.StringData(ruleName),
			"content": llx.StringData(string(data)),
			"source":  llx.StringData(rulePath),
		})
		if err != nil {
			return nil, err
		}
		result = append(result, res)
	}
	return result, nil
}

func (r *mqlCursor) skills() ([]interface{}, error) {
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

		res, err := NewResource(r.MqlRuntime, "cursor.skill", map[string]*llx.RawData{
			"__id":         llx.StringData("cursor.skill/" + dir.name),
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

func (r *mqlCursorMcpServer) id() (string, error) {
	return "cursor.mcpServer/" + r.Name.Data, nil
}

func (r *mqlCursorRule) id() (string, error) {
	return "cursor.rule/" + r.Name.Data, nil
}

func (r *mqlCursorSkill) id() (string, error) {
	return "cursor.skill/" + r.Name.Data, nil
}

func (r *mqlCursorSkill) sha256() (string, error) {
	return contentSHA256(r.Content.Data), nil
}

// Helper types

type cursorMCPConfig struct {
	McpServers map[string]cursorMCPServer `json:"mcpServers"`
}

type cursorMCPServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	URL     string            `json:"url"`
	Env     map[string]string `json:"env"`
}
