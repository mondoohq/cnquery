// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package binaries

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

// Embedded fd binary for Linux amd64
//
//go:embed fd-linux-amd64
var fdLinuxAmd64 []byte

var (
	// fdBinaryPath holds the path to the extracted fd binary
	fdBinaryPath string
	fdMutex      sync.Mutex
	fdExtracted  bool
)

// IsFdSupported returns true if fd is supported on the current platform
func IsFdSupported() bool {
	return runtime.GOOS == "linux" && runtime.GOARCH == "amd64"
}

// GetFdPath returns the path to the fd binary, extracting it if necessary
func GetFdPath() (string, error) {
	fdMutex.Lock()
	defer fdMutex.Unlock()

	if !IsFdSupported() {
		return "", fmt.Errorf("fd is not supported on %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	if fdExtracted && fdBinaryPath != "" {
		// Check if the file still exists
		if _, err := os.Stat(fdBinaryPath); err == nil {
			return fdBinaryPath, nil
		}
		// If file doesn't exist, reset and re-extract
		fdExtracted = false
		fdBinaryPath = ""
	}

	// Extract the binary to a temporary file
	tmpDir, err := os.MkdirTemp("", "mondoo-fd-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	fdBinaryPath = filepath.Join(tmpDir, "fd")

	file, err := os.OpenFile(fdBinaryPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", fmt.Errorf("failed to create fd binary file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, bytes.NewReader(fdLinuxAmd64))
	if err != nil {
		os.Remove(fdBinaryPath)
		return "", fmt.Errorf("failed to write fd binary: %w", err)
	}

	fdExtracted = true
	return fdBinaryPath, nil
}

// IsFdAvailable checks if fd is available and working
func IsFdAvailable() bool {
	if !IsFdSupported() {
		return false
	}

	fdPath, err := GetFdPath()
	if err != nil {
		return false
	}

	// Test if fd works by running it with --version
	cmd := exec.Command(fdPath, "--version")
	err = cmd.Run()
	return err == nil
}

// CleanupFdBinary removes the extracted fd binary (useful for cleanup)
func CleanupFdBinary() {
	fdMutex.Lock()
	defer fdMutex.Unlock()

	if fdExtracted && fdBinaryPath != "" {
		// Remove the entire temp directory
		tmpDir := filepath.Dir(fdBinaryPath)
		os.RemoveAll(tmpDir)
		fdExtracted = false
		fdBinaryPath = ""
	}
}

// GetEmbeddedFdBinary returns the embedded fd binary for the specified platform/arch
func GetEmbeddedFdBinary(platform, arch string) ([]byte, error) {
	// For now, we only have Linux amd64
	if platform == "linux" && arch == "amd64" {
		return fdLinuxAmd64, nil
	}

	return nil, fmt.Errorf("unsupported platform/architecture: %s/%s", platform, arch)
}
