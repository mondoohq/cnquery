// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
)

func TestParseSnapMeta(t *testing.T) {
	// Create a test SnapPkgManager with a mock platform
	spm := &SnapPkgManager{
		platform: &inventory.Platform{
			Name:    "ubuntu",
			Version: "22.04",
			Arch:    "amd64",
		},
	}

	// Open the test manifest file
	manifestFile, err := os.Open("testdata/snap.yaml")
	if err != nil {
		t.Fatalf("Failed to open test manifest file: %v", err)
	}
	defer manifestFile.Close()

	// Parse the manifest
	pkg, err := spm.parseSnapManifest(manifestFile)
	if err != nil {
		t.Fatalf("Failed to parse manifest: %v", err)
	}

	// Verify the parsed package
	assert.Equal(t, "dbgate", pkg.Name)
	assert.Equal(t, "6.1.0", pkg.Version)
	assert.Equal(t, SnapPkgFormat, pkg.Format)
	assert.Contains(t, pkg.Description, "database")
	assert.Equal(t, "pkg:snap/ubuntu/dbgate@6.1.0?arch=amd64", pkg.PUrl)
}
