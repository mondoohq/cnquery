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

// --- Antigravity (Google) ---

const defaultAntigravityConfigDir = ".gemini/antigravity"

func initAntigravity(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "antigravity", defaultAntigravityConfigDir)
}

func (r *mqlAntigravity) id() (string, error) {
	return "antigravity/" + r.ConfigPath.Data, nil
}

func (r *mqlAntigravity) skills() ([]interface{}, error) {
	return skillsFromDir(r.MqlRuntime, filepath.Join(r.ConfigPath.Data, "skills"), "antigravity.skill")
}

func (r *mqlAntigravitySkill) id() (string, error)     { return "antigravity.skill/" + r.Name.Data, nil }
func (r *mqlAntigravitySkill) sha256() (string, error) { return contentSHA256(r.Content.Data), nil }

// --- IBM Bob ---

const defaultBobConfigDir = ".bob"

func initIbmBob(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "ibm.bob", defaultBobConfigDir)
}

func (r *mqlIbmBob) id() (string, error) {
	return "ibm.bob/" + r.ConfigPath.Data, nil
}

func (r *mqlIbmBob) skills() ([]interface{}, error) {
	return skillsFromDir(r.MqlRuntime, filepath.Join(r.ConfigPath.Data, "skills"), "ibm.bob.skill")
}

func (r *mqlIbmBobSkill) id() (string, error)     { return "ibm.bob.skill/" + r.Name.Data, nil }
func (r *mqlIbmBobSkill) sha256() (string, error) { return contentSHA256(r.Content.Data), nil }

// --- OpenClaw ---

const defaultOpenClawConfigDir = ".openclaw"

func initOpenclaw(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "openclaw", defaultOpenClawConfigDir)
}

func (r *mqlOpenclaw) id() (string, error) {
	return "openclaw/" + r.ConfigPath.Data, nil
}

func (r *mqlOpenclaw) skills() ([]interface{}, error) {
	return skillsFromDir(r.MqlRuntime, filepath.Join(r.ConfigPath.Data, "skills"), "openclaw.skill")
}

func (r *mqlOpenclawSkill) id() (string, error)     { return "openclaw.skill/" + r.Name.Data, nil }
func (r *mqlOpenclawSkill) sha256() (string, error) { return contentSHA256(r.Content.Data), nil }

// --- Snowflake Cortex Code ---

const defaultCortexConfigDir = ".snowflake/cortex"

func initSnowflakeCortex(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "snowflake.cortex", defaultCortexConfigDir)
}

func (r *mqlSnowflakeCortex) id() (string, error) {
	return "snowflake.cortex/" + r.ConfigPath.Data, nil
}

func (r *mqlSnowflakeCortex) skills() ([]interface{}, error) {
	return skillsFromDir(r.MqlRuntime, filepath.Join(r.ConfigPath.Data, "skills"), "snowflake.cortex.skill")
}

func (r *mqlSnowflakeCortexSkill) id() (string, error) {
	return "snowflake.cortex.skill/" + r.Name.Data, nil
}
func (r *mqlSnowflakeCortexSkill) sha256() (string, error) { return contentSHA256(r.Content.Data), nil }

// --- Junie (JetBrains) ---

const defaultJunieConfigDir = ".junie"

func initJunie(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "junie", defaultJunieConfigDir)
}

func (r *mqlJunie) id() (string, error) {
	return "junie/" + r.ConfigPath.Data, nil
}

func (r *mqlJunie) skills() ([]interface{}, error) {
	return skillsFromDir(r.MqlRuntime, filepath.Join(r.ConfigPath.Data, "skills"), "junie.skill")
}

func (r *mqlJunieSkill) id() (string, error)     { return "junie.skill/" + r.Name.Data, nil }
func (r *mqlJunieSkill) sha256() (string, error) { return contentSHA256(r.Content.Data), nil }

// --- Augment ---

const defaultAugmentConfigDir = ".augment"

func initAugment(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "augment", defaultAugmentConfigDir)
}

func (r *mqlAugment) id() (string, error) {
	return "augment/" + r.ConfigPath.Data, nil
}

func (r *mqlAugment) skills() ([]interface{}, error) {
	return skillsFromDir(r.MqlRuntime, filepath.Join(r.ConfigPath.Data, "skills"), "augment.skill")
}

func (r *mqlAugmentSkill) id() (string, error)     { return "augment.skill/" + r.Name.Data, nil }
func (r *mqlAugmentSkill) sha256() (string, error) { return contentSHA256(r.Content.Data), nil }

// --- Warp ---

const defaultWarpConfigDir = ".warp"

func initWarp(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "warp", defaultWarpConfigDir)
}

func (r *mqlWarp) id() (string, error) {
	return "warp/" + r.ConfigPath.Data, nil
}

func (r *mqlWarp) skills() ([]interface{}, error) {
	// Warp uses shared ~/.agents/skills/ directory
	skillsDir := filepath.Join(filepath.Dir(r.ConfigPath.Data), ".agents", "skills")
	return skillsFromDir(r.MqlRuntime, skillsDir, "warp.skill")
}

func (r *mqlWarpSkill) id() (string, error)     { return "warp.skill/" + r.Name.Data, nil }
func (r *mqlWarpSkill) sha256() (string, error) { return contentSHA256(r.Content.Data), nil }

// --- Kilo Code ---

const defaultKiloCodeConfigDir = ".kilocode"

func initKilocode(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "kilocode", defaultKiloCodeConfigDir)
}

func (r *mqlKilocode) id() (string, error) {
	return "kilocode/" + r.ConfigPath.Data, nil
}

func (r *mqlKilocode) skills() ([]interface{}, error) {
	return skillsFromDir(r.MqlRuntime, filepath.Join(r.ConfigPath.Data, "skills"), "kilocode.skill")
}

func (r *mqlKilocodeSkill) id() (string, error)     { return "kilocode.skill/" + r.Name.Data, nil }
func (r *mqlKilocodeSkill) sha256() (string, error) { return contentSHA256(r.Content.Data), nil }

// --- OpenHands ---

const defaultOpenHandsConfigDir = ".openhands"

func initOpenhands(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "openhands", defaultOpenHandsConfigDir)
}

func (r *mqlOpenhands) id() (string, error) {
	return "openhands/" + r.ConfigPath.Data, nil
}

func (r *mqlOpenhands) skills() ([]interface{}, error) {
	return skillsFromDir(r.MqlRuntime, filepath.Join(r.ConfigPath.Data, "skills"), "openhands.skill")
}

func (r *mqlOpenhandsSkill) id() (string, error)     { return "openhands.skill/" + r.Name.Data, nil }
func (r *mqlOpenhandsSkill) sha256() (string, error) { return contentSHA256(r.Content.Data), nil }

// --- Qwen Code ---

const defaultQwenCodeConfigDir = ".qwen"

func initQwenCode(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "qwen.code", defaultQwenCodeConfigDir)
}

func (r *mqlQwenCode) id() (string, error) {
	return "qwen.code/" + r.ConfigPath.Data, nil
}

func (r *mqlQwenCode) skills() ([]interface{}, error) {
	return skillsFromDir(r.MqlRuntime, filepath.Join(r.ConfigPath.Data, "skills"), "qwen.code.skill")
}

func (r *mqlQwenCodeSkill) id() (string, error)     { return "qwen.code.skill/" + r.Name.Data, nil }
func (r *mqlQwenCodeSkill) sha256() (string, error) { return contentSHA256(r.Content.Data), nil }
