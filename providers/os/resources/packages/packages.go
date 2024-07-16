// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
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

	// This is used for the CPE generation
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
func ResolveSystemPkgManager(conn shared.Connection) (OperatingSystemPkgManager, error) {
	var pm OperatingSystemPkgManager

	asset := conn.Asset()
	if asset == nil || asset.Platform == nil {
		return nil, errors.New("cannot find OS information for package detection")
	}

	switch {
	case asset.Platform.IsFamily("arch"): // arch family
		pm = &PacmanPkgManager{conn: conn, platform: asset.Platform}
	case asset.Platform.IsFamily("debian"): // debian family
		pm = &DebPkgManager{conn: conn, platform: asset.Platform}
	case asset.Platform.Name == "amazonlinux" || asset.Platform.Name == "photon" || asset.Platform.Name == "wrlinux":
		fallthrough
	case asset.Platform.IsFamily("redhat"): // rhel family
		pm = &RpmPkgManager{conn: conn, platform: asset.Platform}
	case asset.Platform.IsFamily("suse"): // suse handling
		pm = &SusePkgManager{RpmPkgManager{conn: conn, platform: asset.Platform}}
	case asset.Platform.Name == "alpine" || asset.Platform.Name == "wolfi": // alpine & wolfi share apk
		pm = &AlpinePkgManager{conn: conn, platform: asset.Platform}
	case asset.Platform.Name == "macos": // mac os family
		pm = &MacOSPkgManager{conn: conn}
	case asset.Platform.Name == "windows":
		pm = &WinPkgManager{conn: conn, platform: asset.Platform}
	case asset.Platform.Name == "scratch" || asset.Platform.Name == "coreos":
		pm = &ScratchPkgManager{conn: conn}
	case asset.Platform.Name == "openwrt":
		pm = &OpkgPkgManager{conn: conn}
	case asset.Platform.Name == "solaris":
		pm = &SolarisPkgManager{conn: conn}
	case asset.Platform.Name == "cos":
		pm = &CosPkgManager{conn: conn}
	case asset.Platform.Name == "freebsd":
		pm = &FreeBSDPkgManager{conn: conn}
	case asset.Platform.Name == "aix":
		pm = &AixPkgManager{conn: conn, platform: asset.Platform}
	case asset.Platform.IsFamily("linux"):
		// no clear package manager for linux platform found
		// most likely we land here if we have a yocto-based system
		opkgPaths := []string{"/bin/opkg", "/usr/bin/opkg"}
		for i := range opkgPaths {
			if _, err := conn.FileSystem().Stat(opkgPaths[i]); err == nil {
				pm = &OpkgPkgManager{conn: conn}
				break
			}
		}
	}

	if pm == nil {
		return nil, errors.New("could not detect suitable package manager for platform: " + asset.Platform.Name)
	}

	return pm, nil
}
