// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"os"
	"path/filepath"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

var _ plugin.Connection = (*KustomizeConnection)(nil)

type KustomizeConnection struct {
	plugin.Connection
	Conf           *inventory.Config
	asset          *inventory.Asset
	path           string
	kustomizations []*KustomizationEntry
}

// KustomizationEntry holds a parsed kustomization and its directory path.
type KustomizationEntry struct {
	Path          string
	Kustomization *types.Kustomization
}

func NewKustomizeConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*KustomizeConnection, error) {
	conn := &KustomizeConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	cc := asset.Connections[0]
	kustomizePath, ok := cc.Options["path"]
	if !ok || kustomizePath == "" {
		return nil, errors.New("kustomize provider requires a 'path' option")
	}
	conn.path = filepath.Clean(kustomizePath)

	entries, err := loadKustomizations(kustomizePath)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, errors.New("no kustomization.yaml found at " + kustomizePath)
	}
	conn.kustomizations = entries

	return conn, nil
}

func (c *KustomizeConnection) Name() string {
	return "kustomize"
}

func (c *KustomizeConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *KustomizeConnection) Kustomizations() []*KustomizationEntry {
	return c.kustomizations
}

func (c *KustomizeConnection) Path() string {
	return c.path
}

// kustomizationFilenames are the recognized filenames for kustomization files.
var kustomizationFilenames = []string{
	"kustomization.yaml",
	"kustomization.yml",
	"Kustomization",
}

// loadKustomizations finds and parses kustomization files from a path.
func loadKustomizations(kustomizePath string) ([]*KustomizationEntry, error) {
	fi, err := os.Stat(kustomizePath)
	if err != nil {
		return nil, err
	}

	if !fi.IsDir() {
		return nil, errors.New("kustomize path must be a directory: " + kustomizePath)
	}

	// Check if this directory has a kustomization file
	if entry, err := loadSingleKustomization(kustomizePath); err == nil {
		return []*KustomizationEntry{entry}, nil
	}

	// Otherwise scan subdirectories
	var entries []*KustomizationEntry
	dirEntries, err := os.ReadDir(kustomizePath)
	if err != nil {
		return nil, err
	}

	for _, de := range dirEntries {
		if !de.IsDir() {
			continue
		}
		subPath := filepath.Join(kustomizePath, de.Name())
		if entry, err := loadSingleKustomization(subPath); err == nil {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func loadSingleKustomization(dir string) (*KustomizationEntry, error) {
	fSys := filesys.MakeFsOnDisk()

	for _, filename := range kustomizationFilenames {
		filePath := filepath.Join(dir, filename)
		data, err := fSys.ReadFile(filePath)
		if err != nil {
			continue
		}

		var k types.Kustomization
		if err := k.Unmarshal(data); err != nil {
			return nil, err
		}
		return &KustomizationEntry{
			Path:          dir,
			Kustomization: &k,
		}, nil
	}

	return nil, errors.New("no kustomization file found in " + dir)
}
