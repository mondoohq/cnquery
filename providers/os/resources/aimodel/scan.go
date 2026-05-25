// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package aimodel

import (
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/afero"
)

// ModelInfo holds the metadata for a single discovered AI model cache entry.
// Each scanner populates what it can; fields left empty mean the source
// doesn't provide that information.
type ModelInfo struct {
	Name          string
	Source        string
	Vendor        string
	Family        string
	Path          string
	Size          int64
	ModifiedAt    time.Time
	Format        string
	Version       string
	Quantization  string
	ParameterSize string
	Architecture  string
	License       string
	Tags          []string
	Description   string
}

var (
	reQuantization = regexp.MustCompile(`(?i)(Q[0-9]+_[A-Z0-9_]+|F16|F32|FP16|FP32)`)
	// Leading separator (dash, underscore, colon, space) avoids matching "b" inside words.
	reParamSize = regexp.MustCompile(`(?i)[-_: ](\d+\.?\d*)[bB](?:[-_. ]|$)`)
)

// ScanAll runs every scanner and returns the combined results.
func ScanAll(afs *afero.Afero, home, osFamily string) []ModelInfo {
	var all []ModelInfo
	scanners := []func(*afero.Afero, string) []ModelInfo{
		ScanOllama,
		ScanHuggingFace,
		ScanLMStudio,
		func(fs *afero.Afero, h string) []ModelInfo { return ScanGPT4All(fs, h, osFamily) },
		ScanPyTorchHub,
		ScanKeras,
		ScanTFHub,
		ScanJan,
	}
	for _, scan := range scanners {
		all = append(all, scan(afs, home)...)
	}
	return all
}

// --- Helpers ---

func dirSizeAndLatestMtime(afs *afero.Afero, dir string) (int64, time.Time) {
	var totalSize int64
	var latest time.Time
	entries, err := afs.ReadDir(dir)
	if err != nil {
		return 0, latest
	}
	for _, e := range entries {
		if e.IsDir() || strings.HasSuffix(e.Name(), ".lock") {
			continue
		}
		totalSize += e.Size()
		if e.ModTime().After(latest) {
			latest = e.ModTime()
		}
	}
	return totalSize, latest
}

func dirSizeRecursive(afs *afero.Afero, dir string) (int64, time.Time) {
	var totalSize int64
	var latest time.Time
	_ = afero.Walk(afs, dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		totalSize += info.Size()
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})
	return totalSize, latest
}

func isHex(s string) bool {
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}
