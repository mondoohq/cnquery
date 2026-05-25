// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package aimodel

import (
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// ScanKeras discovers models cached by Keras (~/.keras/models).
// Supports .h5 (HDF5) and .keras (native Keras v3) formats.
func ScanKeras(afs *afero.Afero, home string) []ModelInfo {
	modelsDir := filepath.Join(home, ".keras", "models")
	entries, err := afs.ReadDir(modelsDir)
	if err != nil {
		return nil
	}

	var results []ModelInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		lower := strings.ToLower(e.Name())
		if !strings.HasSuffix(lower, ".h5") && !strings.HasSuffix(lower, ".keras") {
			continue
		}

		format := "h5"
		if strings.HasSuffix(lower, ".keras") {
			format = "keras"
		}

		name := strings.TrimSuffix(strings.TrimSuffix(e.Name(), ".h5"), ".keras")

		results = append(results, ModelInfo{
			Name:       name,
			Source:     "keras",
			Path:       filepath.Join(modelsDir, e.Name()),
			Size:       e.Size(),
			ModifiedAt: e.ModTime(),
			Format:     format,
		})
	}
	return results
}
