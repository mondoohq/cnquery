// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/core/resources/versions/semver"
)

// toolPackageSpec describes how a tool resource resolves its backing package
// (the `package` accessor added to each AI coding-tool resource). See
// docs/adr/028-tool-install-indicator.md for the design.
type toolPackageSpec struct {
	// packageName is the canonical name used for an abstract package. Because
	// an abstract package has an unknown source (origin "unknown"), this is a
	// clean tool identifier (the resource name with dots turned into hyphens),
	// not a guess at an upstream package name.
	packageName string
	// managerCandidates are names tried against the system package managers via
	// the name-based `package(name)` lookup. Curated conservatively: only
	// distinctive, low-false-positive names are listed; ambiguous or
	// common-word names are left empty (the tool then always resolves to an
	// abstract package). Exact-name matching is a first cut — the robust
	// replacement (which package owns the tool's binary) is an ADR follow-up.
	managerCandidates []string
	// vendor is the producing vendor when confidently known; empty otherwise.
	vendor string
	// inferVersion optionally determines the tool version for the abstract
	// package. Best-effort: it returns "" (unknown) rather than an error when
	// the source is missing or unreadable.
	inferVersion func(runtime *plugin.Runtime, configPath string) (string, error)
}

// toolPackageSpecs is keyed by MQL resource name.
var toolPackageSpecs = map[string]toolPackageSpec{
	"claude.code":    {packageName: "claude-code", vendor: "Anthropic", inferVersion: inferClaudeVersion},
	"openai.codex":   {packageName: "openai-codex", vendor: "OpenAI", inferVersion: inferCodexVersion},
	"cursor":         {packageName: "cursor", managerCandidates: []string{"cursor"}, vendor: "Anysphere"},
	"github.copilot": {packageName: "github-copilot", vendor: "GitHub"},
	// Distinctive names, not bare "goose"/"gemini": bare "goose" collides with
	// the widely-packaged pressly/goose DB-migration tool (Homebrew core), and
	// the Gemini CLI ships as npm `@google/gemini-cli`. These match only the
	// real tool where it exists as an OS package; the binary-ownership
	// follow-up will attribute the common npm/script installs.
	"goose":            {packageName: "goose", managerCandidates: []string{"block-goose-cli"}, vendor: "Block"},
	"gemini":           {packageName: "gemini", managerCandidates: []string{"gemini-cli"}, vendor: "Google"},
	"windsurf":         {packageName: "windsurf", managerCandidates: []string{"windsurf"}},
	"zed":              {packageName: "zed", managerCandidates: []string{"zed"}, vendor: "Zed Industries"},
	"roo":              {packageName: "roo"},
	"cline":            {packageName: "cline"},
	"kiro":             {packageName: "kiro", managerCandidates: []string{"kiro"}},
	"continuedev":      {packageName: "continuedev"},
	"trae":             {packageName: "trae", managerCandidates: []string{"trae"}},
	"opencode":         {packageName: "opencode", managerCandidates: []string{"opencode"}},
	"pi":               {packageName: "pi"},
	"mistral.vibe":     {packageName: "mistral-vibe", vendor: "Mistral AI"},
	"antigravity":      {packageName: "antigravity", managerCandidates: []string{"antigravity"}, vendor: "Google"},
	"ibm.bob":          {packageName: "ibm-bob", vendor: "IBM"},
	"openclaw":         {packageName: "openclaw"},
	"snowflake.cortex": {packageName: "snowflake-cortex", vendor: "Snowflake"},
	"junie":            {packageName: "junie", managerCandidates: []string{"junie"}, vendor: "JetBrains"},
	"augment":          {packageName: "augment"},
	"warp":             {packageName: "warp", vendor: "Warp"},
	"kilocode":         {packageName: "kilocode", managerCandidates: []string{"kilocode"}},
	"openhands":        {packageName: "openhands", managerCandidates: []string{"openhands"}},
	"qwen.code":        {packageName: "qwen-code", vendor: "Alibaba"},
}

// resolveToolPackage returns the real system-package-manager entry that
// installed the tool when it can be attributed to one, otherwise an abstract
// package (origin "unknown", empty format, never inserted into `packages`).
func resolveToolPackage(runtime *plugin.Runtime, configPath string, spec toolPackageSpec) (*mqlPackage, error) {
	// (a) Try to attribute the tool to a real package via the name-based init
	// lookup. NewResource runs initPackage; a managed hit is the same cached
	// instance that appears in the `packages` collection.
	for _, name := range spec.managerCandidates {
		raw, err := NewResource(runtime, "package", map[string]*llx.RawData{
			"name": llx.StringData(name),
		})
		if err != nil {
			return nil, err
		}
		if pkg, ok := raw.(*mqlPackage); ok && pkg.Installed.Data {
			return pkg, nil
		}
	}

	// (b) Not attributable — detect presence and, where possible, a version,
	// then synthesize an abstract package.
	installed := configDirPresent(runtime, configPath)
	version := ""
	if installed && spec.inferVersion != nil {
		v, err := spec.inferVersion(runtime, configPath)
		if err != nil {
			return nil, err
		}
		version = v
	}

	raw, err := CreateResource(runtime, "package", map[string]*llx.RawData{
		// synthetic id; distinct from the real "format://name/version/arch"
		// scheme and hidden (no `id` field on package).
		"__id":      llx.StringData("tool://" + spec.packageName),
		"name":      llx.StringData(spec.packageName),
		"installed": llx.BoolData(installed),
		"origin":    llx.StringData("unknown"), // abstract discriminator (seeds origin())
		"format":    llx.StringData(""),        // no package-manager format
	})
	if err != nil {
		return nil, err
	}
	pkg := raw.(*mqlPackage)

	setStrOrNull(&pkg.Version, version)
	setStrOrNull(&pkg.Vendor, spec.vendor)

	// Mirror the initPackage not-found stub: give every remaining field a
	// resolved state (null unless meaningful) so any field queries cleanly.
	pkg.Outdated = plugin.TValue[bool]{Data: false, State: plugin.StateIsSet}
	pkg.Epoch.State = plugin.StateIsSet | plugin.StateIsNull
	pkg.Available.State = plugin.StateIsSet | plugin.StateIsNull
	pkg.Description.State = plugin.StateIsSet | plugin.StateIsNull
	pkg.Purl.State = plugin.StateIsSet | plugin.StateIsNull
	pkg.Cpes.State = plugin.StateIsSet | plugin.StateIsNull
	pkg.Arch.State = plugin.StateIsSet | plugin.StateIsNull
	pkg.Status.State = plugin.StateIsSet | plugin.StateIsNull
	pkg.Files.State = plugin.StateIsSet | plugin.StateIsNull
	pkg.License.State = plugin.StateIsSet | plugin.StateIsNull
	pkg.InstallDate.State = plugin.StateIsSet | plugin.StateIsNull

	return pkg, nil
}

