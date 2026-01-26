// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNetBSDPackages(t *testing.T) {
	data, err := os.Open("testdata/packages_netbsd.txt")
	require.NoError(t, err)
	defer data.Close()

	pkgs, err := ParseNetBSDPackages(data)
	require.NoError(t, err)
	assert.Equal(t, 10, len(pkgs))

	// Test first package (bash)
	assert.Equal(t, "bash", pkgs[0].Name)
	assert.Equal(t, "5.1.16", pkgs[0].Version)
	assert.Equal(t, "The GNU Bourne Again Shell", pkgs[0].Description)
	assert.Equal(t, "shells/bash", pkgs[0].Origin)
	assert.Equal(t, "x86_64", pkgs[0].Arch)
	assert.Equal(t, NetbsdPkgFormat, pkgs[0].Format)

	// Test second package (curl)
	assert.Equal(t, "curl", pkgs[1].Name)
	assert.Equal(t, "8.0.1", pkgs[1].Version)
	assert.Equal(t, "Command line tool for transferring files with URL syntax", pkgs[1].Description)
	assert.Equal(t, "www/curl", pkgs[1].Origin)

	// Test third package (nginx with nb suffix)
	assert.Equal(t, "nginx", pkgs[2].Name)
	assert.Equal(t, "1.24.0nb1", pkgs[2].Version)
	assert.Equal(t, "Lightweight HTTP server and mail proxy server", pkgs[2].Description)
	assert.Equal(t, "www/nginx", pkgs[2].Origin)

	// Test package with hyphen in name (perl)
	assert.Equal(t, "perl", pkgs[3].Name)
	assert.Equal(t, "5.36.0", pkgs[3].Version)

	// Test package with complex name (git-base)
	assert.Equal(t, "git-base", pkgs[6].Name)
	assert.Equal(t, "2.40.0", pkgs[6].Version)

	// Test package with version suffix (sudo)
	assert.Equal(t, "sudo", pkgs[7].Name)
	assert.Equal(t, "1.9.13p3", pkgs[7].Version)

	// Test package with complex version (python39)
	assert.Equal(t, "python39", pkgs[5].Name)
	assert.Equal(t, "3.9.16", pkgs[5].Version)

	// Verify all packages have the netbsd format
	for _, pkg := range pkgs {
		assert.Equal(t, NetbsdPkgFormat, pkg.Format)
	}
}
