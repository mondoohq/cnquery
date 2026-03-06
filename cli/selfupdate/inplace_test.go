// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package selfupdate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwapBinaryInPlace(t *testing.T) {
	t.Run("swaps binary and creates .old file", func(t *testing.T) {
		dir := t.TempDir()
		originalPath := filepath.Join(dir, "mybinary")
		stagedPath := filepath.Join(dir, "staged", "mybinary")

		// Create the "original" binary.
		require.NoError(t, os.WriteFile(originalPath, []byte("original-content"), 0o755))

		// Create the "staged" binary in a subdirectory.
		require.NoError(t, os.MkdirAll(filepath.Dir(stagedPath), 0o755))
		require.NoError(t, os.WriteFile(stagedPath, []byte("new-content"), 0o755))

		// Patch os.Executable for the test by calling swapBinaryInPlaceFrom directly.
		resultPath, err := swapBinaryInPlaceFrom(stagedPath, originalPath)
		require.NoError(t, err)
		assert.Equal(t, originalPath, resultPath)

		// The original path should now contain the staged content.
		content, err := os.ReadFile(originalPath)
		require.NoError(t, err)
		assert.Equal(t, "new-content", string(content))

		// The .old file should contain the original content.
		oldContent, err := os.ReadFile(originalPath + ".old")
		require.NoError(t, err)
		assert.Equal(t, "original-content", string(oldContent))
	})

	t.Run("rolls back on copy failure", func(t *testing.T) {
		dir := t.TempDir()
		originalPath := filepath.Join(dir, "mybinary")

		// Create the "original" binary.
		require.NoError(t, os.WriteFile(originalPath, []byte("original-content"), 0o755))

		// Staged path does not exist — copy will fail.
		stagedPath := filepath.Join(dir, "nonexistent", "mybinary")

		_, err := swapBinaryInPlaceFrom(stagedPath, originalPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to copy staged binary")

		// After rollback, the original should be restored.
		content, err := os.ReadFile(originalPath)
		require.NoError(t, err)
		assert.Equal(t, "original-content", string(content))

		// .old should not remain after rollback.
		_, err = os.Stat(originalPath + ".old")
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("succeeds with leftover .old file", func(t *testing.T) {
		dir := t.TempDir()
		originalPath := filepath.Join(dir, "mybinary")
		stagedPath := filepath.Join(dir, "staged", "mybinary")

		// Create the "original" binary and a leftover .old from a previous swap.
		require.NoError(t, os.WriteFile(originalPath, []byte("original-content"), 0o755))
		require.NoError(t, os.WriteFile(originalPath+".old", []byte("stale-old"), 0o755))

		// Create the "staged" binary.
		require.NoError(t, os.MkdirAll(filepath.Dir(stagedPath), 0o755))
		require.NoError(t, os.WriteFile(stagedPath, []byte("new-content"), 0o755))

		resultPath, err := swapBinaryInPlaceFrom(stagedPath, originalPath)
		require.NoError(t, err)
		assert.Equal(t, originalPath, resultPath)

		// Original path has the new content.
		content, err := os.ReadFile(originalPath)
		require.NoError(t, err)
		assert.Equal(t, "new-content", string(content))

		// .old now contains the previous original, not the stale leftover.
		oldContent, err := os.ReadFile(originalPath + ".old")
		require.NoError(t, err)
		assert.Equal(t, "original-content", string(oldContent))
	})

	t.Run("no-op when staged equals original", func(t *testing.T) {
		dir := t.TempDir()
		binaryPath := filepath.Join(dir, "mybinary")
		require.NoError(t, os.WriteFile(binaryPath, []byte("content"), 0o755))

		resultPath, err := swapBinaryInPlaceFrom(binaryPath, binaryPath)
		require.NoError(t, err)
		assert.Equal(t, binaryPath, resultPath)

		// No .old file should be created.
		_, err = os.Stat(binaryPath + ".old")
		assert.True(t, os.IsNotExist(err))
	})
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")

	require.NoError(t, os.WriteFile(src, []byte("hello"), 0o755))

	require.NoError(t, copyFile(src, dst))

	content, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(content))

	info, err := os.Stat(dst)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
}

func TestCleanupOldBinary(t *testing.T) {
	// CleanupOldBinary is a no-op on non-Windows because inPlaceUpdateEnabled is false.
	// We just verify it doesn't panic.
	CleanupOldBinary()
}
