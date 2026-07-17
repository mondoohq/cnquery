// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"path/filepath"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// --- Roo Code ---

const defaultRooConfigDir = ".roo"

func initRoo(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "roo", defaultRooConfigDir)
}

func (r *mqlRoo) id() (string, error) {
	return "roo/" + r.ConfigPath.Data, nil
}

func (r *mqlRoo) skills() ([]interface{}, error) {
	return agentSkills(r.MqlRuntime, "roo.skill", r.ConfigPath.Data, defaultRooConfigDir,
		filepath.Join(defaultRooConfigDir, "skills"), filepath.Join(r.ConfigPath.Data, "skills"))
}

func (r *mqlRooSkill) id() (string, error)     { return "roo.skill/" + r.Source.Data, nil }
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
	// Cline reads skills from the shared ~/.agents/skills directory (not its own
	// ~/.cline config dir), matching the vercel-labs/skills convention.
	return agentSkills(r.MqlRuntime, "cline.skill", r.ConfigPath.Data, defaultClineConfigDir,
		filepath.Join(".agents", "skills"),
		filepath.Join(filepath.Dir(r.ConfigPath.Data), ".agents", "skills"))
}

func (r *mqlClineSkill) id() (string, error)     { return "cline.skill/" + r.Source.Data, nil }
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
	return agentSkills(r.MqlRuntime, "kiro.skill", r.ConfigPath.Data, defaultKiroConfigDir,
		filepath.Join(defaultKiroConfigDir, "skills"), filepath.Join(r.ConfigPath.Data, "skills"))
}

func (r *mqlKiroSkill) id() (string, error)     { return "kiro.skill/" + r.Source.Data, nil }
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
	return agentSkills(r.MqlRuntime, "continuedev.skill", r.ConfigPath.Data, defaultContinueConfigDir,
		filepath.Join(defaultContinueConfigDir, "skills"), filepath.Join(r.ConfigPath.Data, "skills"))
}

func (r *mqlContinuedevSkill) id() (string, error)     { return "continuedev.skill/" + r.Source.Data, nil }
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
	return agentSkills(r.MqlRuntime, "trae.skill", r.ConfigPath.Data, defaultTraeConfigDir,
		filepath.Join(defaultTraeConfigDir, "skills"), filepath.Join(r.ConfigPath.Data, "skills"))
}

func (r *mqlTraeSkill) id() (string, error)     { return "trae.skill/" + r.Source.Data, nil }
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
	return agentSkills(r.MqlRuntime, "opencode.skill", r.ConfigPath.Data, defaultOpenCodeConfigDir,
		filepath.Join(defaultOpenCodeConfigDir, "skills"), filepath.Join(r.ConfigPath.Data, "skills"))
}

func (r *mqlOpencodeSkill) id() (string, error)     { return "opencode.skill/" + r.Source.Data, nil }
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
	return agentSkills(r.MqlRuntime, "pi.skill", r.ConfigPath.Data, defaultPiConfigDir,
		filepath.Join(defaultPiConfigDir, "skills"), filepath.Join(r.ConfigPath.Data, "skills"))
}

func (r *mqlPiSkill) id() (string, error)     { return "pi.skill/" + r.Source.Data, nil }
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
	return agentSkills(r.MqlRuntime, "mistral.vibe.skill", r.ConfigPath.Data, defaultMistralVibeConfigDir,
		filepath.Join(defaultMistralVibeConfigDir, "skills"), filepath.Join(r.ConfigPath.Data, "skills"))
}

func (r *mqlMistralVibeSkill) id() (string, error)     { return "mistral.vibe.skill/" + r.Source.Data, nil }
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
	return agentSkills(r.MqlRuntime, "antigravity.skill", r.ConfigPath.Data, defaultAntigravityConfigDir,
		filepath.Join(defaultAntigravityConfigDir, "skills"), filepath.Join(r.ConfigPath.Data, "skills"))
}

