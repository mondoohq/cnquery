// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package aimodel

import (
	"path/filepath"

	"github.com/spf13/afero"
)

// ScanTFHub discovers TensorFlow Hub modules cached at
// ~/.cache/tfhub_modules. Each module directory must contain a
// saved_model.pb file to be recognized as a valid TF SavedModel.
func ScanTFHub(afs *afero.Afero, home string) []ModelInfo {
	modulesDir := filepath.Join(home, ".cache", "tfhub_modules")
	entries, err := afs.ReadDir(modulesDir)
	if err != nil {
		return nil
	}

	var results []ModelInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		moduleDir := filepath.Join(modulesDir, e.Name())

		// TF Hub modules contain saved_model.pb
		savedModel := filepath.Join(moduleDir, "saved_model.pb")
		if exists, _ := afs.Exists(savedModel); !exists {
			continue
		}

		totalSize, modTime := dirSizeRecursive(afs, moduleDir)

		results = append(results, ModelInfo{
			Name:       e.Name(),
			Source:     "tfhub",
			Path:       moduleDir,
			Size:       totalSize,
			ModifiedAt: modTime,
			Format:     "savedmodel",
		})
	}
	return results
}
