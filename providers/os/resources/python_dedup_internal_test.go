// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The default paths overlap by design -- a specific pattern such as
// "/opt/homebrew/lib/python*" sits alongside the general "/opt/*/lib/python*"
// that also matches it. Every directory must still be scanned exactly once.
//
// Regression test for an inventory that was roughly half duplicates: on a macOS
// host python.packages returned 1085 entries for 546 distinct packages, because
// each site-packages directory was collected once per matching pattern. Every
// extra copy became another SBOM entry and another vulnerability finding for a
// package that is installed only once.
//
// These dist-info directories deliberately carry no METADATA, so the packages
// resolve through the directory-name fallback and the test needs no plugin
// runtime.
func TestCollectPythonPackagesInPaths_DeduplicatesOverlappingGlobMatches(t *testing.T) {
	fs := afero.NewMemMapFs()
	const siteDir = "/opt/homebrew/lib/python3.11/site-packages"
	require.NoError(t, fs.MkdirAll(siteDir+"/requests-2.32.3.dist-info", 0o755))

	// precondition: more than one pattern resolves to this directory, which is
	// what made the duplication possible
	matches := 0
	for _, walkPath := range walkedPaths(t, fs) {
		if walkPath == "/opt/homebrew/lib/python3.11" {
			matches++
		}
	}
	require.Greater(t, matches, 1, "precondition: the directory must be matched by several patterns")

	results, err := collectPythonPackagesInPaths(nil, fs, defaultPythonPaths)
	require.NoError(t, err)

	require.Len(t, results, 1, "a directory matched by several patterns must be collected once")
	assert.Equal(t, "requests", results[0].Name)
	assert.Equal(t, "2.32.3", results[0].Version)
}

// An install whose metadata cannot be read still has its identity in the
// directory name. Dropping it loses a package that is genuinely present -- a
// silent gap in the inventory rather than a visible error.
func TestCollectPythonPackages_FallsBackToDistInfoDirName(t *testing.T) {
	const siteDir = "/usr/lib/python3.13/site-packages"

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll(siteDir+"/litellm-1.80.15.dist-info", 0o755))

	results, err := collectPythonPackages(nil, fs, siteDir)
	require.NoError(t, err)

	require.Len(t, results, 1, "a dist-info without METADATA must still be reported")
	assert.Equal(t, "litellm", results[0].Name)
	assert.Equal(t, "1.80.15", results[0].Version)
	assert.Equal(t, "pkg:pypi/litellm@1.80.15", results[0].Purl)
	assert.Equal(t, siteDir+"/litellm-1.80.15.dist-info", results[0].File,
		"the file must point at the directory the identity came from")
}

// A dist-info directory whose name carries no version must not be turned into a
// package with a garbage version.
func TestCollectPythonPackages_DirNameFallbackWithoutVersion(t *testing.T) {
	const siteDir = "/usr/lib/python3.13/site-packages"

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll(siteDir+"/mypackage.egg-info", 0o755))

	results, err := collectPythonPackages(nil, fs, siteDir)
	require.NoError(t, err)

	require.Len(t, results, 1)
	assert.Equal(t, "mypackage", results[0].Name)
	assert.Empty(t, results[0].Version)
}

// Dependencies resolve among the siblings in the environment that declares them,
// so every package has to know which directory it was installed into.
func TestPythonPackageSiteDir(t *testing.T) {
	tests := []struct {
		name string
		file string
		want string
	}{
		{
			"dist-info metadata",
			"/usr/lib/python3.13/site-packages/requests-2.32.3.dist-info/METADATA",
			"/usr/lib/python3.13/site-packages",
		},
		{
			"egg-info pkg-info",
			"/usr/lib/python3.6/site-packages/python_dateutil-2.6.1-py3.6.egg-info/PKG-INFO",
			"/usr/lib/python3.6/site-packages",
		},
		{
			// the directory-name fallback records the dist-info directory itself
			"dist-info directory",
			"/usr/lib/python3.13/site-packages/litellm-1.80.15.dist-info",
			"/usr/lib/python3.13/site-packages",
		},
		{
			// a bare .egg-info file sits directly in the site directory
			"egg-info file",
			"/usr/lib/python3.11/site-packages/setuptools.egg-info",
			"/usr/lib/python3.11/site-packages",
		},
		{
			// packages read out of a manifest belong to the manifest's directory
			"requirements manifest",
			"/srv/app/requirements.txt",
			"/srv/app",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, pythonPackageSiteDir(tc.file))
		})
	}
}
