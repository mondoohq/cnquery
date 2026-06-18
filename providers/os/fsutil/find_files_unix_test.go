// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package fsutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindFilesPermissionFilter(t *testing.T) {
	fs := afero.NewMemMapFs()
	mkDir(t, fs, "root/a")
	mkDir(t, fs, "root/b")
	mkDir(t, fs, "root/c")
	mkDir(t, fs, "root/c/d")

	mkFile(t, fs, "root/file0")
	mkFile(t, fs, "root/a/file1")
	mkFile(t, fs, "root/a/file2")
	mkFile(t, fs, "root/b/file1")
	mkFile(t, fs, "root/c/file4")
	mkFile(t, fs, "root/c/d/file5")

	require.NoError(t, fs.Chmod("root/c/file4", 0o002))

	perm := uint32(0o002)
	permFiles, err := FindFiles(afero.NewIOFS(fs), "root", nil, "f", &perm, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, permFiles, []string{"root/c/file4"})
}

// TestFindFilesZeroPermissionMatchesAll is a regression test for
// https://github.com/mondoohq/mql/issues/8467
//
// files.find built via CreateResource without a permissions argument reaches
// the native walker with a 0 mask (initFilesFind's 0o777 default only applies
// to MQL-instantiated resources). A 0 mask must mean "no permission filter"
// (like `find -perm -0`), not "match nothing" — otherwise the pam/sudoers/
// modprobe/rsyslog/... `*.d` discoveries return no files on filesystem/tar
// connections.
func TestFindFilesZeroPermissionMatchesAll(t *testing.T) {
	fs := afero.NewMemMapFs()
	mkDir(t, fs, "root")
	mkFile(t, fs, "root/a")
	mkFile(t, fs, "root/b")
	require.NoError(t, fs.Chmod("root/a", 0o644))
	require.NoError(t, fs.Chmod("root/b", 0o600))

	zero := uint32(0)
	files, err := FindFiles(afero.NewIOFS(fs), "root", nil, "f", &zero, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"root/a", "root/b"}, files,
		"a 0 permission mask must match all files, not none")
}

// TestFindFilesFollowsSymlinks is a regression test for
// https://github.com/mondoohq/mql/issues/8467
//
// authselect manages key /etc/pam.d service files (system-auth, password-auth,
// ...) as symlinks into /etc/authselect. The command-based backend runs
// `find -L` and follows them, but the native walker reported lstat info and
// skipped symlinks, so type:"file" missed those service files and pam.module
// resolved to an empty husk. A real filesystem is needed here because afero's
// MemMapFs has no symlink support.
func TestFindFilesFollowsSymlinks(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "real.conf"), []byte("x"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))
	require.NoError(t, os.Symlink("real.conf", filepath.Join(dir, "link.conf"))) // -> regular file
	require.NoError(t, os.Symlink("sub", filepath.Join(dir, "dirlink")))         // -> directory
	require.NoError(t, os.Symlink("missing", filepath.Join(dir, "broken")))      // dangling

	iofs := os.DirFS(dir)

	// type:"file" follows symlinks to regular files (mirrors `find -L`).
	files, err := FindFiles(iofs, ".", nil, "file", nil, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"real.conf", "link.conf"}, files,
		"type:file must include a symlink that points at a regular file")

	// type:"link" still matches every symlink (additive), including the one
	// that resolves to a file/dir and the dangling one.
	links, err := FindFiles(iofs, ".", nil, "link", nil, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"link.conf", "dirlink", "broken"}, links,
		"type:link must still match all symlinks")

	// type:"directory" follows a symlink that points at a directory.
	dirs, err := FindFiles(iofs, ".", nil, "directory", nil, nil)
	require.NoError(t, err)
	assert.Contains(t, dirs, "sub")
	assert.Contains(t, dirs, "dirlink",
		"type:directory must include a symlink that points at a directory")
}
