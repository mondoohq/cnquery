// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"bufio"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/purl"
)

const (
	PacmanPkgFormat = "pacman"
	// PacmanLocalDB is the root of pacman's local package database. Each
	// installed package has a directory named "<name>-<version>" that holds a
	// `files` manifest listing everything the package put on disk. AUR
	// packages installed through helpers like yay or paru register here too,
	// so they're covered without any extra handling.
	PacmanLocalDB = "/var/lib/pacman/local"
)

var PACMAN_REGEX = regexp.MustCompile(`^([\w-]*)\s([\w\d-+.:]+)$`)

func ParsePacmanPackages(pf *inventory.Platform, input io.Reader) []Package {
	pkgs := []Package{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		m := PACMAN_REGEX.FindStringSubmatch(line)
		if m != nil {
			name := m[1]
			version := m[2]
			pkgs = append(pkgs, Package{
				Name:           name,
				Version:        version,
				Format:         PacmanPkgFormat,
				FilesAvailable: PkgFilesAsync,
				PUrl:           purl.NewPackageURL(pf, purl.TypeAlpm, name, version).String(),
			})
		}
	}
	return pkgs
}

// Arch, Manjaro
type PacmanPkgManager struct {
	conn     shared.Connection
	platform *inventory.Platform
}

func (ppm *PacmanPkgManager) Name() string {
	return "Pacman Package Manager"
}

func (ppm *PacmanPkgManager) Format() string {
	return PacmanPkgFormat
}

func (ppm *PacmanPkgManager) List() ([]Package, error) {
	// Primary: pacman -Q CLI
	if ppm.conn.Capabilities().Has(shared.Capability_RunCommand) {
		cmd, err := ppm.conn.RunCommand("pacman -Q")
		if err == nil && cmd.ExitStatus == 0 {
			return ParsePacmanPackages(ppm.platform, cmd.Stdout), nil
		}
		log.Debug().Err(err).Msg("mql[pacman]> could not run pacman -Q, falling back to filesystem")
	}

	// Fallback: parse /var/lib/pacman/local/*/desc files
	return ppm.listFromFS()
}

func (ppm *PacmanPkgManager) listFromFS() ([]Package, error) {
	afs := &afero.Afero{Fs: ppm.conn.FileSystem()}
	return ParsePacmanDB(ppm.platform, afs, PacmanLocalDB)
}

// ParsePacmanDB parses the pacman local database directory structure.
// Each subdirectory contains a `desc` file with package metadata.
func ParsePacmanDB(pf *inventory.Platform, afs *afero.Afero, dbPath string) ([]Package, error) {
	entries, err := afs.ReadDir(dbPath)
	if err != nil {
		return nil, fmt.Errorf("could not read pacman database at %s: %w", dbPath, err)
	}

	var pkgs []Package
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// path.Join (not filepath.Join) is intentional — these are always
		// Linux filesystem paths, even when mql runs on a different OS.
		descPath := path.Join(dbPath, entry.Name(), "desc")
		pkg, err := parsePacmanDesc(pf, afs, descPath)
		if err != nil {
			log.Debug().Err(err).Str("path", descPath).Msg("mql[pacman]> could not parse desc")
			continue
		}
		if pkg != nil {
			pkgs = append(pkgs, *pkg)
		}
	}

	return pkgs, nil
}

// parsePacmanDesc parses a single pacman desc file.
// The format uses %KEY% sections followed by values on subsequent lines.
func parsePacmanDesc(pf *inventory.Platform, afs *afero.Afero, descPath string) (*Package, error) {
	f, err := afs.Open(descPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fields := parsePacmanDescSections(f)

	name := fields["%NAME%"]
	version := fields["%VERSION%"]
	if name == "" {
		return nil, nil
	}

	return &Package{
		Name:        name,
		Version:     version,
		Arch:        fields["%ARCH%"],
		Description: fields["%DESC%"],
		// Pacman desc files carry %LICENSE% as a multi-line block, one
		// SPDX identifier per line; parsePacmanDescSections keeps only
		// the first which is correct for most packages. The fast
		// `pacman -Q` path doesn't surface license at all; that gap is
		// expected and gets filled in only when the FS fallback runs.
		License:        fields["%LICENSE%"],
		Format:         PacmanPkgFormat,
		FilesAvailable: PkgFilesAsync,
		PUrl:           purl.NewPackageURL(pf, purl.TypeAlpm, name, version).String(),
	}, nil
}

// parsePacmanDescSections reads a desc file and returns a map of section key to value.
func parsePacmanDescSections(r io.Reader) map[string]string {
	fields := make(map[string]string)
	scanner := bufio.NewScanner(r)
	var currentKey string

	for scanner.Scan() {
		line := scanner.Text()

		// Section header: %KEY% (at least 3 chars, exactly two % signs)
		if len(line) >= 3 && line[0] == '%' && line[len(line)-1] == '%' && strings.Count(line, "%") == 2 {
			currentKey = line
			continue
		}

		// Empty line ends a section value
		if line == "" {
			currentKey = ""
			continue
		}

		// Value line — only keep the first value line per section
		// (multi-value sections like %DEPENDS% are not needed for SBOM)
		if currentKey != "" && fields[currentKey] == "" {
			fields[currentKey] = line
		}
	}

	return fields
}

func (ppm *PacmanPkgManager) Available() (map[string]PackageUpdate, error) {
	return nil, errors.New("Available() not implemented for pacman")
}

// Files returns the file manifest for a pacman package. pacman records the
// files each package owns in <PacmanLocalDB>/<name>-<version>/files. The
// directory suffix is the full version including epoch (e.g. "1:1.6.7-1"),
// which is exactly what both `pacman -Q` and the desc files report, so no
// special epoch handling is needed. We return the path to that manifest,
// matching the async convention used by the dpkg and rpm package managers.
func (ppm *PacmanPkgManager) Files(name string, version string, arch string) ([]FileRecord, error) {
	if name == "" || version == "" {
		return nil, nil
	}

	fs := ppm.conn.FileSystem()
	// path.Join (not filepath.Join) is intentional — these are always Linux
	// filesystem paths, even when mql runs on a different OS.
	filesDB := path.Join(PacmanLocalDB, name+"-"+version, "files")
	if _, err := fs.Stat(filesDB); err != nil {
		return nil, nil
	}

	return []FileRecord{{Path: filesDB}}, nil
}

var pacmanOwnerRegex = regexp.MustCompile(`is owned by (\S+)`)

// FindFileOwner implements PkgFileOwnershipResolver via `pacman -Qo`, which
// prints "<path> is owned by <name> <version>" and exits non-zero when no
// package owns the path.
func (ppm *PacmanPkgManager) FindFileOwner(path string) (string, error) {
	if !ppm.conn.Capabilities().Has(shared.Capability_RunCommand) {
		return "", nil
	}
	cmd, err := ppm.conn.RunCommand("pacman -Qo " + shellQuote(path))
	if err != nil {
		return "", err
	}
	if cmd.ExitStatus != 0 {
		return "", nil
	}
	return parsePacmanOwner(readCommandOutput(cmd.Stdout)), nil
}

// parsePacmanOwner extracts the package name from `pacman -Qo` output of the
// form "<path> is owned by <name> <version>".
func parsePacmanOwner(output string) string {
	m := pacmanOwnerRegex.FindStringSubmatch(output)
	if m == nil {
		return ""
	}
	return m[1]
}
