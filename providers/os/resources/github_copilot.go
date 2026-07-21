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

const defaultGitHubCopilotConfigDir = ".config/github-copilot"

func initGithubCopilot(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "github.copilot", defaultGitHubCopilotConfigDir)
}

func (r *mqlGithubCopilot) id() (string, error) {
	return "github.copilot/" + r.ConfigPath.Data, nil
}

func (r *mqlGithubCopilot) accounts() ([]interface{}, error) {
	afs := connectionAfs(r.MqlRuntime)
	configDir := r.ConfigPath.Data

	data, err := afs.ReadFile(filepath.Join(configDir, "apps.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// apps.json is a map of "host:appId" -> account info
	var apps map[string]copilotApp
	if err := json.Unmarshal(data, &apps); err != nil {
		return nil, fmt.Errorf("failed to parse github-copilot apps.json: %w", err)
	}

	var result []interface{}
	for key, app := range apps {
		res, err := NewResource(r.MqlRuntime, "github.copilot.account", map[string]*llx.RawData{
			"__id":        llx.StringData("github.copilot.account/" + key),
			"user":        llx.StringData(app.User),
			"githubAppId": llx.StringData(app.GitHubAppID),
		})
		if err != nil {
			return nil, err
		}
		result = append(result, res)
	}
	return result, nil
}

func (r *mqlGithubCopilot) mcpServers() ([]interface{}, error) {
	afs := connectionAfs(r.MqlRuntime)
	configDir := r.ConfigPath.Data

	// Check multiple MCP config locations (VS Code, IntelliJ)
	mcpPaths := []string{
		filepath.Join(configDir, "mcp.json"),
		filepath.Join(configDir, "intellij", "mcp.json"),
	}

	var result []interface{}
	for _, mcpPath := range mcpPaths {
		data, err := afs.ReadFile(mcpPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		var config copilotMCPConfig
		if err := json.Unmarshal(data, &config); err != nil {
			continue // skip malformed files
		}

		for name, server := range config.Servers {
			argsAny := make([]interface{}, len(server.Args))
			for i, a := range server.Args {
				argsAny[i] = a
			}

			res, err := NewResource(r.MqlRuntime, "github.copilot.mcpServer", map[string]*llx.RawData{
				"__id":    llx.StringData("github.copilot.mcpServer/" + name),
				"name":    llx.StringData(name),
				"type":    llx.StringData(server.Type),
				"command": llx.StringData(server.Command),
				"args":    llx.ArrayData(argsAny, types.String),
			})
			if err != nil {
				return nil, err
			}
			result = append(result, res)
		}
	}
	return result, nil
}

func (r *mqlGithubCopilot) skills() ([]interface{}, error) {
	// Copilot skills live at ~/.copilot/skills/, independent of the config dir.
	return skillsAllUsers(r.MqlRuntime, filepath.Join(".copilot", "skills"), "github.copilot.skill")
}

// Child resource ID methods

func (r *mqlGithubCopilotAccount) id() (string, error) {
	return "github.copilot.account/" + r.User.Data, nil
}

func (r *mqlGithubCopilotMcpServer) id() (string, error) {
	return "github.copilot.mcpServer/" + r.Name.Data, nil
}

func (r *mqlGithubCopilotSkill) id() (string, error) {
	return "github.copilot.skill/" + r.Source.Data, nil
}

func (r *mqlGithubCopilotSkill) sha256() (string, error) {
	return contentSHA256(r.Content.Data), nil
}

// Helper types

type copilotApp struct {
	User        string `json:"user"`
	OAuthToken  string `json:"oauth_token"`
	GitHubAppID string `json:"githubAppId"`
}

type copilotMCPConfig struct {
	Servers map[string]copilotMCPServer `json:"servers"`
}

type copilotMCPServer struct {
	Type    string   `json:"type"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}
