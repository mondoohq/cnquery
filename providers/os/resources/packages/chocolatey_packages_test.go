// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNuspec(t *testing.T) {
	afs := &afero.Afero{Fs: afero.NewOsFs()}

	// Test git.nuspec (v2015 schema, has dependencies)
	pkg, err := parseNuspec(afs, "./testdata/chocolatey/git/git.nuspec")
	require.NoError(t, err)
	assert.Equal(t, "git", pkg.Name)
	assert.Equal(t, "2.45.2", pkg.Version)
	assert.Equal(t, "The Git Development Community", pkg.Author)
	assert.Equal(t, "Git (install) is a fast, scalable, distributed revision control system.", pkg.Summary)
	assert.Contains(t, pkg.Description, "Git for Windows")
	assert.Equal(t, "https://opensource.org/licenses/GPL-2.0", pkg.LicenseUrl)
	assert.Equal(t, "https://gitforwindows.org/", pkg.ProjectUrl)
	assert.Equal(t, []string{"git", "vcs", "dvcs", "version-control", "admin"}, pkg.Tags)
	assert.Equal(t, []string{"git.install"}, pkg.Dependencies)

	// Test 7zip.nuspec (v2010 schema, no dependencies)
	pkg, err = parseNuspec(afs, "./testdata/chocolatey/7zip/7zip.nuspec")
	require.NoError(t, err)
	assert.Equal(t, "7zip", pkg.Name)
	assert.Equal(t, "24.08", pkg.Version)
	assert.Equal(t, "Igor Pavlov", pkg.Author)
	assert.Nil(t, pkg.Dependencies)

	// Test chocolatey.nuspec (multiple dependencies)
	pkg, err = parseNuspec(afs, "./testdata/chocolatey/chocolatey/chocolatey.nuspec")
	require.NoError(t, err)
	assert.Equal(t, "chocolatey", pkg.Name)
	assert.Equal(t, "2.3.0", pkg.Version)
	assert.Equal(t, []string{"chocolatey.lib", "chocolatey-core.extension"}, pkg.Dependencies)
}

func TestNewChocolateyPurl(t *testing.T) {
	assert.Equal(t, "pkg:chocolatey/git@2.45.2", newChocolateyPurl("git", "2.45.2"))
	assert.Equal(t, "pkg:chocolatey/7zip@24.08", newChocolateyPurl("7zip", "24.08"))
}

func TestChocolateyParseFromLib(t *testing.T) {
	afs := &afero.Afero{Fs: afero.NewOsFs()}

	mgr := &ChocolateyPkgManager{}
	pkgs, err := mgr.parseFromLib(afs, "./testdata/chocolatey")
	require.NoError(t, err)

	assert.Equal(t, 3, len(pkgs))

	// Find git package — should be pinned (.pin file exists)
	var gitPkg *ChocolateyPackage
	for i := range pkgs {
		if pkgs[i].Name == "git" {
			gitPkg = &pkgs[i]
			break
		}
	}
	require.NotNil(t, gitPkg)
	assert.True(t, gitPkg.Pinned)
	assert.Equal(t, "pkg:chocolatey/git@2.45.2", gitPkg.Purl)
	assert.Equal(t, "testdata/chocolatey/git", gitPkg.Path)

	// 7zip should not be pinned
	var zipPkg *ChocolateyPackage
	for i := range pkgs {
		if pkgs[i].Name == "7zip" {
			zipPkg = &pkgs[i]
			break
		}
	}
	require.NotNil(t, zipPkg)
	assert.False(t, zipPkg.Pinned)
}
