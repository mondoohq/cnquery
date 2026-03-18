// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tmp

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFile_Default(t *testing.T) {
	f, err := File()
	require.NoError(t, err)
	defer os.Remove(f.Name())
	defer f.Close()

	assert.FileExists(t, f.Name())
}

func TestFile_CustomTmpDir(t *testing.T) {
	customDir := t.TempDir()
	t.Setenv("MONDOO_TMP_DIR", customDir)

	f, err := File()
	require.NoError(t, err)
	defer os.Remove(f.Name())
	defer f.Close()

	assert.True(t, strings.HasPrefix(f.Name(), customDir))
}

func TestDir_Default(t *testing.T) {
	dir, err := Dir()
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestDir_CustomTmpDir(t *testing.T) {
	customDir := t.TempDir()
	t.Setenv("MONDOO_TMP_DIR", customDir)

	dir, err := Dir()
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	assert.True(t, strings.HasPrefix(dir, customDir))
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}
