// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
)

type PkgFilesAvailable int

const (
	// PkgFilesNotAvailable means that the package manager does not provide any file information about the packages.
	// This is the default value.
	PkgFilesNotAvailable PkgFilesAvailable = 0
	// PkgFilesIncluded means that the package manager includes the files in the package metadata and can be queried
	// via the List function.
	PkgFilesIncluded PkgFilesAvailable = 1
	// PkgFilesAsync means that the package manager does not include the files in the package metadata and needs to be
	// queried asynchronously via the Files function.
	PkgFilesAsync PkgFilesAvailable = 2
)

type Package struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Epoch       string `json:"epoch,omitempty"`
	Arch        string `json:"arch"`
	Status      string `json:"status,omitempty"`
	Description string `json:"description"`

	// this may be the source package or an origin
	// e.g. on alpine it is used for parent  packages
	// o 	Package Origin - https://wiki.alpinelinux.org/wiki/Apk_spec
	Origin string `json:"origin"`
	Format string `json:"format"`

	// Package Url follows https://github.com/package-url/purl-spec
	PUrl string `json:"purl,omitempty"`

	// Package CPE
	CPEs []string `json:"cpes,omitempty"`

	// Package files (optional, only for some package managers)
	FilesAvailable PkgFilesAvailable `json:"files_available,omitempty"`
	Files          []FileRecord      `json:"files,omitempty"`

	// This is used for the CPE generation and exposed via MQL
	Vendor string `json:"vendor,omitempty"`
}

type FileRecord struct {
	Path     string      `json:"path"`
	Digest   PkgDigest   `json:"digest"`
	FileInfo PkgFileInfo `json:"permission"`
}

type PkgDigest struct {
	Value     string `json:"value"`
	Algorithm string `json:"type"`
}

type PkgFileInfo struct {
	Size  int64  `json:"size"`
	Mode  uint16 `json:"mode"`
	Flags int32  `json:"flags"`
	Owner string `json:"owner"`
	Group string `json:"group"`
}

// extends Package to store available version
type PackageUpdate struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Arch      string `json:"arch"`
	Available string `json:"available"`
	Repo      string `json:"repo"`
}

type OperatingSystemPkgManager interface {
	Name() string
	// List returns a list of Packages
	List() ([]Package, error)
	// Available returns a map of available package updates from the perspective of the package manager
	Available() (map[string]PackageUpdate, error)
	// Files returns a list of files on disk for a given package
	Files(name string, version string, arch string) ([]FileRecord, error)
}

// this will find the right package manager for the operating system
func ResolveSystemPkgManagers(conn shared.Connection) ([]OperatingSystemPkgManager, error) {
	var pms []OperatingSystemPkgManager

	asset := conn.Asset()
	if asset == nil || asset.Platform == nil {
		return nil, errors.New("cannot find OS information for package detection")
	}

	switch {
	case asset.Platform.IsFamily("arch"): // arch family
		pms = append(pms, &PacmanPkgManager{conn: conn, platform: asset.Platform})
	case asset.Platform.IsFamily("debian"): // debian family
		pms = append(pms, &DebPkgManager{conn: conn, platform: asset.Platform})
		// This is supported in Debian and Ubuntu:
		// https: // snapcraft.io/docs/distro-support
		pms = append(pms, &SnapPkgManager{conn: conn, platform: asset.Platform})
	case asset.Platform.Name == "amazonlinux" || asset.Platform.Name == "photon" || asset.Platform.Name == "wrlinux" || asset.Platform.Name == "bottlerocket":
		fallthrough
	case asset.Platform.IsFamily("redhat") || asset.Platform.IsFamily("euler") || asset.Platform.Name == "mageia": // rhel/euler/mageia based systems
		pms = append(pms, &RpmPkgManager{conn: conn, platform: asset.Platform})
		if asset.Platform.Name == "fedora" {
			// https: // snapcraft.io/docs/distro-support
			pms = append(pms, &SnapPkgManager{conn: conn, platform: asset.Platform})
		}
	case asset.Platform.IsFamily("suse"): // suse handling
		pms = append(pms, &SusePkgManager{RpmPkgManager{conn: conn, platform: asset.Platform}})
	case asset.Platform.Name == "alpine" || asset.Platform.Name == "wolfi": // alpine & wolfi share apk
		pms = append(pms, &AlpinePkgManager{conn: conn, platform: asset.Platform})
	case asset.Platform.Name == "macos": // macos family
		pms = append(pms, &MacOSPkgManager{conn: conn, platform: asset.Platform})
	case asset.Platform.Name == "windows":
		pms = append(pms, &WinPkgManager{conn: conn, platform: asset.Platform})
	case asset.Platform.Name == "scratch" || asset.Platform.Name == "coreos":
		pms = append(pms, &ScratchPkgManager{conn: conn})
	case asset.Platform.Name == "openwrt":
		pms = append(pms, &OpkgPkgManager{conn: conn})
	case asset.Platform.Name == "solaris":
		pms = append(pms, &SolarisPkgManager{conn: conn})
	case asset.Platform.Name == "cos":
		pms = append(pms, &CosPkgManager{conn: conn})
	case asset.Platform.Name == "freebsd" || asset.Platform.Name == "dragonflybsd": // both use pkg cli
		pms = append(pms, &FreeBSDPkgManager{conn: conn})
	case asset.Platform.Name == "aix":
		pms = append(pms, &AixPkgManager{conn: conn, platform: asset.Platform})
	case asset.Platform.IsFamily("linux"):
		// no clear package manager for linux platform found
		// most likely we land here if we have a yocto-based system
		opkgPaths := []string{"/bin/opkg", "/usr/bin/opkg"}
		for i := range opkgPaths {
			if _, err := conn.FileSystem().Stat(opkgPaths[i]); err == nil {
				pms = append(pms, &OpkgPkgManager{conn: conn})
				break
			}
		}
	}

	if len(pms) == 0 {
		return nil, errors.New("could not detect suitable package manager for platform: " + asset.Platform.Name)
	}

	return pms, nil
}
