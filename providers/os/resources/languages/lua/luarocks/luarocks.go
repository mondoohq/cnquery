// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package luarocks

import (
	"bufio"
	"io"
	"path"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/providers/os/resources/languages"
	lualang "go.mondoo.com/mql/v13/providers/os/resources/languages/lua"
	"go.mondoo.com/mql/v13/sbom"
)

// Extractor parses LuaRocks installed package listings.
type Extractor struct{}

func (e *Extractor) Name() string {
	return "LuaRocks Extractor"
}

// Parse parses `luarocks list --porcelain` output.
// Format: NAME\tVERSION\tSTATUS\tROCKS_DIR (tab-delimited)
func (e *Extractor) Parse(r io.Reader, filename string) (languages.Bom, error) {
	pkgs, _ := ParseLuaRocksList(r, filename)
	return &luarocksBom{transitive: pkgs}, nil
}

// ParseLuaRocksList parses the porcelain output of `luarocks list --porcelain`.
// Format: NAME\tVERSION\tSTATUS\tROCKS_DIR (tab-delimited)
func ParseLuaRocksList(r io.Reader, evidencePath string) (languages.Packages, []string) {
	var pkgs languages.Packages
	var filePaths []string
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}

		name := strings.TrimSpace(fields[0])
		version := strings.TrimSpace(fields[1])

		if name == "" {
			continue
		}

		// Use ROCKS_DIR (field 3) for evidence if available, otherwise fall back
		evidence := evidencePath
		if len(fields) >= 4 {
			rocksDir := strings.TrimSpace(fields[3])
			if rocksDir != "" {
				rockspecPath := path.Join(rocksDir, name, version)
				evidence = rockspecPath
				filePaths = append(filePaths, rockspecPath)
			}
		}

		pkg := &languages.Package{
			Name:    name,
			Version: version,
			Purl:    lualang.NewPackageUrl(name, version),
		}
		if evidence != "" {
			pkg.EvidenceList = []*sbom.Evidence{lualang.NewEvidence(evidence)}
		}

		pkgs = append(pkgs, pkg)
	}

	return pkgs, filePaths
}

// ParseRocksDir scans a LuaRocks rocks directory for installed packages.
// Structure: ROCKS_DIR/NAME/VERSION/*.rockspec
func ParseRocksDir(afs *afero.Afero, rocksDir string) (languages.Packages, []string) {
	entries, err := afs.ReadDir(rocksDir)
	if err != nil {
		return nil, nil
	}

	var pkgs languages.Packages
	var filePaths []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pkgName := entry.Name()

		// path.Join (not filepath.Join) — always Linux paths
		pkgDir := path.Join(rocksDir, pkgName)
		versions, err := afs.ReadDir(pkgDir)
		if err != nil {
			continue
		}

		for _, vEntry := range versions {
			if !vEntry.IsDir() {
				continue
			}
			version := vEntry.Name()

			// Look for .rockspec file inside
			versionDir := path.Join(pkgDir, version)
			files, err := afs.ReadDir(versionDir)
			if err != nil {
				continue
			}

			hasRockspec := false
			for _, f := range files {
				if strings.HasSuffix(f.Name(), ".rockspec") {
					hasRockspec = true
					filePaths = append(filePaths, path.Join(versionDir, f.Name()))
					break
				}
			}

			if !hasRockspec {
				// Even without a rockspec, the directory name is enough
				filePaths = append(filePaths, versionDir)
			}

			pkgs = append(pkgs, &languages.Package{
				Name:    pkgName,
				Version: version,
				Purl:    lualang.NewPackageUrl(pkgName, version),
				EvidenceList: []*sbom.Evidence{
					lualang.NewEvidence(versionDir),
				},
			})
		}
	}

	if len(pkgs) == 0 {
		return nil, nil
	}

	log.Debug().Int("count", len(pkgs)).Str("dir", rocksDir).Msg("mql[lua]> found luarocks packages")
	return pkgs, filePaths
}

type luarocksBom struct {
	transitive languages.Packages
}

func (b *luarocksBom) Root() *languages.Package {
	return nil
}

func (b *luarocksBom) Direct() languages.Packages {
	return nil
}

func (b *luarocksBom) Transitive() languages.Packages {
	return b.transitive
}
