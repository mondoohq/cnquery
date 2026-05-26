// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/python"
)

func TestCollectPythonManifestPackages_PipfileLock(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = afero.WriteFile(fs, "/project/Pipfile.lock", []byte(`{
		"_meta": {"hash": {"sha256": "abc"}, "pipfile-spec": 6},
		"default": {
			"openai": {"version": "==1.30.0"},
			"requests": {"version": "==2.32.0"}
		},
		"develop": {}
	}`), 0644)

	results := collectPythonManifestPackages(fs, "/project")

	require.Len(t, results, 2)
	names := map[string]string{}
	for _, r := range results {
		names[r.Name] = r.Version
	}
	assert.Equal(t, "1.30.0", names["openai"])
	assert.Equal(t, "2.32.0", names["requests"])

	for _, r := range results {
		assert.NotEmpty(t, r.Purl, "should have PURL for %s", r.Name)
		assert.Equal(t, "/project/Pipfile.lock", r.File)
	}
}

func TestCollectPythonManifestPackages_PoetryLock(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = afero.WriteFile(fs, "/project/poetry.lock", []byte(`
[[package]]
name = "anthropic"
version = "0.25.0"

[[package]]
name = "httpx"
version = "0.27.0"
`), 0644)

	results := collectPythonManifestPackages(fs, "/project")

	require.Len(t, results, 2)
	names := map[string]string{}
	for _, r := range results {
		names[r.Name] = r.Version
	}
	assert.Equal(t, "0.25.0", names["anthropic"])
	assert.Equal(t, "0.27.0", names["httpx"])
}

func TestCollectPythonManifestPackages_RequirementsTxt(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = afero.WriteFile(fs, "/project/requirements.txt", []byte(`
transformers>=4.40.0
torch==2.3.0
numpy
# comment line
`), 0644)

	results := collectPythonManifestPackages(fs, "/project")

	require.Len(t, results, 3)
	byName := map[string]string{}
	for _, r := range results {
		byName[r.Name] = r.Version
	}
	assert.Contains(t, byName, "transformers")
	assert.Contains(t, byName, "torch")
	assert.Contains(t, byName, "numpy")

	// Pinned versions (==) should be extracted
	assert.Equal(t, "2.3.0", byName["torch"])
	// Unpinned constraints (>=) have empty version
	assert.Equal(t, "", byName["transformers"])
	// Bare names have empty version
	assert.Equal(t, "", byName["numpy"])
}

func TestCollectPythonManifestPackages_LockFilePriority(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Both Pipfile.lock and requirements.txt exist — lock file should win
	_ = afero.WriteFile(fs, "/project/Pipfile.lock", []byte(`{
		"_meta": {"hash": {"sha256": "abc"}, "pipfile-spec": 6},
		"default": {"openai": {"version": "==1.30.0"}},
		"develop": {}
	}`), 0644)
	_ = afero.WriteFile(fs, "/project/requirements.txt", []byte("openai\ntorch\n"), 0644)

	results := collectPythonManifestPackages(fs, "/project")

	// Should use Pipfile.lock (1 package with version), not requirements.txt (2 packages without)
	require.Len(t, results, 1)
	assert.Equal(t, "openai", results[0].Name)
	assert.Equal(t, "1.30.0", results[0].Version)
}

func TestCollectPythonManifestPackages_NoManifest(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = fs.Mkdir("/project", 0755)

	results := collectPythonManifestPackages(fs, "/project")
	assert.Nil(t, results)
}

func TestCollectPythonManifestPackages_SetupPy(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = afero.WriteFile(fs, "/project/setup.py", []byte(`
from setuptools import setup

setup(
    name="myproject",
    version="1.0.0",
    install_requires=[
        'openai==1.30.0',
        "transformers==4.40.0",
        'numpy>=1.21',
    ],
)
`), 0644)

	results := collectPythonManifestPackages(fs, "/project")

	require.Len(t, results, 2)
	byName := map[string]string{}
	for _, r := range results {
		byName[r.Name] = r.Version
	}
	assert.Equal(t, "1.30.0", byName["openai"])
	assert.Equal(t, "4.40.0", byName["transformers"])

	for _, r := range results {
		assert.Equal(t, "/project/setup.py", r.File)
		assert.NotEmpty(t, r.Purl, "should have PURL for %s", r.Name)
	}
}

func TestCollectPythonManifestPackages_SetupCfg(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = afero.WriteFile(fs, "/project/setup.cfg", []byte(`
[options]
install_requires =
    requests==2.31.0
    anthropic==0.25.0
`), 0644)

	results := collectPythonManifestPackages(fs, "/project")

	require.Len(t, results, 2)
	byName := map[string]string{}
	for _, r := range results {
		byName[r.Name] = r.Version
	}
	assert.Equal(t, "2.31.0", byName["requests"])
	assert.Equal(t, "0.25.0", byName["anthropic"])
}

func TestCollectPythonManifestPackages_LockFileBeatsSetupPy(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Both Pipfile.lock and setup.py exist — lock file should win
	_ = afero.WriteFile(fs, "/project/Pipfile.lock", []byte(`{
		"_meta": {"hash": {"sha256": "abc"}, "pipfile-spec": 6},
		"default": {"openai": {"version": "==1.30.0"}},
		"develop": {}
	}`), 0644)
	_ = afero.WriteFile(fs, "/project/setup.py", []byte(`
setup(install_requires=['openai==1.28.0', 'torch==2.3.0'])
`), 0644)

	results := collectPythonManifestPackages(fs, "/project")

	require.Len(t, results, 1)
	assert.Equal(t, "openai", results[0].Name)
	assert.Equal(t, "1.30.0", results[0].Version)
}

func TestMergePythonPackages(t *testing.T) {
	primary := []python.PackageDetails{
		{Name: "openai", Version: "1.30.0"},
		{Name: "requests", Version: "2.32.0"},
	}
	secondary := []python.PackageDetails{
		{Name: "openai", Version: "1.28.0"}, // duplicate, should be skipped
		{Name: "torch", Version: "2.3.0"},   // new, should be added
	}

	merged := mergePythonPackages(primary, secondary)

	require.Len(t, merged, 3)
	names := map[string]string{}
	for _, p := range merged {
		names[p.Name] = p.Version
	}
	assert.Equal(t, "1.30.0", names["openai"], "primary should take precedence")
	assert.Equal(t, "2.32.0", names["requests"])
	assert.Equal(t, "2.3.0", names["torch"])
}

func TestMergePythonPackages_CaseInsensitive(t *testing.T) {
	primary := []python.PackageDetails{
		{Name: "PyYAML", Version: "6.0.1"},
	}
	secondary := []python.PackageDetails{
		{Name: "pyyaml", Version: "6.0.0"},
	}

	merged := mergePythonPackages(primary, secondary)
	require.Len(t, merged, 1)
	assert.Equal(t, "PyYAML", merged[0].Name)
}

func TestMergePythonPackages_EmptySecondary(t *testing.T) {
	primary := []python.PackageDetails{
		{Name: "openai", Version: "1.30.0"},
	}

	merged := mergePythonPackages(primary, nil)
	assert.Equal(t, primary, merged)
}
