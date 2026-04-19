// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	packageurl "github.com/package-url/packageurl-go"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

const (
	NixPkgFormat = "nix"
)

type NixPkgManager struct {
	conn     shared.Connection
	platform *inventory.Platform
}

func (npm *NixPkgManager) Name() string {
	return "Nix Package Manager"
}

func (npm *NixPkgManager) Format() string {
	return NixPkgFormat
}

func (npm *NixPkgManager) List() ([]Package, error) {
	// Primary: nix-env CLI (available on NixOS and standalone Nix)
	if npm.conn.Capabilities().Has(shared.Capability_RunCommand) {
		pkgs, err := npm.listFromCLI()
		if err == nil {
			return pkgs, nil
		}
		log.Debug().Err(err).Msg("mql[nix]> could not enumerate via CLI, falling back to filesystem")
	}

	// Fallback: parse /nix/store/ directory names
	return npm.listFromFS()
}

func (npm *NixPkgManager) listFromCLI() ([]Package, error) {
	cmd, err := npm.conn.RunCommand("nix-env --query --installed --json")
	if err != nil {
		return nil, err
	}
	if cmd.ExitStatus != 0 {
		return nil, fmt.Errorf("nix-env query failed with exit code %d", cmd.ExitStatus)
	}

	return ParseNixJSON(cmd.Stdout)
}

// nixJSONOutput represents the JSON output of `nix-env --query --installed --json`.
// The top-level object maps attribute names to package info objects.
type nixJSONPackage struct {
	Name    string `json:"pname"`
	Version string `json:"version"`
	// Full derivation name (e.g., "curl-8.7.1")
	System string `json:"system"`
}

// ParseNixJSON parses the JSON output of `nix-env --query --installed --json`.
func ParseNixJSON(r io.Reader) ([]Package, error) {
	var packages map[string]nixJSONPackage
	if err := json.NewDecoder(r).Decode(&packages); err != nil {
		return nil, fmt.Errorf("could not parse nix-env JSON: %w", err)
	}

	pkgs := make([]Package, 0, len(packages))
	for _, np := range packages {
		name := np.Name
		if name == "" {
			continue
		}

		pkgs = append(pkgs, Package{
			Name:    name,
			Version: np.Version,
			Format:  NixPkgFormat,
			PUrl:    newNixPurl(name, np.Version),
		})
	}

	return pkgs, nil
}

func (npm *NixPkgManager) listFromFS() ([]Package, error) {
	afs := &afero.Afero{Fs: npm.conn.FileSystem()}
	return ParseNixStore(afs, "/nix/store")
}

// nixStoreRegex matches Nix store path directory names.
// Format: <32-char hash>-<name>-<version>
// Example: "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee-curl-8.7.1"
var nixStoreRegex = regexp.MustCompile(`^[a-z0-9]{32}-(.+)$`)

// ParseNixStore enumerates packages from the /nix/store directory.
// Each directory name is <hash>-<name>-<version>. We extract name and
// version by splitting on the last hyphen before a version-like segment.
func ParseNixStore(afs *afero.Afero, storePath string) ([]Package, error) {
	entries, err := afs.ReadDir(storePath)
	if err != nil {
		return nil, fmt.Errorf("could not read nix store at %s: %w", storePath, err)
	}

	// Use a map to deduplicate (same package may appear multiple times
	// in the store with different hashes)
	seen := map[string]Package{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		m := nixStoreRegex.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}

		nameVersion := m[1]
		name, version := splitNixNameVersion(nameVersion)
		if name == "" {
			continue
		}

		// Skip .drv (derivation) directories and internal entries
		if strings.HasSuffix(name, ".drv") {
			continue
		}

		key := name + "@" + version
		if _, ok := seen[key]; !ok {
			seen[key] = Package{
				Name:    name,
				Version: version,
				Format:  NixPkgFormat,
				PUrl:    newNixPurl(name, version),
			}
		}
	}

	pkgs := make([]Package, 0, len(seen))
	for _, p := range seen {
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

// splitNixNameVersion splits a Nix store name-version string.
// Example: "curl-8.7.1" → ("curl", "8.7.1")
// Example: "python3.11-requests-2.31.0" → ("python3.11-requests", "2.31.0")
// The version starts at the last hyphen followed by a digit.
func splitNixNameVersion(s string) (string, string) {
	for i := len(s) - 1; i > 0; i-- {
		if s[i] == '-' && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}

// newNixPurl creates a PURL for a Nix package.
// Format: pkg:nix/name@version
func newNixPurl(name, version string) string {
	if name == "" {
		return ""
	}
	return packageurl.NewPackageURL(
		packageurl.TypeNix,
		"",
		name,
		version,
		nil,
		"",
	).String()
}

func (npm *NixPkgManager) Available() (map[string]PackageUpdate, error) {
	return map[string]PackageUpdate{}, nil
}

func (npm *NixPkgManager) Files(name string, version string, arch string) ([]FileRecord, error) {
	return nil, nil
}
