// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/types"
)

const defaultCursorConfigDir = ".cursor"

func initCursor(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["configPath"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, fmt.Errorf("wrong type for 'configPath' in cursor initialization, it must be a string")
		}
		if path == "" {
			delete(args, "configPath")
		}
	}

	if _, ok := args["configPath"]; !ok {
		home, err := targetHomeDir(runtime)
		if err != nil {
			return nil, nil, err
		}
		args["configPath"] = llx.StringData(filepath.Join(home, defaultCursorConfigDir))
	}

	return args, nil, nil
}

func (r *mqlCursor) id() (string, error) {
	return "cursor/" + r.ConfigPath.Data, nil
}

func (r *mqlCursor) afs() *afero.Afero {
	conn := r.MqlRuntime.Connection.(shared.Connection)
	return &afero.Afero{Fs: conn.FileSystem()}
}

func (r *mqlCursor) mcpServers() ([]interface{}, error) {
	afs := r.afs()
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
	afs := r.afs()
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

// Child resource ID methods

func (r *mqlCursorMcpServer) id() (string, error) {
	return "cursor.mcpServer/" + r.Name.Data, nil
}

func (r *mqlCursorRule) id() (string, error) {
	return "cursor.rule/" + r.Name.Data, nil
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
