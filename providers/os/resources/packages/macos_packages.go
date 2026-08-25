// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers/os/connection/shared"
	"go.mondoo.com/mql/providers/os/resources/parsers"
	"go.mondoo.com/mql/providers/os/resources/purl"
	plist "howett.net/plist"
)

const (
	MacosPkgFormat = "macos"
)

type sysProfilerItem struct {
	Name    string `plist:"_name"`
	Version string `plist:"version"`
	Path    string `plist:"path"`
}

type sysProfiler struct {
	Items []sysProfilerItem `plist:"_items"`
}

// infoPlist holds the version keys we care about from an app bundle's
// Contents/Info.plist.
type infoPlist struct {
	ShortVersion  string `plist:"CFBundleShortVersionString"`
	BundleVersion string `plist:"CFBundleVersion"`
}

// parse macos system version property list
func ParseMacOSPackages(conn shared.Connection, platform *inventory.Platform, input io.Reader) ([]Package, error) {
	var r io.ReadSeeker
	r, ok := input.(io.ReadSeeker)

	// if the read seaker is not implemented lets cache stdout in-memory
	if !ok {
		packageList, err := io.ReadAll(input)
		if err != nil {
			return nil, err
		}
		r = strings.NewReader(string(packageList))
	}

	var data []sysProfiler
	decoder := plist.NewDecoder(r)
	err := decoder.Decode(&data)
	if err != nil {
		return nil, err
	}

	if len(data) != 1 {
		return nil, errors.New("format not supported")
	}

	pkgs := make([]Package, 0, len(data[0].Items))
	for _, entry := range data[0].Items {
		// system_profiler only surfaces CFBundleShortVersionString as the
		// version. Some bundles (e.g. PWAs) ship a version only in
		// CFBundleVersion, so fall back to the bundle's Info.plist when
		// system_profiler reports no version.
		version := entry.Version
		if version == "" {
			// system_profiler lists every path carrying a bundle-like
			// extension, not only application bundles: Firefox origin storage
			// (https+++example.app), app-group script containers
			// (group.is.workflow.my.app) and daemon directories all show up. It
			// names them after the basename minus its final dot component, so a
			// reverse-DNS directory collapses into a fragment like "com" or
			// "org.eclipse". An application bundle always carries a
			// Contents/Info.plist, so its absence means this is not an installed
			// application: drop the entry instead of reporting it with a
			// versionless purl that can never match advisory data.
			//
			// Entries that do report a version are never checked, because some
			// real applications have no Contents/Info.plist. Wrapped iOS apps
			// keep theirs under Wrapper/ and would otherwise be lost.
			bundleVersion, isBundle := bundleVersionFromInfoPlist(conn, entry.Path)
			if !isBundle {
				log.Debug().
					Str("name", entry.Name).
					Str("path", entry.Path).
					Msg("skipping entry that is not an application bundle")
				continue
			}
			version = bundleVersion
		}

		// We need a special handling for Firefox to determine ESR installations
		purlQualifiers := getPurlQualifiers(conn, entry)

		pkg := Package{
			Name:           entry.Name,
			Version:        version,
			Format:         MacosPkgFormat,
			FilesAvailable: PkgFilesIncluded,
			Arch:           platform.Arch,
			PUrl: purl.NewPackageURL(
				platform, purl.TypeMacos, entry.Name, version, purl.WithQualifiers(purlQualifiers),
			).String(),
		}
		if entry.Path != "" {
			pkg.Files = []FileRecord{
				{
					Path: entry.Path,
				},
			}
		}
		pkgs = append(pkgs, pkg)
	}

	return pkgs, nil
}

// MacOS
type MacOSPkgManager struct {
	conn     shared.Connection
	platform *inventory.Platform
}

func (mpm *MacOSPkgManager) Name() string {
	return "macOS Package Manager"
}

func (mpm *MacOSPkgManager) Format() string {
	return MacosPkgFormat
}

func (mpm *MacOSPkgManager) List() ([]Package, error) {
	cmd, err := mpm.conn.RunCommand("system_profiler SPApplicationsDataType -xml")
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}

	return ParseMacOSPackages(mpm.conn, mpm.platform, cmd.Stdout)
}

func (mpm *MacOSPkgManager) Available() (map[string]PackageUpdate, error) {
	return nil, errors.New("cannot determine available packages for macOS")
}

func (mpm *MacOSPkgManager) Files(name string, version string, arch string) ([]FileRecord, error) {
	// nothing extra to be done here since the list is already included in the package list
	return nil, nil
}

// bundleVersionFromInfoPlist recovers an app's version from its
// Contents/Info.plist when system_profiler did not report one. It prefers
// CFBundleShortVersionString (the user-facing version) and falls back to
// CFBundleVersion (the build version).
//
// The second return value reports whether the path is an application bundle at
// all, which is true whenever a Contents/Info.plist could be read. Callers use
// it to tell a real application that simply carries no version (some Apple
// CoreServices apps ship neither version key) from a directory that merely has
// a bundle-like extension.
func bundleVersionFromInfoPlist(conn shared.Connection, path string) (string, bool) {
	if path == "" {
		return "", false
	}

	infoPath := filepath.Join(path, "Contents", "Info.plist")
	f, err := conn.FileSystem().Open(infoPath)
	if err != nil {
		log.Debug().Err(err).Str("path", infoPath).Msg("could not open Info.plist")
		return "", false
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		log.Debug().Err(err).Str("path", infoPath).Msg("could not read Info.plist")
		return "", false
	}

	// The Info.plist is there, so this is a bundle even if we cannot read a
	// version out of it.
	var info infoPlist
	if _, err := plist.Unmarshal(content, &info); err != nil {
		log.Debug().Err(err).Str("path", infoPath).Msg("could not parse Info.plist")
		return "", true
	}

	if info.ShortVersion != "" {
		return info.ShortVersion, true
	}
	return info.BundleVersion, true
}

func getPurlQualifiers(conn shared.Connection, entry sysProfilerItem) map[string]string {
	qualifiers := make(map[string]string)
	if entry.Name == "Firefox" {
		appIni := ""
		if entry.Path != "" {
			appIni = filepath.Join(entry.Path, "Contents", "Resources", "application.ini")
		}
		if appIni != "" {
			f, err := conn.FileSystem().Open(appIni)
			if err != nil {
				log.Debug().Err(err).Msg("could not open application.ini")
				return nil
			}
			defer f.Close()
			content, err := io.ReadAll(f)
			if err != nil {
				log.Debug().Err(err).Msg("could not read application.ini")
				return nil
			}
			ini := parsers.ParseIni(string(content), "=")
			if ini != nil {
				if data, ok := ini.Fields["App"]; ok {
					fields, ok := data.(map[string]any)
					if ok {
						if name, ok := fields["RemotingName"]; ok {
							qualifiers["remoting-name"] = name.(string)
						}
					}
				}
			}
		}
	}
	return qualifiers
}
