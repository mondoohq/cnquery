// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"path/filepath"
	"sort"
	"strings"
)

// Dialect names the tool a configuration is written for.
//
// Terraform and OpenTofu share the HCL language and the `terraform show -json`
// representation, so they are read by the same resources. What differs is which
// files on disk make up a configuration: OpenTofu additionally reads a
// .tofu-flavored set, and prefers it over the .tf equivalent.
type Dialect string

const (
	DialectTerraform Dialect = "terraform"
	DialectOpenTofu  Dialect = "opentofu"
)

// OptionDialect is the connection option carrying an explicitly chosen dialect.
// It is set from the --iac-tool flag, or from the connector name when the user
// invoked the provider as `opentofu` / `tofu`.
const OptionDialect = "iac-tool"

// ParseDialect maps a user-supplied name onto a dialect, defaulting to
// Terraform for anything unrecognized.
func ParseDialect(s string) Dialect {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "opentofu", "tofu":
		return DialectOpenTofu
	default:
		return DialectTerraform
	}
}

// fileClass groups the configuration file kinds that OpenTofu defines a
// .tofu-flavored override for. Precedence applies within one class: main.tofu
// replaces main.tf, but never main.tf.json, which belongs to a different class.
type fileClass int

const (
	classConfig     fileClass = iota // .tf           / .tofu
	classConfigJSON                  // .tf.json      / .tofu.json
	classVars                        // .tfvars       / .tofuvars
	classVarsJSON                    // .tfvars.json  / .tofuvars.json
)

// configExtensions maps a file suffix onto its class and dialect.
//
// Order matters. The longest suffix has to be tested first, so that
// prod.tofuvars.json is recognized as a variables file rather than falling
// through to a shorter, differently-classed suffix.
var configExtensions = []struct {
	suffix  string
	class   fileClass
	dialect Dialect
}{
	{".tofuvars.json", classVarsJSON, DialectOpenTofu},
	{".tofuvars", classVars, DialectOpenTofu},
	{".tofu.json", classConfigJSON, DialectOpenTofu},
	{".tofu", classConfig, DialectOpenTofu},
	{".tfvars.json", classVarsJSON, DialectTerraform},
	{".tfvars", classVars, DialectTerraform},
	{".tf.json", classConfigJSON, DialectTerraform},
	{".tf", classConfig, DialectTerraform},
}

// configFile is a path recognized as part of a Terraform or OpenTofu
// configuration.
type configFile struct {
	path    string
	class   fileClass
	dialect Dialect

	// dir and stem identify the slot this file occupies. Together with class
	// they are what a .tofu file overrides: same directory, same name, same
	// kind of file.
	dir  string
	stem string
}

// classifyConfigFile recognizes a configuration file by its extension. A file
// whose whole name is the extension (".tf") is not a configuration file, since
// there is no name left to match a .tofu override against.
func classifyConfigFile(path string) (configFile, bool) {
	name := filepath.Base(path)
	for _, ext := range configExtensions {
		if len(name) > len(ext.suffix) && strings.HasSuffix(name, ext.suffix) {
			return configFile{
				path:    path,
				class:   ext.class,
				dialect: ext.dialect,
				dir:     filepath.Dir(path),
				stem:    strings.TrimSuffix(name, ext.suffix),
			}, true
		}
	}
	return configFile{}, false
}

// resolvedConfig is the outcome of applying OpenTofu's precedence rules to a
// set of candidate paths.
type resolvedConfig struct {
	// Configs are the configuration files to parse, sorted.
	Configs []string
	// Vars are the variable definition files to read, sorted.
	Vars []string
	// Dialect is DialectOpenTofu when any .tofu-flavored file is part of the
	// configuration, whether or not it overrode anything.
	Dialect Dialect
	// Overridden lists the .tf files that a .tofu file replaced, sorted. They
	// are reported so a scan can explain why a file on disk was not read.
	Overridden []string
}

// slot identifies the position a .tofu file competes for.
type slot struct {
	dir   string
	stem  string
	class fileClass
}

// resolveConfigFiles applies OpenTofu's file precedence to the candidates.
//
// OpenTofu loads foo.tofu *instead of* foo.tf when both are present, and
// likewise for the .json, .tfvars and .tfvars.json variants. The rule is per
// directory and per file name: a.tofu overrides a.tf and leaves b.tf alone.
// Reading the .tf file in that situation would describe configuration that
// OpenTofu never applies.
//
// Paths that are not configuration files are ignored.
func resolveConfigFiles(paths []string) resolvedConfig {
	return resolveConfigFilesAs(paths, "")
}

// resolveConfigFilesAs is resolveConfigFiles with an explicitly chosen dialect.
//
// An empty dialect auto-detects: any .tofu-flavored file present means the
// configuration is OpenTofu's. Forcing DialectTerraform reads the directory the
// way Terraform itself does, ignoring .tofu files entirely rather than letting
// them override anything, which is what a repository shared between the two
// tools looks like to Terraform.
func resolveConfigFilesAs(paths []string, forced Dialect) resolvedConfig {
	chosen := make(map[slot]configFile, len(paths))
	var overridden []string

	for _, p := range paths {
		f, ok := classifyConfigFile(p)
		if !ok {
			continue
		}
		if forced == DialectTerraform && f.dialect == DialectOpenTofu {
			continue
		}

		key := slot{dir: f.dir, stem: f.stem, class: f.class}
		prev, exists := chosen[key]
		if !exists {
			chosen[key] = f
			continue
		}

		// Same slot, so one of the two is the OpenTofu override. Two files of
		// the same dialect cannot collide here: the extension is part of the
		// name, so they would occupy different slots.
		if f.dialect == DialectOpenTofu && prev.dialect == DialectTerraform {
			chosen[key] = f
			overridden = append(overridden, prev.path)
		} else if f.dialect == DialectTerraform && prev.dialect == DialectOpenTofu {
			overridden = append(overridden, f.path)
		}
	}

	res := resolvedConfig{Dialect: DialectTerraform}
	for _, f := range chosen {
		if f.dialect == DialectOpenTofu {
			res.Dialect = DialectOpenTofu
		}
		switch f.class {
		case classConfig, classConfigJSON:
			res.Configs = append(res.Configs, f.path)
		case classVars, classVarsJSON:
			res.Vars = append(res.Vars, f.path)
		}
	}

	if forced != "" {
		res.Dialect = forced
	}

	// map iteration is unordered; sort so a scan is reproducible
	sort.Strings(res.Configs)
	sort.Strings(res.Vars)
	sort.Strings(overridden)
	res.Overridden = overridden

	return res
}
