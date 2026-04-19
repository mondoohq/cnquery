// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"unicode"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/purl"
)

const (
	GentooPkgFormat = "gentoo"
)

// ParseGentooPackages parses the output of
// `qlist -Iv --format '%{CATEGORY}/%{PN}:%{PVR}'`.
// Format: CATEGORY/NAME:VERSION (e.g., "net-misc/curl:8.4.0")
func ParseGentooPackages(pf *inventory.Platform, r io.Reader) ([]Package, error) {
	pkgs := []Package{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Split by colon delimiter
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		categoryName := strings.TrimSpace(parts[0])
		version := strings.TrimSpace(parts[1])

		pkgs = append(pkgs, Package{
			Name:    categoryName,
			Version: version,
			Format:  GentooPkgFormat,
			PUrl:    newEbuildPurl(pf, categoryName, version),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return pkgs, nil
}

// newEbuildPurl creates a PURL for a Gentoo ebuild package.
// Format: pkg:ebuild/CATEGORY/NAME@VERSION
func newEbuildPurl(pf *inventory.Platform, categoryName, version string) string {
	category, name := splitCategoryName(categoryName)
	return purl.NewPackageURL(pf, purl.TypeEbuild, name, version,
		purl.WithNamespace(category),
	).String()
}

// splitCategoryName splits "category/name" into its parts.
func splitCategoryName(categoryName string) (string, string) {
	idx := strings.LastIndex(categoryName, "/")
	if idx < 0 {
		return "", categoryName
	}
	return categoryName[:idx], categoryName[idx+1:]
}

// Gentoo
type GentooPkgManager struct {
	conn     shared.Connection
	platform *inventory.Platform
}

func (f *GentooPkgManager) Name() string {
	return "Gentoo Package Manager"
}

func (f *GentooPkgManager) Format() string {
	return GentooPkgFormat
}

func (f *GentooPkgManager) List() ([]Package, error) {
	// Primary: qlist CLI
	if f.conn.Capabilities().Has(shared.Capability_RunCommand) {
		cmd, err := f.conn.RunCommand("qlist -Iv --format '%{CATEGORY}/%{PN}:%{PVR}'")
		if err != nil {
			log.Debug().Err(err).Msg("mql[gentoo]> could not run qlist, falling back to filesystem")
		} else if cmd.ExitStatus != 0 {
			log.Debug().Int("exitStatus", cmd.ExitStatus).Msg("mql[gentoo]> qlist returned non-zero, falling back to filesystem")
		} else {
			return ParseGentooPackages(f.platform, cmd.Stdout)
		}
	}

	// Fallback: parse /var/db/pkg/ directory structure
	return f.listFromFS()
}

func (f *GentooPkgManager) listFromFS() ([]Package, error) {
	fs := f.conn.FileSystem()
	if fs == nil {
		return nil, errors.New("gentoo package manager requires either command execution or filesystem access")
	}
	afs := &afero.Afero{Fs: fs}
	return ParsePortageDB(f.platform, afs, "/var/db/pkg")
}

// ParsePortageDB parses the Portage installed package database directory.
// Structure: /var/db/pkg/CATEGORY/NAME-VERSION/
func ParsePortageDB(pf *inventory.Platform, afs *afero.Afero, dbPath string) ([]Package, error) {
	categories, err := afs.ReadDir(dbPath)
	if err != nil {
		return nil, fmt.Errorf("could not read portage database at %s: %w", dbPath, err)
	}

	var pkgs []Package
	for _, cat := range categories {
		if !cat.IsDir() {
			continue
		}
		category := cat.Name()

		// path.Join (not filepath.Join) is intentional — these are always
		// Linux filesystem paths, even when mql runs on a different OS.
		catPath := path.Join(dbPath, category)
		entries, err := afs.ReadDir(catPath)
		if err != nil {
			log.Debug().Err(err).Str("path", catPath).Msg("mql[gentoo]> could not read category")
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			name, version := splitPortageDirName(entry.Name())
			if name == "" || version == "" {
				continue
			}

			fullName := category + "/" + name

			// Try to read description
			descPath := path.Join(catPath, entry.Name(), "DESCRIPTION")
			description := readFileContent(afs, descPath)

			pkgs = append(pkgs, Package{
				Name:        fullName,
				Version:     version,
				Description: description,
				Format:      GentooPkgFormat,
				PUrl:        newEbuildPurl(pf, fullName, version),
			})
		}
	}

	return pkgs, nil
}

// splitPortageDirName splits a directory name like "curl-8.4.0" or
// "dhcpcd-10.0.5-r1" into (name, version). The version starts at the last
// hyphen that is followed by a digit.
func splitPortageDirName(dirName string) (string, string) {
	// Walk backwards to find the last hyphen followed by a digit
	for i := len(dirName) - 1; i > 0; i-- {
		if dirName[i] == '-' && i+1 < len(dirName) && unicode.IsDigit(rune(dirName[i+1])) {
			return dirName[:i], dirName[i+1:]
		}
	}
	return dirName, ""
}

// readFileContent reads the first line of a file, returning empty string on error.
func readFileContent(afs *afero.Afero, path string) string {
	f, err := afs.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

func (f *GentooPkgManager) Available() (map[string]PackageUpdate, error) {
	return map[string]PackageUpdate{}, nil
}

func (f *GentooPkgManager) Files(name string, version string, arch string) ([]FileRecord, error) {
	// not yet implemented
	return nil, nil
}
