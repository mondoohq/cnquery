// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package aiapp

import (
	"time"

	"github.com/spf13/afero"
)

// AppInfo holds metadata for a discovered AI application.
type AppInfo struct {
	Name      string
	Category  string // "desktop", "ide-extension", "browser-extension"
	Version   string
	Vendor    string
	Path      string
	Installed bool
	UpdatedAt time.Time
}

// DetectContext carries shared state needed by every detector.
type DetectContext struct {
	Fs       *afero.Afero
	Home     string
	OSFamily string
}

// Detector discovers locally installed AI applications from a single source.
type Detector interface {
	Detect(ctx DetectContext) []AppInfo
}

// Detectors returns all registered application detectors.
func Detectors() []Detector {
	return []Detector{
		&DesktopDetector{},
		&VSCodeDetector{},
		&ChromeDetector{},
	}
}

// DetectAll runs every detector and returns the combined results.
func DetectAll(afs *afero.Afero, home, osFamily string) []AppInfo {
	ctx := DetectContext{Fs: afs, Home: home, OSFamily: osFamily}
	var all []AppInfo
	for _, d := range Detectors() {
		all = append(all, d.Detect(ctx)...)
	}
	return all
}
