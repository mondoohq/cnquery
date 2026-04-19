// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"os"
	"path/filepath"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

// skillsFromDir reads SKILL.md files from subdirectories of skillsDir and
// returns them as MQL resources of the given resourceType. This is the shared
// implementation used by all simple skill-only agent resources.
func skillsFromDir(runtime *plugin.Runtime, skillsDir, resourceType string) ([]interface{}, error) {
	afs := connectionAfs(runtime)

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

		res, err := NewResource(runtime, resourceType, map[string]*llx.RawData{
			"__id":         llx.StringData(resourceType + "/" + dir.name),
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

// --- Roo Code ---

const defaultRooConfigDir = ".roo"

func initRoo(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "roo", defaultRooConfigDir)
}

func (r *mqlRoo) id() (string, error) {
	return "roo/" + r.ConfigPath.Data, nil
}

func (r *mqlRoo) skills() ([]interface{}, error) {
	return skillsFromDir(r.MqlRuntime, filepath.Join(r.ConfigPath.Data, "skills"), "roo.skill")
}

func (r *mqlRooSkill) id() (string, error)     { return "roo.skill/" + r.Name.Data, nil }
func (r *mqlRooSkill) sha256() (string, error) { return contentSHA256(r.Content.Data), nil }

// --- Cline ---

const defaultClineConfigDir = ".cline"

func initCline(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "cline", defaultClineConfigDir)
}

func (r *mqlCline) id() (string, error) {
	return "cline/" + r.ConfigPath.Data, nil
}

func (r *mqlCline) skills() ([]interface{}, error) {
	// Cline uses a shared ~/.agents/skills/ directory (not relative to its
	// own ~/.cline config dir). This matches the vercel-labs/skills definition.
	// When a custom configPath is provided, derive the skills path from the
	// same parent directory to stay consistent.
	skillsDir := filepath.Join(filepath.Dir(r.ConfigPath.Data), ".agents", "skills")
	return skillsFromDir(r.MqlRuntime, skillsDir, "cline.skill")
}

func (r *mqlClineSkill) id() (string, error)     { return "cline.skill/" + r.Name.Data, nil }
func (r *mqlClineSkill) sha256() (string, error) { return contentSHA256(r.Content.Data), nil }

// --- Kiro CLI ---

const defaultKiroConfigDir = ".kiro"

func initKiro(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "kiro", defaultKiroConfigDir)
}

func (r *mqlKiro) id() (string, error) {
	return "kiro/" + r.ConfigPath.Data, nil
}

func (r *mqlKiro) skills() ([]interface{}, error) {
	return skillsFromDir(r.MqlRuntime, filepath.Join(r.ConfigPath.Data, "skills"), "kiro.skill")
}

func (r *mqlKiroSkill) id() (string, error)     { return "kiro.skill/" + r.Name.Data, nil }
func (r *mqlKiroSkill) sha256() (string, error) { return contentSHA256(r.Content.Data), nil }

// --- Continue ---

const defaultContinueConfigDir = ".continue"

func initContinuedev(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "continuedev", defaultContinueConfigDir)
}

func (r *mqlContinuedev) id() (string, error) {
	return "continuedev/" + r.ConfigPath.Data, nil
}

func (r *mqlContinuedev) skills() ([]interface{}, error) {
	return skillsFromDir(r.MqlRuntime, filepath.Join(r.ConfigPath.Data, "skills"), "continuedev.skill")
}

func (r *mqlContinuedevSkill) id() (string, error)     { return "continuedev.skill/" + r.Name.Data, nil }
func (r *mqlContinuedevSkill) sha256() (string, error) { return contentSHA256(r.Content.Data), nil }

// --- Trae ---

const defaultTraeConfigDir = ".trae"

func initTrae(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "trae", defaultTraeConfigDir)
}

func (r *mqlTrae) id() (string, error) {
	return "trae/" + r.ConfigPath.Data, nil
}

func (r *mqlTrae) skills() ([]interface{}, error) {
	return skillsFromDir(r.MqlRuntime, filepath.Join(r.ConfigPath.Data, "skills"), "trae.skill")
}

func (r *mqlTraeSkill) id() (string, error)     { return "trae.skill/" + r.Name.Data, nil }
func (r *mqlTraeSkill) sha256() (string, error) { return contentSHA256(r.Content.Data), nil }

// --- OpenCode ---

const defaultOpenCodeConfigDir = ".config/opencode"

func initOpencode(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "opencode", defaultOpenCodeConfigDir)
}

func (r *mqlOpencode) id() (string, error) {
	return "opencode/" + r.ConfigPath.Data, nil
}

func (r *mqlOpencode) skills() ([]interface{}, error) {
	return skillsFromDir(r.MqlRuntime, filepath.Join(r.ConfigPath.Data, "skills"), "opencode.skill")
}

func (r *mqlOpencodeSkill) id() (string, error)     { return "opencode.skill/" + r.Name.Data, nil }
func (r *mqlOpencodeSkill) sha256() (string, error) { return contentSHA256(r.Content.Data), nil }

// --- Pi ---

const defaultPiConfigDir = ".pi/agent"

func initPi(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "pi", defaultPiConfigDir)
}

func (r *mqlPi) id() (string, error) {
	return "pi/" + r.ConfigPath.Data, nil
}

func (r *mqlPi) skills() ([]interface{}, error) {
	return skillsFromDir(r.MqlRuntime, filepath.Join(r.ConfigPath.Data, "skills"), "pi.skill")
}

func (r *mqlPiSkill) id() (string, error)     { return "pi.skill/" + r.Name.Data, nil }
func (r *mqlPiSkill) sha256() (string, error) { return contentSHA256(r.Content.Data), nil }

// --- Mistral Vibe ---

const defaultMistralVibeConfigDir = ".vibe"

func initMistralVibe(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "mistral.vibe", defaultMistralVibeConfigDir)
}

func (r *mqlMistralVibe) id() (string, error) {
	return "mistral.vibe/" + r.ConfigPath.Data, nil
}

func (r *mqlMistralVibe) skills() ([]interface{}, error) {
	return skillsFromDir(r.MqlRuntime, filepath.Join(r.ConfigPath.Data, "skills"), "mistral.vibe.skill")
}

func (r *mqlMistralVibeSkill) id() (string, error)     { return "mistral.vibe.skill/" + r.Name.Data, nil }
func (r *mqlMistralVibeSkill) sha256() (string, error) { return contentSHA256(r.Content.Data), nil }
