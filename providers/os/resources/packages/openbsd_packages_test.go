// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOpenbsdPackages(t *testing.T) {
	f, err := os.Open("testdata/openbsd-pkg-info.txt")
	require.NoError(t, err)
	defer f.Close()

	pkgs, err := ParseOpenbsdPackages(f)
	require.NoError(t, err)
	assert.Equal(t, 11, len(pkgs))

	// Test simple package
	assert.Equal(t, "bash", pkgs[1].Name)
	assert.Equal(t, "5.2.26", pkgs[1].Version)
	assert.Equal(t, "GNU Bourne Again Shell", pkgs[1].Description)
	assert.Equal(t, OpenbsdPkgFormat, pkgs[1].Format)

	// Test package with pN revision suffix
	assert.Equal(t, "bzip2", pkgs[2].Name)
	assert.Equal(t, "1.0.8p0", pkgs[2].Version)

	// Test package with pN revision suffix in version
	assert.Equal(t, "python", pkgs[6].Name)
	assert.Equal(t, "3.11.8p0", pkgs[6].Version)

	// Test package with flavor suffix (vim-9.1.100-no_x11)
	assert.Equal(t, "vim", pkgs[8].Name)
	assert.Equal(t, "9.1.100-no_x11", pkgs[8].Version)

	// Test package with hyphen in name (git-lfs)
	assert.Equal(t, "git-lfs", pkgs[10].Name)
	assert.Equal(t, "3.4.1", pkgs[10].Version)

	// Verify all packages have the openbsd format
	for _, pkg := range pkgs {
		assert.Equal(t, OpenbsdPkgFormat, pkg.Format)
	}
}
