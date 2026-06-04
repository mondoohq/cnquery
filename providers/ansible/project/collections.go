// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// VendoredCollection is a collection actually installed into the project under
// collections/ansible_collections/, as opposed to merely declared in
// requirements.yml. Comparing the two reveals drift between intended and
// installed dependencies.
type VendoredCollection struct {
	Namespace string
	Name      string // fully qualified "namespace.name"
	Version   string
	Path      string
}

// Manifest is the project's own galaxy.yml, present when the project is itself
// an Ansible collection.
type Manifest struct {
	Path      string
	Namespace string
	Name      string
	Version   string
}

// loadVendoredCollections scans collections/ansible_collections/<ns>/<name>/ for
// installed collections, reading each one's version from its galaxy.yml or
// MANIFEST.json.
func loadVendoredCollections(root string) ([]*VendoredCollection, error) {
	base := filepath.Join(root, "collections", "ansible_collections")
	namespaces, err := os.ReadDir(base)
	if err != nil {
		return nil, nil // no vendored collections is normal
	}

	var collections []*VendoredCollection
	for _, ns := range namespaces {
		if !ns.IsDir() {
			continue
		}
		names, err := os.ReadDir(filepath.Join(base, ns.Name()))
		if err != nil {
			continue
		}
		for _, name := range names {
			if !name.IsDir() {
				continue
			}
			path := filepath.Join(base, ns.Name(), name.Name())
			collections = append(collections, &VendoredCollection{
				Namespace: ns.Name(),
				Name:      ns.Name() + "." + name.Name(),
				Version:   readCollectionVersion(path),
				Path:      path,
			})
		}
	}

	sort.Slice(collections, func(i, j int) bool { return collections[i].Name < collections[j].Name })
	return collections, nil
}

// readCollectionVersion reads an installed collection's version from its
// galaxy.yml, falling back to the MANIFEST.json written by ansible-galaxy.
func readCollectionVersion(path string) string {
	if data, err := os.ReadFile(filepath.Join(path, "galaxy.yml")); err == nil {
		var g struct {
			Version string `yaml:"version"`
		}
		if yaml.Unmarshal(data, &g) == nil && g.Version != "" {
			return g.Version
		}
	}
	if data, err := os.ReadFile(filepath.Join(path, "MANIFEST.json")); err == nil {
		var m struct {
			CollectionInfo struct {
				Version string `json:"version"`
			} `json:"collection_info"`
		}
		if json.Unmarshal(data, &m) == nil {
			return m.CollectionInfo.Version
		}
	}
	return ""
}

// loadManifest reads a galaxy.yml at the project root, which marks the project
// itself as a collection. Returns nil when absent or not a collection manifest.
func loadManifest(root string) *Manifest {
	path := filepath.Join(root, "galaxy.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var g struct {
		Namespace string `yaml:"namespace"`
		Name      string `yaml:"name"`
		Version   string `yaml:"version"`
	}
	if yaml.Unmarshal(data, &g) != nil || (g.Namespace == "" && g.Name == "") {
		return nil
	}
	return &Manifest{
		Path:      path,
		Namespace: g.Namespace,
		Name:      g.Name,
		Version:   g.Version,
	}
}
