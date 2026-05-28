// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectPythonPackagesFromContents(t *testing.T) {
	t.Run("parses a requirements.txt from in-memory content", func(t *testing.T) {
		contents := map[string]string{
			"requirements.txt": "requests==2.31.0\nurllib3>=2.0.0\n",
		}
		pkgs := collectPythonPackagesFromContents(contents)
		names := map[string]string{}
		for _, p := range pkgs {
			names[p.Name] = p.Version
		}
		require.Equal(t, "2.31.0", names["requests"])
		require.Contains(t, names, "urllib3")
		for _, p := range pkgs {
			require.Equal(t, "requirements.txt", p.File)
		}
	})

	t.Run("prefers poetry.lock over requirements.txt", func(t *testing.T) {
		poetryLock := `[[package]]
name = "django"
version = "4.2.7"
description = "A high-level Python Web framework."

[metadata]
lock-version = "2.0"
python-versions = "^3.11"
content-hash = "abc123"
`
		contents := map[string]string{
			"poetry.lock":      poetryLock,
			"requirements.txt": "flask==3.0.0\n",
		}
		pkgs := collectPythonPackagesFromContents(contents)
		require.NotEmpty(t, pkgs)
		for _, p := range pkgs {
			require.Equal(t, "poetry.lock", p.File, "requirements.txt must be ignored when a lockfile is present")
		}
		names := map[string]bool{}
		for _, p := range pkgs {
			names[p.Name] = true
		}
		require.True(t, names["django"], "expected django from poetry.lock, got %v", names)
		require.False(t, names["flask"], "flask must not appear when poetry.lock is present")
	})

	t.Run("ignores unknown filenames", func(t *testing.T) {
		pkgs := collectPythonPackagesFromContents(map[string]string{
			"README.md": "not a manifest",
		})
		require.Empty(t, pkgs)
	})
}
