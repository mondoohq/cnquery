// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/cpe"
	"go.mondoo.com/mql/v13/providers/os/resources/purl"
)

const (
	DpkgPkgFormat = "deb"
)

var (
	DPKG_REGEX = regexp.MustCompile(`^(.+):\s(.+)$`)
	// e.g. source with version: samba (2:4.17.12+dfsg-0+deb12u1)
	DPKG_ORIGIN_REGEX = regexp.MustCompile(`^\s*([^\(]*)(?:\((.*)\))?\s*$`)
)

// ParseDpkgPackages parses the dpkg database content located in /var/lib/dpkg/status
func ParseDpkgPackages(pf *inventory.Platform, input io.Reader) ([]Package, error) {
	const STATE_RESET = 0
	const STATE_DESC = 1
	pkgs := []Package{}

	add := func(pkg Package) {
		// do sanitization checks to ensure we have minimal information
		if pkg.Name != "" && pkg.Version != "" {
			pkg.PUrl = purl.NewPackageURL(pf, purl.TypeDebian, pkg.Name, pkg.Version,
				purl.WithArch(pkg.Arch),
				purl.WithEpoch(pkg.Epoch),
			).String()
			cpes, _ := cpe.NewPackage2Cpe(pkg.Name, pkg.Name, pkg.Version, pkg.Epoch, pkg.Arch)
			cpesWithoutArch, _ := cpe.NewPackage2Cpe(pkg.Name, pkg.Name, pkg.Version, pkg.Epoch, "")
			cpes = append(cpes, cpesWithoutArch...)
			pkg.CPEs = cpes
			pkgs = append(pkgs, pkg)
		} else {
			log.Debug().Msg("ignored deb packages since information is missing")
		}
	}

	scanner := bufio.NewScanner(input)
	pkg := Package{Format: DpkgPkgFormat}
	state := STATE_RESET
	var key string
	for scanner.Scan() {
		line := scanner.Text()

		// reset package definition once we reach a newline
		if len(line) == 0 {
			add(pkg)
			pkg = Package{
				Format:         DpkgPkgFormat,
				FilesAvailable: PkgFilesAsync,
			}
		}

		m := DPKG_REGEX.FindStringSubmatch(line)
		key = ""
		if m != nil {
			key = m[1]
			state = STATE_RESET
		}
		switch {
		case key == "Package":
			pkg.Name = strings.TrimSpace(m[2])
		case key == "Version":
			pkg.Version = strings.TrimSpace(m[2])
		case key == "Architecture":
			pkg.Arch = strings.TrimSpace(m[2])
		case key == "Status":
			pkg.Status = strings.TrimSpace(m[2])
		case key == "Source":
			pkg.Origin = strings.TrimSpace(m[2])
		// description supports multi-line statements, start desc
		case key == "Description":
			pkg.Description = strings.TrimSpace(m[2])
			state = STATE_DESC
		// next desc line, append to previous one
		case state == STATE_DESC:
			pkg.Description += "\n" + strings.TrimSpace(line)
		}
	}

	// if the last line is not an empty line we have things in flight, lets check it
	add(pkg)

	return pkgs, nil
}

// ParseDpkgCopyrightLicense reads the per-package DEP-5 copyright file at
// /usr/share/doc/<pkg>/copyright and returns the first `License:` value
// found anywhere in the file. Returns the empty string when the file is
// missing, not DEP-5, or no License field is present. Called lazily
// from the `license()` method on the `package` resource — only when
// MQL actually asks for the license, so we don't pay the per-package
// read cost on every `packages` enumeration.
//
// DEP-5 reference: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
// A top-level `License:` is rare in practice; most Ubuntu/Debian
// packages carry their license expression in the first `Files: *`
// paragraph (after the header's blank line). We scan the whole file
// and take the first `License:` regardless of paragraph — that yields
// the package's primary license for typical DEP-5 files and the only
// license present for single-paragraph ones.
//
// Many older packages use free-form copyright files with no `License:`
// field at all (just a `See /usr/share/common-licenses/X` pointer);
// those return empty rather than us guessing.
func ParseDpkgCopyrightLicense(fs afero.Fs, pkgName string) string {
	if fs == nil || pkgName == "" {
		return ""
	}
	path := "/usr/share/doc/" + pkgName + "/copyright"
	f, err := fs.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// DEP-5 allows multi-line license bodies indented under the
		// short name; the short name is on the same line as
		// `License:`. We only return that short name. Lines beginning
		// with whitespace are continuation/body and are skipped.
		if strings.HasPrefix(line, "License:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "License:"))
		}
	}
	return ""
}

var DPKG_UPDATE_REGEX = regexp.MustCompile(`^Inst\s([a-zA-Z0-9.\-_]+)\s\[([a-zA-Z0-9.\-\+]+)\]\s\(([a-zA-Z0-9.\-\+]+)\s*(.*)\)(.*)$`)

func ParseDpkgUpdates(input io.Reader) (map[string]PackageUpdate, error) {
	pkgs := map[string]PackageUpdate{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		m := DPKG_UPDATE_REGEX.FindStringSubmatch(line)
		if m != nil {
			pkgs[m[1]] = PackageUpdate{
				Name:      m[1],
				Version:   m[2],
				Available: m[3],
			}
		}
	}
	return pkgs, nil
}

// Debian, Ubuntu
type DebPkgManager struct {
	conn     shared.Connection
	platform *inventory.Platform
}

func (dpm *DebPkgManager) Name() string {
	return "Debian Package Manager"
}

func (dpm *DebPkgManager) Format() string {
	return DpkgPkgFormat
}

