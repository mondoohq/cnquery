// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers/os/fsutil"
)

// walkedPaths returns every directory defaultPythonPaths resolves to on the
// given filesystem. This is the discovery half of collectPythonPackagesInPaths,
// isolated so it can be exercised without a plugin runtime.
func walkedPaths(t *testing.T, fs afero.Fs) []string {
	t.Helper()
	found := []string{}
	err := fsutil.WalkGlob(fs, defaultPythonPaths, func(fs afero.Fs, walkPath string) error {
		found = append(found, walkPath)
		return nil
	})
	require.NoError(t, err)
	return found
}

// afero.Glob is based on filepath.Match, where "*" matches any run of
// non-separator characters -- including a leading dot. That is what makes
// "/app/*/lib/python*" match "/app/.venv/...", unlike shell globbing. The venv
// patterns all depend on this, so pin the behavior.
func TestDefaultPythonPaths_GlobMatchesDotDirectories(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/app/.venv/lib/python3.13/site-packages", 0o755))

	assert.Contains(t, walkedPaths(t, fs), "/app/.venv/lib/python3.13")
}

func TestDefaultPythonPaths_VirtualenvLayouts(t *testing.T) {
	tests := []struct {
		name string
		dir  string
		want string
	}{
		// the layout that prompted this: a container app with its deps in a venv
		{"app dot-venv", "/app/.venv/lib/python3.13/site-packages", "/app/.venv/lib/python3.13"},
		{"app plain venv", "/app/venv/lib/python3.12/site-packages", "/app/venv/lib/python3.12"},
		{"app env", "/app/env/lib/python3.11/site-packages", "/app/env/lib/python3.11"},
		{"opt venv", "/opt/venv/lib/python3.12/site-packages", "/opt/venv/lib/python3.12"},
		{"opt app venv", "/opt/myapp/lib/python3.12/site-packages", "/opt/myapp/lib/python3.12"},
		{"rootfs venv", "/venv/lib/python3.12/site-packages", "/venv/lib/python3.12"},
		{"rootfs dot-venv", "/.venv/lib/python3.12/site-packages", "/.venv/lib/python3.12"},
		{"code", "/code/.venv/lib/python3.12/site-packages", "/code/.venv/lib/python3.12"},
		{"workspace", "/workspace/.venv/lib/python3.12/site-packages", "/workspace/.venv/lib/python3.12"},
		{"srv", "/srv/app/lib/python3.12/site-packages", "/srv/app/lib/python3.12"},
		{"usr src app", "/usr/src/app/.venv/lib/python3.12/site-packages", "/usr/src/app/.venv/lib/python3.12"},
		{"root home", "/root/.venv/lib/python3.12/site-packages", "/root/.venv/lib/python3.12"},
		{"virtualenvwrapper", "/home/dev/.virtualenvs/proj/lib/python3.12/site-packages", "/home/dev/.virtualenvs/proj/lib/python3.12"},
		{"user venv", "/home/dev/.venv/lib/python3.12/site-packages", "/home/dev/.venv/lib/python3.12"},
		// non-dotted venv names must work wherever dotted ones do
		{"user plain venv", "/home/dev/venv/lib/python3.12/site-packages", "/home/dev/venv/lib/python3.12"},
		{"user env", "/home/dev/env/lib/python3.12/site-packages", "/home/dev/env/lib/python3.12"},
		{"macos user venv", "/Users/dev/venv/lib/python3.12/site-packages", "/Users/dev/venv/lib/python3.12"},
		// a project checkout carrying its own venv, one level below a known root
		{"project under app", "/app/myservice/.venv/lib/python3.12/site-packages", "/app/myservice/.venv/lib/python3.12"},
		{"project under workspace", "/workspace/proj/.venv/lib/python3.12/site-packages", "/workspace/proj/.venv/lib/python3.12"},
		{"project under opt", "/opt/myservice/.venv/lib/python3.12/site-packages", "/opt/myservice/.venv/lib/python3.12"},
		// dist-packages is resolved by the caller, so the walk only needs the parent
		{"debian dist-packages", "/usr/lib/python3/dist-packages", "/usr/lib/python3"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			require.NoError(t, fs.MkdirAll(tc.dir, 0o755))
			assert.Contains(t, walkedPaths(t, fs), tc.want)
		})
	}
}

// Documents where the glob list deliberately stops. Without "**" the depth has
// to be bounded, and under user homes -- where a shared machine can have many
// users each with many project directories -- an extra level costs a ReadDir per
// directory. Container roots like /app are shallow and purpose-built, so they do
// get the extra level (see "project under app" above).
//
// If this test starts failing because a deeper pattern was added, that is a
// deliberate trade-off, not a bug: update it and note the scan-cost impact.
func TestDefaultPythonPaths_DeepUserProjectVenvNotCovered(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/home/dev/proj/.venv/lib/python3.12/site-packages", 0o755))

	assert.NotContains(t, walkedPaths(t, fs), "/home/dev/proj/.venv/lib/python3.12",
		"a venv nested under a project directory in a user home is a known gap; use the python resource's path argument")
}

// The system paths must keep working exactly as before.
func TestDefaultPythonPaths_SystemLayoutsStillMatch(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/usr/lib/python3.11/site-packages", 0o755))
	require.NoError(t, fs.MkdirAll("/usr/local/lib/python3.9/site-packages", 0o755))

	found := walkedPaths(t, fs)
	assert.Contains(t, found, "/usr/lib/python3.11")
	assert.Contains(t, found, "/usr/local/lib/python3.9")
}

// A glob that lands on a file instead of a directory must not abort the whole
// search -- with patterns this broad, one odd match would otherwise cost us
// every other venv on the system.
//
// This one runs on a real filesystem on purpose: reading "<file>/site-packages"
// has to fail with ENOTDIR rather than "not exist" for the skip to be exercised,
// and MemMapFs does not reproduce that distinction.
func TestCollectPythonPackagesInPaths_SkipsUnreadableMatches(t *testing.T) {
	fs := afero.NewBasePathFs(afero.NewOsFs(), t.TempDir())

	// /opt/*/lib/python* matches this, but it is a file, so reading
	// "<match>/site-packages" fails with ENOTDIR
	require.NoError(t, fs.MkdirAll("/opt/weird/lib", 0o755))
	f, err := fs.Create("/opt/weird/lib/python3.12")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	// a real venv alongside it, which must still be found
	require.NoError(t, fs.MkdirAll("/app/.venv/lib/python3.13/site-packages", 0o755))

	found := []string{}
	err = fsutil.WalkGlob(fs, defaultPythonPaths, func(fs afero.Fs, walkPath string) error {
		found = append(found, walkPath)
		return nil
	})
	require.NoError(t, err)
	require.Contains(t, found, "/opt/weird/lib/python3.12", "precondition: the file must be matched by a glob")

	_, err = collectPythonPackagesInPaths(nil, fs, defaultPythonPaths)
	assert.NoError(t, err, "an unreadable match must be skipped, not fatal")
}
