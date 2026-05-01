// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"bufio"
	"fmt"
	"io"
	"path"
	"strings"

	packageurl "github.com/package-url/packageurl-go"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

const (
	FlatpakPkgFormat = "flatpak"
)

type FlatpakPkgManager struct {
	conn     shared.Connection
	platform *inventory.Platform
}

func (fpm *FlatpakPkgManager) Name() string {
	return "Flatpak Package Manager"
}

func (fpm *FlatpakPkgManager) Format() string {
	return FlatpakPkgFormat
}

func (fpm *FlatpakPkgManager) List() ([]Package, error) {
	// Primary: flatpak CLI
	if fpm.conn.Capabilities().Has(shared.Capability_RunCommand) {
		pkgs, err := fpm.listFromCLI()
		if err == nil {
			return pkgs, nil
		}
		log.Debug().Err(err).Msg("mql[flatpak]> could not enumerate via CLI, falling back to filesystem")
	}

	// Fallback: parse /var/lib/flatpak/app/ directory
	return fpm.listFromFS()
}

func (fpm *FlatpakPkgManager) listFromCLI() ([]Package, error) {
	// --app excludes runtimes, --columns controls output fields
	cmd, err := fpm.conn.RunCommand("flatpak list --app --columns=application,version,origin,arch")
	if err != nil {
		return nil, err
	}
	if cmd.ExitStatus != 0 {
		return nil, fmt.Errorf("flatpak list failed with exit code %d", cmd.ExitStatus)
	}

	return ParseFlatpakList(cmd.Stdout)
}

// ParseFlatpakList parses the output of
// `flatpak list --app --columns=application,version,origin,arch`.
// Each line is tab-delimited: APPLICATION\tVERSION\tORIGIN\tARCH
func ParseFlatpakList(r io.Reader) ([]Package, error) {
	var pkgs []Package
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}

		appID := strings.TrimSpace(fields[0])
		version := strings.TrimSpace(fields[1])
		origin := ""
		arch := ""
		if len(fields) >= 3 {
			origin = strings.TrimSpace(fields[2])
		}
		if len(fields) >= 4 {
			arch = strings.TrimSpace(fields[3])
		}

		if appID == "" {
			continue
		}

		pkgs = append(pkgs, Package{
			Name:    appID,
			Version: version,
			Arch:    arch,
			Format:  FlatpakPkgFormat,
			Origin:  origin,
			PUrl:    newFlatpakPurl(appID, version, origin),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return pkgs, nil
}

func (fpm *FlatpakPkgManager) listFromFS() ([]Package, error) {
	afs := &afero.Afero{Fs: fpm.conn.FileSystem()}

	var pkgs []Package

	// System-wide installations
	sysPkgs, err := parseFlatpakDir(afs, "/var/lib/flatpak/app")
	if err == nil {
		pkgs = append(pkgs, sysPkgs...)
	}

	// Per-user installations (common home directories)
	homeDirs := []string{"/home", "/root"}
	for _, homeBase := range homeDirs {
		entries, err := afs.ReadDir(homeBase)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			userAppDir := path.Join(homeBase, entry.Name(), ".local/share/flatpak/app")
			userPkgs, err := parseFlatpakDir(afs, userAppDir)
			if err == nil {
				pkgs = append(pkgs, userPkgs...)
			}
		}
	}

	return pkgs, nil
}

// parseFlatpakDir enumerates Flatpak apps from the filesystem.
// Structure: /var/lib/flatpak/app/APP_ID/ARCH/BRANCH/active/metadata
func parseFlatpakDir(afs *afero.Afero, appDir string) ([]Package, error) {
	apps, err := afs.ReadDir(appDir)
	if err != nil {
		return nil, fmt.Errorf("could not read flatpak app directory at %s: %w", appDir, err)
	}

	var pkgs []Package
	for _, app := range apps {
		if !app.IsDir() {
			continue
		}
		appID := app.Name()

		// path.Join (not filepath.Join) is intentional — these are always
		// Linux filesystem paths, even when mql runs on a different OS.
		archDir := path.Join(appDir, appID)
		arches, err := afs.ReadDir(archDir)
		if err != nil {
			continue
		}

		for _, archEntry := range arches {
			if !archEntry.IsDir() {
				continue
			}
			arch := archEntry.Name()

			branchDir := path.Join(archDir, arch)
			branches, err := afs.ReadDir(branchDir)
			if err != nil {
				continue
			}

			for _, branchEntry := range branches {
				if !branchEntry.IsDir() {
					continue
				}

				// Read metadata from active deployment
				metadataPath := path.Join(branchDir, branchEntry.Name(), "active", "metadata")
				origin, version := parseFlatpakMetadata(afs, metadataPath)

				pkgs = append(pkgs, Package{
					Name:    appID,
					Version: version,
					Arch:    arch,
					Format:  FlatpakPkgFormat,
					Origin:  origin,
					PUrl:    newFlatpakPurl(appID, version, origin),
				})
			}
		}
	}

	return pkgs, nil
}

// parseFlatpakMetadata reads a Flatpak metadata file (INI-like format)
// and extracts the origin and version. These are deploy-level keys that
// Flatpak appends to the metadata file. Lines inside [Extension ...]
// sections are skipped to avoid false matches.
func parseFlatpakMetadata(afs *afero.Afero, metadataPath string) (string, string) {
	f, err := afs.Open(metadataPath)
	if err != nil {
		return "", ""
	}
	defer f.Close()

	var origin, version string
	inExtension := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Track INI sections — skip [Extension ...] blocks
		if len(line) > 0 && line[0] == '[' {
			inExtension = strings.HasPrefix(line, "[Extension ")
			continue
		}

		if inExtension {
			continue
		}

		if strings.HasPrefix(line, "origin=") {
			origin = strings.TrimPrefix(line, "origin=")
		}
		if strings.HasPrefix(line, "version=") {
			version = strings.TrimPrefix(line, "version=")
		}

		if origin != "" && version != "" {
			break
		}
	}
	return origin, version
}

// newFlatpakPurl creates a PURL for a Flatpak application.
// Format: pkg:flatpak/origin/app-id@version
func newFlatpakPurl(appID, version, origin string) string {
	if appID == "" {
		return ""
	}
	return packageurl.NewPackageURL(
		"flatpak",
		origin,
		appID,
		version,
		nil,
		"",
	).String()
}

func (fpm *FlatpakPkgManager) Available() (map[string]PackageUpdate, error) {
	return map[string]PackageUpdate{}, nil
}

func (fpm *FlatpakPkgManager) Files(name string, version string, arch string) ([]FileRecord, error) {
	return nil, nil
}