// configDirPresent is the v1 presence signal: the tool's configPath directory
// exists on the target. Best-effort; a stronger binary-ownership signal is an
// ADR follow-up.
func configDirPresent(runtime *plugin.Runtime, configPath string) bool {
	if configPath == "" {
		return false
	}
	info, err := connectionAfs(runtime).Stat(configPath)
	return err == nil && info.IsDir()
}

// setStrOrNull sets a string field to val, or to a resolved-null state when val
// is empty, so the field always reads cleanly in MQL.
func setStrOrNull(t *plugin.TValue[string], val string) {
	if val != "" {
		*t = plugin.TValue[string]{Data: val, State: plugin.StateIsSet}
	} else {
		*t = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	}
}

// inferCodexVersion reads the OpenAI Codex version.json (same source as
// openai.codex.version()). Best-effort: unknown on any read/parse failure.
func inferCodexVersion(runtime *plugin.Runtime, configPath string) (string, error) {
	var ver struct {
		LatestVersion string `json:"latest_version"`
	}
	if err := readJSONFileAfero(connectionAfs(runtime), configPath, "version.json", &ver); err != nil {
		return "", nil
	}
	return ver.LatestVersion, nil
}

// inferClaudeVersion runs `claude --version` through the command resource
// (Claude Code writes no version file, so we probe the binary). The output is
// e.g. "2.1.191 (Claude Code)"; we take the leading token and keep it only if
// MQL's semver parser recognizes it. Best-effort: unknown when the binary is
// absent or the output carries no recognizable version.
func inferClaudeVersion(runtime *plugin.Runtime, configPath string) (string, error) {
	o, err := CreateResource(runtime, "command", map[string]*llx.RawData{
		"command": llx.StringData("claude --version"),
	})
	if err != nil {
		return "", nil
	}
	cmd := o.(*mqlCommand)
	if cmd.GetExitcode().Data != 0 {
		return "", nil
	}
	fields := strings.Fields(cmd.GetStdout().Data)
	if len(fields) == 0 {
		return "", nil
	}
	version := fields[0]
	// Validate with MQL's semver parser instead of a bespoke regex. The parser
	// exposes only Compare, so we parse by self-compare: a valid version
	// compares against itself without error; an invalid one returns an error.
	if _, err := (semver.Parser{}).Compare(version, version); err != nil {
		return "", nil
	}
	return version, nil
}

// compute_package accessors — one per tool resource. Each delegates to the
// shared resolver with the tool's spec. The method name is compute_package (not
// package) because `package` is a Go keyword; the generator's fieldCall prefixes
// reserved words with "compute_".

func (r *mqlClaudeCode) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["claude.code"])
}

func (r *mqlOpenaiCodex) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["openai.codex"])
}

func (r *mqlCursor) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["cursor"])
}

func (r *mqlGithubCopilot) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["github.copilot"])
}

func (r *mqlGoose) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["goose"])
}

func (r *mqlGemini) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["gemini"])
}

func (r *mqlWindsurf) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["windsurf"])
}

func (r *mqlZed) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["zed"])
}

func (r *mqlRoo) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["roo"])
}

func (r *mqlCline) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["cline"])
}

func (r *mqlKiro) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["kiro"])
}

func (r *mqlContinuedev) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["continuedev"])
}

func (r *mqlTrae) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["trae"])
}

func (r *mqlOpencode) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["opencode"])
}

func (r *mqlPi) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["pi"])
}

func (r *mqlMistralVibe) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["mistral.vibe"])
}

func (r *mqlAntigravity) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["antigravity"])
}

func (r *mqlIbmBob) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["ibm.bob"])
}

func (r *mqlOpenclaw) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["openclaw"])
}

func (r *mqlSnowflakeCortex) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["snowflake.cortex"])
}

func (r *mqlJunie) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["junie"])
}

func (r *mqlAugment) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["augment"])
}

func (r *mqlWarp) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["warp"])
}

func (r *mqlKilocode) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["kilocode"])
}

func (r *mqlOpenhands) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["openhands"])
}

func (r *mqlQwenCode) compute_package() (*mqlPackage, error) {
	return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["qwen.code"])
}
