// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package binaries

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsFdSupported(t *testing.T) {
	// This test will only pass on Linux amd64
	expected := runtime.GOOS == "linux" && runtime.GOARCH == "amd64"
	assert.Equal(t, expected, IsFdSupported())
}

func TestGetFdPath(t *testing.T) {
	if !IsFdSupported() {
		t.Skip("fd is not supported on this platform")
	}

	// Test getting fd path
	fdPath, err := GetFdPath()
	require.NoError(t, err)
	require.NotEmpty(t, fdPath)

	// Verify the file exists and is executable
	info, err := os.Stat(fdPath)
	require.NoError(t, err)
	assert.False(t, info.IsDir())
	assert.True(t, info.Mode()&0o111 != 0, "fd binary should be executable")

	// Test that calling it again returns the same path
	fdPath2, err := GetFdPath()
	require.NoError(t, err)
	assert.Equal(t, fdPath, fdPath2)

	// Clean up
	CleanupFdBinary()

	// Verify cleanup worked
	_, err = os.Stat(fdPath)
	assert.True(t, os.IsNotExist(err))
}

func TestIsFdAvailable(t *testing.T) {
	if !IsFdSupported() {
		t.Skip("fd is not supported on this platform")
		assert.False(t, IsFdAvailable())
		return
	}

	// fd should be available on supported platforms
	available := IsFdAvailable()
	assert.True(t, available)

	// Clean up
	CleanupFdBinary()
}

func TestFdBinarySize(t *testing.T) {
	if !IsFdSupported() {
		t.Skip("fd is not supported on this platform")
	}

	// The embedded binary should have reasonable size (>1MB, <10MB)
	assert.Greater(t, len(fdLinuxAmd64), 1024*1024, "fd binary should be larger than 1MB")
	assert.Less(t, len(fdLinuxAmd64), 10*1024*1024, "fd binary should be smaller than 10MB")
}

func TestFdPathInTempDir(t *testing.T) {
	if !IsFdSupported() {
		t.Skip("fd is not supported on this platform")
	}

	fdPath, err := GetFdPath()
	require.NoError(t, err)

	// Verify it's in a temp directory
	tempDir := os.TempDir()
	assert.True(t, filepath.HasPrefix(fdPath, tempDir), "fd path should be in temp directory")

	// Verify the binary name
	assert.Equal(t, "fd", filepath.Base(fdPath))

	// Clean up
	CleanupFdBinary()
}
