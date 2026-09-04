// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsApplicationBundlePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"application bundle", "/Applications/Docker.app", true},
		{"OS-shipped application", "/System/Applications/Utilities/Terminal.app", true},
		{"per-user application", "/Users/user/Applications/Chrome Apps.localized/Teams (PWA).app", true},
		{"trailing separator", "/Applications/Docker.app/", true},
		// APFS is case-insensitive by default, so this bundle launches.
		{"uppercase extension", "/Applications/Docker.APP", true},
		// Only a whole Contents segment means "inside a bundle".
		{"Contents as a name fragment", "/Applications/TableOfContents/Docker.app", true},

		// The reported bug: a renamed bundle keeps its old version forever, and
		// system_profiler names it "Docker.app" because it strips only .back.
		{"renamed bundle", "/Applications/Docker.app.back", false},
		{"disabled bundle", "/Applications/Docker.app.disabled", false},
		{"service bundle", "/System/Library/Services/Spotlight.service", false},
		{"non-bundle directory", "/Users/user/Library/HTTPStorages/com.apple.ctcategories.service", false},
		{"no path reported", "", false},

		// Helper bundles ship with, and are patched by, their container.
		{"helper in a renamed bundle", "/Applications/Docker.app.back/Contents/MacOS/Docker Desktop.app", false},
		{"login item helper", "/Applications/Docker.app.back/Contents/Library/LoginItems/DockerHelper.app", false},
		{"Finder pseudo-application", "/System/Library/CoreServices/Finder.app/Contents/Applications/AirDrop.app", false},
		{"framework XPC service", "/System/Library/Frameworks/PaperKit.framework/Contents/LinkedNotesUIService.app", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, isApplicationBundlePath(test.path))
		})
	}
}
