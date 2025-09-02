// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/resources/parsers"
	"go.mondoo.com/cnquery/v12/providers/os/resources/purl"
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

	pkgs := make([]Package, len(data[0].Items))
	for i, entry := range data[0].Items {
		// We need a special handling for Firefox to determine ESR installations
		purlQualifiers := getPurlQualifiers(conn, entry)
		pkgs[i].Name = entry.Name
		pkgs[i].Version = entry.Version
		pkgs[i].Format = MacosPkgFormat
		pkgs[i].FilesAvailable = PkgFilesIncluded
		pkgs[i].Arch = platform.Arch
		pkgs[i].PUrl = purl.NewPackageURL(
			platform, purl.TypeMacos, entry.Name, entry.Version, purl.WithQualifiers(purlQualifiers),
		).String()
		if entry.Path != "" {
			pkgs[i].Files = []FileRecord{
				{
					Path: entry.Path,
				},
			}
		}
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
