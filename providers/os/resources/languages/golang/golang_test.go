// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package golang

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPackageUrl(t *testing.T) {
	tests := []struct {
		modulePath string
		version    string
		expected   string
	}{
		{"github.com/pkg/errors", "v0.9.1", "pkg:golang/github.com/pkg/errors@v0.9.1"},
		{"golang.org/x/sync", "v0.3.0", "pkg:golang/golang.org/x/sync@v0.3.0"},
		{"github.com/foo/bar/v2", "v2.1.0", "pkg:golang/github.com/foo/bar/v2@v2.1.0"},
		{"example.com/v2", "v2.0.0", "pkg:golang/example.com/v2@v2.0.0"},
		{"github.com/example/myproject", "", "pkg:golang/github.com/example/myproject"},
	}

	for _, tt := range tests {
		t.Run(tt.modulePath, func(t *testing.T) {
			result := NewPackageUrl(tt.modulePath, tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewCpes(t *testing.T) {
	tests := []struct {
		modulePath string
		version    string
		name       string
	}{
		// Standard 3-part path
		{"github.com/pkg/errors", "v0.9.1", "standard 3-part path"},
		// Major version suffix: should use "bar" as product, not "v2"
		{"github.com/foo/bar/v2", "v2.1.0", "major version suffix"},
		// 2-part path
		{"example.com/mylib", "v1.0.0", "2-part path"},
		// Edge case: 2-part path where name looks like a version
		{"example.com/v2", "v2.0.0", "2-part path with version-like name"},
		// Single part
		{"mypackage", "v1.0.0", "single part"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			cpes := NewCpes(tt.modulePath, tt.version)
			// CPE generation may or may not succeed depending on the input,
			// but it must never panic.
			_ = cpes
		})
	}
}

func TestNewCpesMajorVersionStripping(t *testing.T) {
	// For "github.com/foo/bar/v2", the product should be "bar" not "v2"
	cpes := NewCpes("github.com/foo/bar/v2", "v2.1.0")
	assert.NotEmpty(t, cpes)
	// Verify the CPE contains "bar" as the product, not "v2"
	assert.Contains(t, cpes[0], ":bar:")
	assert.NotContains(t, cpes[0], ":v2:")
}

func TestNewCpesEdgeCaseTwoPartVersion(t *testing.T) {
	// For "example.com/v2", this is a 2-part path where the name happens
	// to look like a major version. The major version stripping logic
	// should handle this gracefully.
	cpes := NewCpes("example.com/v2", "v2.0.0")
	// Should not panic and should produce some result
	_ = cpes
}