func (dpm *DebPkgManager) List() ([]Package, error) {
	fs := dpm.conn.FileSystem()
	dpkgStatusFile := "/var/lib/dpkg/status"
	dpkgStatusDir := "/var/lib/dpkg/status.d"
	_, fErr := fs.Stat(dpkgStatusFile)
	dStat, dErr := fs.Stat(dpkgStatusDir)

	if fErr != nil && dErr != nil {
		log.Debug().Err(fErr).Str("path", dpkgStatusFile).Msg("cannot find status file")
		log.Debug().Err(dErr).Str("path", dpkgStatusDir).Msg("cannot find status dir")
		return nil, fmt.Errorf("could not find dpkg package list")
	}

	pkgList := []Package{}
	// main pkg file for debian systems
	if fErr == nil {
		log.Debug().Str("file", dpkgStatusFile).Msg("parse dpkg status file")
		fi, err := fs.Open(dpkgStatusFile)
		if err != nil {
			return nil, fmt.Errorf("could not read dpkg package list")
		}
		defer fi.Close()

		list, err := ParseDpkgPackages(dpm.platform, fi)
		if err != nil {
			return nil, fmt.Errorf("could not parse dpkg package list")
		}
		pkgList = append(pkgList, list...)
	}

	// e.g. google distroless images stores their pkg data in /var/lib/dpkg/status.d/
	if dErr == nil && dStat.IsDir() {
		afutil := afero.Afero{Fs: fs}
		wErr := afutil.Walk(dpkgStatusDir, func(path string, f os.FileInfo, fErr error) error {
			if f == nil || f.IsDir() {
				return nil
			}

			log.Debug().Str("path", path).Msg("walk file")
			fi, err := fs.Open(path)
			if err != nil {
				log.Debug().Err(err).Str("path", path).Msg("could open file")
				return fmt.Errorf("could not read dpkg package list")
			}

			list, err := ParseDpkgPackages(dpm.platform, fi)
			fi.Close()
			if err != nil {
				log.Debug().Err(err).Str("path", path).Msg("could not parse")
				return fmt.Errorf("could not parse dpkg package list")
			}

			log.Debug().Int("pkgs", len(list)).Msg("completed parsing")
			pkgList = append(pkgList, list...)
			return nil
		})
		if wErr != nil {
			return nil, wErr
		}
	}

	return pkgList, nil
}

func (dpm *DebPkgManager) Available() (map[string]PackageUpdate, error) {
	// TODO: run this as a complete shell script in motor
	// DEBIAN_FRONTEND=noninteractive apt-get update >/dev/null 2>&1
	// readlock() { cat /proc/locks | awk '{print $5}' | grep -v ^0 | xargs -I {1} find /proc/{1}/fd -maxdepth 1 -exec readlink {} \; | grep '^/var/lib/dpkg/lock$'; }
	// while test -n "$(readlock)"; do sleep 1; done
	// DEBIAN_FRONTEND=noninteractive apt-get upgrade --dry-run
	_, _ = dpm.conn.RunCommand("DEBIAN_FRONTEND=noninteractive apt-get update >/dev/null 2>&1")

	cmd, err := dpm.conn.RunCommand("DEBIAN_FRONTEND=noninteractive apt-get upgrade --dry-run")
	if err != nil {
		log.Debug().Err(err).Msg("mql[packages]> could not run apt-get upgrade")
		return nil, fmt.Errorf("could not run apt-get upgrade")
	}
	return ParseDpkgUpdates(cmd.Stdout)
}

// FindFileOwner implements PkgFileOwnershipResolver via `dpkg -S`, which prints
// "<name>[:<arch>]: <path>" (possibly across several lines, including diversion
// notes) and exits non-zero when no package owns the path.
func (dpm *DebPkgManager) FindFileOwner(path string) (string, error) {
	if !dpm.conn.Capabilities().Has(shared.Capability_RunCommand) {
		return "", nil
	}
	cmd, err := dpm.conn.RunCommand("dpkg -S " + shellQuote(path))
	if err != nil {
		return "", err
	}
	if cmd.ExitStatus != 0 {
		return "", nil
	}
	return parseDpkgOwner(readCommandOutput(cmd.Stdout)), nil
}

// parseDpkgOwner extracts the package name from `dpkg -S` output. Lines look
// like "<name>[:<arch>]: <path>"; the output may include diversion notes
// ("diversion by X from: …") which are skipped because a package spec never
// contains a space. The optional ":<arch>" multiarch qualifier is stripped to
// match the names reported by List().
func parseDpkgOwner(output string) string {
	for _, line := range strings.Split(output, "\n") {
		idx := strings.Index(line, ": ")
		if idx <= 0 {
			continue
		}
		spec := strings.TrimSpace(line[:idx])
		if c := strings.IndexByte(spec, ':'); c >= 0 {
			spec = spec[:c]
		}
		if spec != "" && !strings.ContainsRune(spec, ' ') {
			return spec
		}
	}
	return ""
}

func (dpm *DebPkgManager) Files(name string, version string, arch string) ([]FileRecord, error) {
	fs := dpm.conn.FileSystem()

	dpkgListFiles := []string{
		"/var/lib/dpkg/info/" + name + ".list",
	}

	if arch != "" {
		dpkgListFiles = append(dpkgListFiles, "/var/lib/dpkg/info/"+name+":"+arch+".list")
	}

	fileRecords := []FileRecord{}
	for _, file := range dpkgListFiles {
		if _, err := fs.Stat(file); err != nil {
			continue
		}
		fileRecords = append(fileRecords, FileRecord{Path: file})
	}

	return fileRecords, nil
}