func (r *mqlAntigravitySkill) id() (string, error)     { return "antigravity.skill/" + r.Source.Data, nil }
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
	return agentSkills(r.MqlRuntime, "ibm.bob.skill", r.ConfigPath.Data, defaultBobConfigDir,
		filepath.Join(defaultBobConfigDir, "skills"), filepath.Join(r.ConfigPath.Data, "skills"))
}

func (r *mqlIbmBobSkill) id() (string, error)     { return "ibm.bob.skill/" + r.Source.Data, nil }
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
	return agentSkills(r.MqlRuntime, "openclaw.skill", r.ConfigPath.Data, defaultOpenClawConfigDir,
		filepath.Join(defaultOpenClawConfigDir, "skills"), filepath.Join(r.ConfigPath.Data, "skills"))
}

func (r *mqlOpenclawSkill) id() (string, error)     { return "openclaw.skill/" + r.Source.Data, nil }
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
	return agentSkills(r.MqlRuntime, "snowflake.cortex.skill", r.ConfigPath.Data, defaultCortexConfigDir,
		filepath.Join(defaultCortexConfigDir, "skills"), filepath.Join(r.ConfigPath.Data, "skills"))
}

func (r *mqlSnowflakeCortexSkill) id() (string, error) {
	return "snowflake.cortex.skill/" + r.Source.Data, nil
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
	return agentSkills(r.MqlRuntime, "junie.skill", r.ConfigPath.Data, defaultJunieConfigDir,
		filepath.Join(defaultJunieConfigDir, "skills"), filepath.Join(r.ConfigPath.Data, "skills"))
}

func (r *mqlJunieSkill) id() (string, error)     { return "junie.skill/" + r.Source.Data, nil }
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
	return agentSkills(r.MqlRuntime, "augment.skill", r.ConfigPath.Data, defaultAugmentConfigDir,
		filepath.Join(defaultAugmentConfigDir, "skills"), filepath.Join(r.ConfigPath.Data, "skills"))
}

func (r *mqlAugmentSkill) id() (string, error)     { return "augment.skill/" + r.Source.Data, nil }
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
	// Warp reads skills from the shared ~/.agents/skills directory.
	return agentSkills(r.MqlRuntime, "warp.skill", r.ConfigPath.Data, defaultWarpConfigDir,
		filepath.Join(".agents", "skills"),
		filepath.Join(filepath.Dir(r.ConfigPath.Data), ".agents", "skills"))
}

func (r *mqlWarpSkill) id() (string, error)     { return "warp.skill/" + r.Source.Data, nil }
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
	return agentSkills(r.MqlRuntime, "kilocode.skill", r.ConfigPath.Data, defaultKiloCodeConfigDir,
		filepath.Join(defaultKiloCodeConfigDir, "skills"), filepath.Join(r.ConfigPath.Data, "skills"))
}

func (r *mqlKilocodeSkill) id() (string, error)     { return "kilocode.skill/" + r.Source.Data, nil }
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
	return agentSkills(r.MqlRuntime, "openhands.skill", r.ConfigPath.Data, defaultOpenHandsConfigDir,
		filepath.Join(defaultOpenHandsConfigDir, "skills"), filepath.Join(r.ConfigPath.Data, "skills"))
}

func (r *mqlOpenhandsSkill) id() (string, error)     { return "openhands.skill/" + r.Source.Data, nil }
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
	return agentSkills(r.MqlRuntime, "qwen.code.skill", r.ConfigPath.Data, defaultQwenCodeConfigDir,
		filepath.Join(defaultQwenCodeConfigDir, "skills"), filepath.Join(r.ConfigPath.Data, "skills"))
}

func (r *mqlQwenCodeSkill) id() (string, error)     { return "qwen.code.skill/" + r.Source.Data, nil }
func (r *mqlQwenCodeSkill) sha256() (string, error) { return contentSHA256(r.Content.Data), nil }
