// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// ErrNoKustomization is the sentinel returned by loadSingleKustomization
// when the directory has none of the recognized kustomization filenames.
// Callers use errors.Is to distinguish "look elsewhere" from a real
// parse failure on a file that exists.
var ErrNoKustomization = errors.New("no kustomization file found")

var (
	_ plugin.Connection = (*KustomizeConnection)(nil)
	_ plugin.Closer     = (*KustomizeConnection)(nil)
)

type KustomizeConnection struct {
	plugin.Connection
	Conf           *inventory.Config
	asset          *inventory.Asset
	path           string
	kustomizations []*KustomizationEntry
	closer         func()
}

// KustomizationEntry holds a parsed kustomization and its directory path.
type KustomizationEntry struct {
	Path          string
	Kustomization *types.Kustomization
}

func NewKustomizeConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*KustomizeConnection, error) {
	if asset == nil || len(asset.Connections) == 0 {
		return nil, errors.New("kustomize provider requires an asset with at least one connection")
	}

	conn := &KustomizeConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	// If a git clone is performed below, clean up the temporary directory on any
	// error path. Close() is a no-op when nothing was cloned, and the guard is
	// disarmed once the connection is returned and takes ownership of cleanup.
	cleanupClone := true
	defer func() {
		if cleanupClone {
			conn.Close()
		}
	}()

	cc := asset.Connections[0]
	kustomizePath := cc.Options["path"]
	// When discovered from a git repository (e.g. by the GitHub provider) the
	// asset carries the repo URL plus a repo-relative path to the kustomization
	// directory. Clone the repo and resolve the directory within the checkout.
	// The path option is kept relative so the detector can name the asset from
	// the repo rather than the temporary clone directory.
	if _, ok := cc.Options["http-url"]; ok {
		clonePath, closer, err := plugin.NewGitClone(asset)
		if err != nil {
			return nil, err
		}
		conn.closer = closer
		kustomizePath = filepath.Join(clonePath, kustomizePath)
	}
	if kustomizePath == "" {
		return nil, errors.New("kustomize provider requires a 'path' option")
	}
	conn.path = filepath.Clean(kustomizePath)

	// Discover from the cleaned path so entry.Path matches conn.path and
	// downstream cache keys (kustomize.kustomization:<path>) are stable
	// across "./foo" vs "./foo/" inputs.
	entries, err := loadKustomizations(conn.path)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, errors.New("no kustomization.yaml found at " + conn.path)
	}
	conn.kustomizations = entries

	cleanupClone = false
	return conn, nil
}

// Close cleans up any temporary directory created by a git clone.
func (c *KustomizeConnection) Close() {
	if c.closer != nil {
		c.closer()
	}
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

// scanSkipDirs are subdir names that should not be scanned during
// kustomization discovery. Hidden dirs (starting with `.`) are also
// skipped — they're handled separately via the prefix check.
var scanSkipDirs = map[string]struct{}{
	"node_modules": {},
	"vendor":       {},
	"target":       {},
	"dist":         {},
	"build":        {},
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

	// Check if this directory has a kustomization file.
	// "no kustomization file found" → fall through to subdir scan.
	// Any other error (malformed YAML, unreadable file) is a real
	// failure the caller should see, not silently swallow.
	entry, err := loadSingleKustomization(kustomizePath)
	if err == nil {
		return []*KustomizationEntry{entry}, nil
	}
	if !errors.Is(err, ErrNoKustomization) {
		return nil, err
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
		name := de.Name()
		// Skip hidden dirs (.git, .terraform, .idea, etc.) and a
		// short skip-list of well-known noise dirs. A misconfigured
		// path on a repo root would otherwise spend file handles on
		// directories that can't contain a kustomization.
		if strings.HasPrefix(name, ".") {
			continue
		}
		if _, skip := scanSkipDirs[name]; skip {
			continue
		}
		subPath := filepath.Join(kustomizePath, name)
		entry, err := loadSingleKustomization(subPath)
		if err == nil {
			entries = append(entries, entry)
			continue
		}
		// Quiet skip when the subdir simply has no kustomization
		// file; loud warning when a file exists but won't parse.
		if !errors.Is(err, ErrNoKustomization) {
			log.Warn().Err(err).Str("path", subPath).Msg("failed to load kustomization; skipping")
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

		k := &types.Kustomization{}
		if err := k.Unmarshal(data); err != nil {
			return nil, err
		}
		return &KustomizationEntry{
			Path:          dir,
			Kustomization: k,
		}, nil
	}

	return nil, ErrNoKustomization
}
