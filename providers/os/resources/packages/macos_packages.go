// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"fmt"
	"io"
	"strings"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	plist "howett.net/plist"
)

const (
	MacosPkgFormat = "macos"
)

// parse macos system version property list
func ParseMacOSPackages(input io.Reader) ([]Package, error) {
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

	type sysProfilerItems struct {
		Name    string `plist:"_name"`
		Version string `plist:"version"`
		Path    string `plist:"path"`
	}

	type sysProfiler struct {
		Items []sysProfilerItems `plist:"_items"`
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
		pkgs[i].Name = entry.Name
		pkgs[i].Version = entry.Version
		pkgs[i].Format = MacosPkgFormat
		pkgs[i].FilesAvailable = PkgFilesIncluded
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
	conn shared.Connection
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

	return ParseMacOSPackages(cmd.Stdout)
}

func (mpm *MacOSPkgManager) Available() (map[string]PackageUpdate, error) {
	return nil, errors.New("cannot determine available packages for macOS")
}

func (mpm *MacOSPkgManager) Files(name string, version string, arch string) ([]FileRecord, error) {
	// nothing extra to be done here since the list is already included in the package list
	return nil, nil
}
