// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/types"
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

// targetHomeDir resolves a single user's home as the anchor for a default
// configPath: the logged-in user if any, else the first real user. Data that
// must cover every account (e.g. agent skills) should use targetUserHomes.
func targetHomeDir(runtime *plugin.Runtime) (string, error) {
	users, err := targetUserHomes(runtime)
	if err != nil {
		return "", err
	}

	conn := runtime.Connection.(shared.Connection)
	loggedIn, _ := loggedInUsers(runtime, conn)

	var fallback string
	for _, u := range users {
		// Prefer a user that is currently logged in.
		if loggedIn[u.name] {
			return u.home, nil
		}
		if fallback == "" {
			fallback = u.home
		}
	}

	if fallback != "" {
		return fallback, nil
	}
	return "", fmt.Errorf("no valid user home directory found on target")
}

// collectSkillFiles reads SKILL.md files from the subdirectories of each
// skillsDir, deduped by source path. Unreadable dirs (missing or
// permission-denied when scanning another user's home) are skipped.
func collectSkillFiles(afs *afero.Afero, skillsDirs []string) []skillInfo {
	var result []skillInfo
	seen := map[string]struct{}{}
	for _, skillsDir := range skillsDirs {
		subdirs, err := listSubdirsAfero(afs, skillsDir)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Debug().Err(err).Str("path", skillsDir).Msg("skipping unreadable AI agent skills directory")
			}
			continue
		}

		for _, dir := range subdirs {
			skillPath := filepath.Join(dir.path, "SKILL.md")
			if _, ok := seen[skillPath]; ok {
				continue
			}
			// Mark seen before reading so an unreadable file isn't retried when
			// the same skills dir appears more than once.
			seen[skillPath] = struct{}{}
			data, err := afs.ReadFile(skillPath)
			if err != nil {
				continue
			}
			result = append(result, parseSkillMd(dir.name, skillPath, string(data)))
		}
	}
	return result
}

// readSkillsFromDirs reads SKILL.md files across skillsDirs and returns them as
// MQL resources of resourceType, keyed by source path so same-named skills from
// different users are all reported.
func readSkillsFromDirs(runtime *plugin.Runtime, skillsDirs []string, resourceType string) ([]interface{}, error) {
	skills := collectSkillFiles(connectionAfs(runtime), skillsDirs)

	result := make([]interface{}, 0, len(skills))
	for _, skill := range skills {
		allowedToolsAny := make([]interface{}, len(skill.allowedTools))
		for i, t := range skill.allowedTools {
			allowedToolsAny[i] = t
		}

		res, err := NewResource(runtime, resourceType, map[string]*llx.RawData{
			"__id":         llx.StringData(resourceType + "/" + skill.source),
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

// resolvePerUserDirs resolves a per-user directory list for a resource.
// homeRelDir is the directory relative to each home (e.g. ".cursor/skills").
// When configPath is a per-user default — it matches home/defaultConfigDir for
// some enumerated user — it returns homeRelDir under every user's home, deduped.
// Otherwise (a custom configPath override, or no users could be enumerated) it
// returns just overrideDir, honoring the override exactly.
func resolvePerUserDirs(runtime *plugin.Runtime, configPath, defaultConfigDir, homeRelDir, overrideDir string) []string {
	users, err := targetUserHomes(runtime)
	if err != nil {
		log.Debug().Err(err).Msg("cannot enumerate users; using configPath only")
		return []string{overrideDir}
	}

	// Detect the default case first so the per-user list is built only when used.
	// No match (including an empty user list) means a custom override.
	isDefault := false
	for _, u := range users {
		if filepath.Join(u.home, defaultConfigDir) == configPath {
			isDefault = true
			break
		}
	}
	if !isDefault {
		return []string{overrideDir}
	}

	seen := map[string]struct{}{}
	var dirs []string
	for _, u := range users {
		d := filepath.Join(u.home, homeRelDir)
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		dirs = append(dirs, d)
	}
	return dirs
}

// agentSkills reads an agent's skills across every user, honoring a configPath
// override. For agents whose skills sit at a fixed path under each home.
func agentSkills(runtime *plugin.Runtime, resourceType, configPath, defaultConfigDir, homeRelSkillsDir, overrideSkillsDir string) ([]interface{}, error) {
	dirs := resolvePerUserDirs(runtime, configPath, defaultConfigDir, homeRelSkillsDir, overrideSkillsDir)
	return readSkillsFromDirs(runtime, dirs, resourceType)
}

// skillsAllUsers reads skills from homeRelSkillsDir under every user's home, for
// agents whose skills location is independent of configPath (e.g. Copilot).
func skillsAllUsers(runtime *plugin.Runtime, homeRelSkillsDir, resourceType string) ([]interface{}, error) {
	users, err := targetUserHomes(runtime)
	if err != nil {
		log.Debug().Err(err).Msg("cannot enumerate users for AI agent skills")
		return nil, nil
	}
	dirs := make([]string, 0, len(users))
	for _, u := range users {
		dirs = append(dirs, filepath.Join(u.home, homeRelSkillsDir))
	}
	return readSkillsFromDirs(runtime, dirs, resourceType)
}
