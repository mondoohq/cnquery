// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/purl"
	"gopkg.in/yaml.v3"
)

const (
	SnapPkgFormat = "snap"
)

type SnapPkgManager struct {
	conn     shared.Connection
	platform *inventory.Platform
}

func (spm *SnapPkgManager) Name() string {
	return "Snap Package Manager"
}

func (spm *SnapPkgManager) Format() string {
	return SnapPkgFormat
}

func (spm *SnapPkgManager) List() ([]Package, error) {
	fs := spm.conn.FileSystem()
	snapDir := "/snap"
	afs := &afero.Afero{Fs: fs}
	_, dErr := afs.Stat(snapDir)
	if dErr != nil {
		log.Debug().Str("path", snapDir).Msg("cannot find snap dir")
		return []Package{}, nil
	}

	// e.g. /snap/firefox/6103/meta/snap.yaml
	// https://snapcraft.io/docs/the-snap-format#p-3326-setup-files
	snapRegEx := regexp.MustCompile(`/snap/[^/]+/\d+/meta/snap\.yaml`)
	files := []string{}
	err := afs.Walk(snapDir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		if !snapRegEx.MatchString(path) {
			return nil
		}

		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	pkgList := []Package{}
	for _, file := range files {
		manifest, err := afs.Open(file)
		if err != nil {
			log.Error().Err(err).Str("file", file).Msg("could not open manifest file")
			continue
		}
		pkg, err := spm.parseSnapManifest(manifest)
		if err != nil {
			log.Error().Err(err).Str("file", file).Msg("could not parse manifest file")
			manifest.Close()
			continue
		}
		pkgList = append(pkgList, pkg)
		manifest.Close()
	}
	return pkgList, nil
}

func (spm *SnapPkgManager) Available() (map[string]PackageUpdate, error) {
	return nil, nil
}

func (spm *SnapPkgManager) Files(name string, version string, arch string) ([]FileRecord, error) {
	return nil, nil
}

// SnapManifest represents the structure of a meta/snap.yaml file
type SnapManifest struct {
	Name          string   `yaml:"name"`
	Version       string   `yaml:"version"`
	Summary       string   `yaml:"summary"`
	Description   string   `yaml:"description"`
	Architectures []string `yaml:"architectures"`
}

// parseSnapManifest parses a snap manifest file and returns a package
func (spm *SnapPkgManager) parseSnapManifest(manifest io.Reader) (Package, error) {
	manifestBytes, err := io.ReadAll(manifest)
	if err != nil {
		return Package{}, err
	}

	var snapManifest SnapManifest
	if err := yaml.Unmarshal(manifestBytes, &snapManifest); err != nil {
		return Package{}, fmt.Errorf("failed to parse snap manifest: %w", err)
	}

	arch := ""
	if len(snapManifest.Architectures) > 0 {
		arch = snapManifest.Architectures[0]
	}

	return Package{
		Name:        snapManifest.Name,
		Version:     snapManifest.Version,
		Format:      SnapPkgFormat,
		Description: snapManifest.Description,
		Arch:        arch,
		PUrl: purl.NewPackageURL(
			spm.platform,
			purl.TypeSnap,
			snapManifest.Name,
			snapManifest.Version,
			purl.WithArch(arch),
		).String(),
	}, nil
}
