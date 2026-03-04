// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package fsutil

import (
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
