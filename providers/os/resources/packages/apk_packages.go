// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	cpe2 "go.mondoo.com/mql/v13/providers/os/resources/cpe"
	"go.mondoo.com/mql/v13/providers/os/resources/purl"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

const (
	AlpinePkgFormat   = "apk"
	ApkDbInstalled    = "/lib/apk/db/installed"
	ApkDbInstalledUsr = "/usr/lib/apk/db/installed"
)

var APK_REGEX = regexp.MustCompile(`^([A-Za-z]):(.*)$`)

// ParseApkDbPackages parses the database of the apk package manager located in
// `/lib/apk/db/installed`
// Apk spec: https://wiki.alpinelinux.org/wiki/Apk_spec
func ParseApkDbPackages(pf *inventory.Platform, input io.Reader) []Package {
	pkgs := []Package{}

	var pkgVersion string
	var pkgEpoch string

	add := func(pkg Package) {
		// merge version and epoch
		if pkgEpoch == "0" || pkgEpoch == "" {
			pkg.Version = pkgVersion
		} else {
			pkg.Version = pkgEpoch + ":" + pkgVersion
			pkg.Epoch = pkgEpoch
		}

		pkg.Format = AlpinePkgFormat
		pkg.PUrl = purl.NewPackageURL(pf, purl.TypeApk, pkg.Name, pkg.Version,
			purl.WithArch(pkg.Arch),
			purl.WithEpoch(pkg.Epoch),
		).String()

		cpes, _ := cpe2.NewPackage2Cpe(pkg.Vendor, pkg.Name, pkg.Version, "", pf.Arch)
		pkg.CPEs = cpes

		pkg.FilesAvailable = PkgFilesIncluded
		pkg.Files = append(pkg.Files, FileRecord{
			Path: ApkDbInstalled,
		})

		// do sanitization checks to ensure we have minimal information
		if pkg.Name != "" && pkg.Version != "" {
			pkgs = append(pkgs, pkg)
		} else {
			log.Debug().Msg("ignored apk package since information is missing")
		}
	}

	scanner := bufio.NewScanner(input)
	pkg := Package{}
	var key string
	for scanner.Scan() {
		line := scanner.Text()

		// reset package definition once we reach a newline
		if len(line) == 0 {
			add(pkg)
			// reset values
			pkgEpoch = ""
			pkgVersion = ""
			pkg = Package{}
		}

		m := APK_REGEX.FindStringSubmatch(line)
		key = ""
		if m != nil {
			key = m[1]
		}

		// if we short line, we ignore it since this is not a valid line
		if len(line) < 2 {
			continue
		}

		// Parse the package name or version.
		switch key {
		case "P":
			pkg.Name = m[2] // package name
		case "V":
			pkgVersion = m[2] // package version
		case "A":
			pkg.Arch = m[2] // architecture
		case "t":
			pkgEpoch = m[2] // epoch
		case "o":
			pkg.Origin = m[2] // origin
		case "T":
			pkg.Description = m[2] // description
		case "L":
			pkg.License = m[2] // license (SPDX expression)
		}
	}

	// if the last line is not an empty line we have things in flight, lets check it
	add(pkg)
	return pkgs
}

var APK_UPDATE_REGEX = regexp.MustCompile(`^([a-zA-Z0-9._]+)-([a-zA-Z0-9.\-\+]+)\s+<\s([a-zA-Z0-9.\-\+]+)\s*$`)

func ParseApkUpdates(input io.Reader) (map[string]PackageUpdate, error) {
	pkgs := map[string]PackageUpdate{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		m := APK_UPDATE_REGEX.FindStringSubmatch(line)
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

// Arch, Manjaro
type AlpinePkgManager struct {
	conn     shared.Connection
	platform *inventory.Platform
}

func (apm *AlpinePkgManager) Name() string {
	return "apk Package Manager"
}

func (apm *AlpinePkgManager) Format() string {
	return AlpinePkgFormat
}

func (apm *AlpinePkgManager) List() ([]Package, error) {
	fr, err := apm.conn.FileSystem().Open(ApkDbInstalled)
	if err != nil {
		// Wolfi uses /usr/lib/apk/db/installed (usrmerge layout)
		fr, err = apm.conn.FileSystem().Open(ApkDbInstalledUsr)
		if err != nil {
			return nil, fmt.Errorf("could not read apk package list")
		}
	}
	defer fr.Close()

	return ParseApkDbPackages(apm.platform, fr), nil
}

func (apm *AlpinePkgManager) Available() (map[string]PackageUpdate, error) {
	// it only works if apk is updated
	_, _ = apm.conn.RunCommand("apk update")

	// determine package updates
	cmd, err := apm.conn.RunCommand("apk version -v -l '<'")
	if err != nil {
		log.Debug().Err(err).Msg("mql[packages]> could not read package updates")
		return nil, fmt.Errorf("could not read apk package update list")
	}
	return ParseApkUpdates(cmd.Stdout)
}

func (apm *AlpinePkgManager) Files(name string, version string, arch string) ([]FileRecord, error) {
	// not yet implemented
	return nil, nil
}

var apkOwnerRegex = regexp.MustCompile(`is owned by (\S+)`)

// FindFileOwner implements PkgFileOwnershipResolver via `apk info --who-owns`,
// which prints "<path> is owned by <name>-<version>-r<rel>" and exits non-zero
// when no package owns the path.
func (apm *AlpinePkgManager) FindFileOwner(path string) (string, error) {
	if !apm.conn.Capabilities().Has(shared.Capability_RunCommand) {
		return "", nil
	}
	cmd, err := apm.conn.RunCommand("apk info --who-owns " + shellQuote(path))
	if err != nil {
		return "", err
	}
	if cmd.ExitStatus != 0 {
		return "", nil
	}
	return parseApkOwner(readCommandOutput(cmd.Stdout)), nil
}

// parseApkOwner extracts the package name from `apk info --who-owns` output of
// the form "<path> is owned by <name>-<version>-r<rel>".
func parseApkOwner(output string) string {
	m := apkOwnerRegex.FindStringSubmatch(output)
	if m == nil {
		return ""
	}
	return apkStripVersion(m[1])
}

// apkReleaseSuffix matches the trailing "-r<digits>" apk release component,
// anchored at the end so a stray "-r" inside a version segment (e.g. a
// hypothetical "1.0-rc1") is not mistaken for the release separator.
var apkReleaseSuffix = regexp.MustCompile(`-r\d+$`)

// apkStripVersion turns an apk "name-version-rREL" token into the bare package
// name. apk names can contain hyphens while versions cannot, so we strip the
// trailing "-rREL" release and then the "-version" component rather than
// splitting on the first hyphen.
func apkStripVersion(s string) string {
	if loc := apkReleaseSuffix.FindStringIndex(s); loc != nil {
		s = s[:loc[0]]
	}
	if i := strings.LastIndex(s, "-"); i >= 0 {
		s = s[:i]
	}
	return s
}
