// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"sigs.k8s.io/yaml"
)

// initConfigPath is a shared init helper for resources that resolve a
// configPath from the target's home directory (e.g. claude.code, openai.codex).
func initConfigPath(runtime *plugin.Runtime, args map[string]*llx.RawData, resourceName, defaultDir string) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["configPath"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, fmt.Errorf("wrong type for 'configPath' in %s initialization, it must be a string", resourceName)
		}
		if path == "" {
			delete(args, "configPath")
		}
	}

	if _, ok := args["configPath"]; !ok {
		// Resolve the home directory from the target's user list, not the local host.
		home, err := targetHomeDir(runtime)
		if err != nil {
			return nil, nil, err
		}
		args["configPath"] = llx.StringData(filepath.Join(home, defaultDir))
	}

	return args, nil, nil
}

// connectionAfs returns an afero.Afero wrapping the connection's filesystem.
func connectionAfs(runtime *plugin.Runtime) *afero.Afero {
	conn := runtime.Connection.(shared.Connection)
	return &afero.Afero{Fs: conn.FileSystem()}
}

// listSubdirsAfero returns the names and full paths of subdirectories in dir,
// following symlinks so that symlinked directories are included.
func listSubdirsAfero(afs *afero.Afero, dir string) ([]subdirEntry, error) {
	entries, err := afs.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var result []subdirEntry
	for _, entry := range entries {
		fullPath := filepath.Join(dir, entry.Name())
		// Stat follows symlinks; ReadDir returns Lstat info where
		// symlinked directories report IsDir()==false.
		info, err := afs.Stat(fullPath)
		if err != nil || !info.IsDir() {
			continue
		}
		result = append(result, subdirEntry{name: entry.Name(), path: fullPath})
	}
	return result, nil
}

type subdirEntry struct {
	name string
	path string
}

// readJSONFileAfero reads and unmarshals a JSON file relative to a base directory
// using the provided afero filesystem (which may be remote via SSH, container, etc.).
func readJSONFileAfero(afs *afero.Afero, baseDir string, relPath string, v interface{}) error {
	data, err := afs.ReadFile(filepath.Join(baseDir, relPath))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// dirHasFilesAfero returns true if the directory contains at least one non-directory entry.
func dirHasFilesAfero(afs *afero.Afero, dir string) bool {
	entries, err := afs.ReadDir(dir)
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

// contentSHA256 returns the hex-encoded SHA-256 digest of s.
func contentSHA256(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// Skill parsing types and functions shared by claude.code and openai.codex.

type skillInfo struct {
	name         string
	description  string
	allowedTools []string
	argumentHint string
	source       string
	content      string
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

// targetHomeDir resolves a real user's home directory on the target.
// It prefers a currently logged-in user (via loggedInUsers / "who"),
// falling back to the first non-system user if no one is logged in.
func targetHomeDir(runtime *plugin.Runtime) (string, error) {
	usersResource, err := CreateResource(runtime, "users", map[string]*llx.RawData{})
	if err != nil {
		return "", fmt.Errorf("cannot list users on target: %w", err)
	}

	userList := usersResource.(*mqlUsers).GetList()
	if userList.Error != nil {
		return "", fmt.Errorf("cannot list users on target: %w", userList.Error)
	}

	conn := runtime.Connection.(shared.Connection)
	loggedIn, _ := loggedInUsers(runtime, conn)

	var fallback string
	for _, u := range userList.Data {
		user := u.(*mqlUser)
		home := user.GetHome().Data
		if home == "" || isSystemHomeDir(home) {
			continue
		}
		// Prefer a user that is currently logged in.
		if loggedIn[user.GetName().Data] {
			return home, nil
		}
		if fallback == "" {
			fallback = home
		}
	}

	if fallback != "" {
		return fallback, nil
	}
	return "", fmt.Errorf("no valid user home directory found on target")
}
