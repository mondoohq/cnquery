// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package binaries

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
)

// RemoteFdChecker handles fd availability checking for remote connections
type RemoteFdChecker struct {
	conn      shared.Connection
	platform  string
	arch      string
	checked   bool
	available bool
}

// NewRemoteFdChecker creates a new remote fd checker
func NewRemoteFdChecker(conn shared.Connection) *RemoteFdChecker {
	return &RemoteFdChecker{
		conn: conn,
	}
}

// IsRemoteFdAvailable checks if we can use fd on the remote system (via embedded binary upload)
func (r *RemoteFdChecker) IsRemoteFdAvailable() bool {
	if r.checked {
		return r.available
	}

	r.checked = true

	// We can use fd if we can detect the platform and have an embedded binary for it
	platform, arch, err := r.GetRemotePlatform()
	if err != nil {
		r.available = false
		return false
	}

	// Check if we have an embedded binary for this platform/arch
	_, err = GetEmbeddedFdBinary(platform, arch)
	r.available = err == nil
	return r.available
}

// GetRemotePlatform detects the remote system's platform and architecture
func (r *RemoteFdChecker) GetRemotePlatform() (string, string, error) {
	if r.platform != "" && r.arch != "" {
		return r.platform, r.arch, nil
	}

	if !r.conn.Capabilities().Has(shared.Capability_RunCommand) {
		return "", "", fmt.Errorf("remote command execution not supported")
	}

	// Detect OS
	osCmd, err := r.conn.RunCommand("uname -s")
	if err != nil {
		return "", "", err
	}

	osOut := strings.TrimSpace(osCmd.Stdout.(*bytes.Buffer).String())
	switch strings.ToLower(osOut) {
	case "linux":
		r.platform = "linux"
	case "darwin":
		r.platform = "darwin"
	case "freebsd":
		r.platform = "freebsd"
	default:
		return "", "", fmt.Errorf("unsupported remote platform: %s", osOut)
	}

	// Detect architecture
	archCmd, err := r.conn.RunCommand("uname -m")
	if err != nil {
		return "", "", err
	}

	archOut := strings.TrimSpace(archCmd.Stdout.(*bytes.Buffer).String())
	switch archOut {
	case "x86_64":
		r.arch = "amd64"
	case "aarch64", "arm64":
		r.arch = "arm64"
	case "armv7l":
		r.arch = "arm"
	default:
		return "", "", fmt.Errorf("unsupported remote architecture: %s", archOut)
	}

	return r.platform, r.arch, nil
}

// ExecuteRemoteFdCommand uploads and executes the embedded fd binary on the remote system
func ExecuteRemoteFdCommand(conn shared.Connection, from string, compiledRegexp *regexp.Regexp, fileType string, permissions int64, name string, depth int64) ([]string, error) {
	if !conn.Capabilities().Has(shared.Capability_RunCommand) {
		return nil, fmt.Errorf("remote command execution not supported")
	}

	if !conn.Capabilities().Has(shared.Capability_File) {
		return nil, fmt.Errorf("remote file operations not supported")
	}

	// Detect remote platform and get appropriate binary
	checker := NewRemoteFdChecker(conn)
	platform, arch, err := checker.GetRemotePlatform()
	if err != nil {
		return nil, fmt.Errorf("failed to detect remote platform: %w", err)
	}

	// Get the appropriate embedded binary
	binaryData, err := GetEmbeddedFdBinary(platform, arch)
	if err != nil {
		return nil, fmt.Errorf("no embedded fd binary available for %s/%s: %w", platform, arch, err)
	}

	// Upload binary to temporary location
	tmpPath := fmt.Sprintf("/tmp/mondoo-fd-%s-%s", platform, arch)
	err = uploadBinaryToRemote(conn, binaryData, tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to upload fd binary: %w", err)
	}

	// Ensure cleanup happens
	defer func() {
		cleanupCmd := fmt.Sprintf("rm -f '%s'", tmpPath)
		conn.RunCommand(cleanupCmd) // Ignore errors in cleanup
	}()

	// Build fd command arguments
	var args []string
	args = append(args, tmpPath) // Use uploaded binary path

	// Add pattern - fd uses the pattern as the first argument
	if compiledRegexp != nil {
		args = append(args, compiledRegexp.String())
	} else if name != "" {
		args = append(args, name)
	} else {
		args = append(args, ".") // Match everything if no pattern specified
	}

	// Add search directory (quoted for safety)
	args = append(args, fmt.Sprintf("'%s'", from))

	// Add file type filter
	if fileType != "" {
		switch fileType {
		case "file":
			args = append(args, "--type", "f")
		case "directory":
			args = append(args, "--type", "d")
		case "link":
			args = append(args, "--type", "l")
		case "socket":
			args = append(args, "--type", "s")
			// fd doesn't support character/block device types directly
		}
	}

	// Add depth limit
	if depth > 0 {
		args = append(args, "--max-depth", fmt.Sprintf("%d", depth))
	}

	// Add absolute paths for consistency
	args = append(args, "--absolute-path")

	// Build and execute the command string
	cmdStr := strings.Join(args, " ")

	cmd, err := conn.RunCommand(cmdStr)
	if err != nil {
		return nil, fmt.Errorf("failed to run remote fd command: %w", err)
	}

	if cmd.ExitStatus != 0 {
		return nil, fmt.Errorf("remote fd command failed with exit status %d", cmd.ExitStatus)
	}

	lines := strings.TrimSpace(cmd.Stdout.(*bytes.Buffer).String())
	if lines == "" {
		return []string{}, nil
	}

	foundFiles := strings.Split(lines, "\n")

	// Filter by permissions if specified (fd doesn't support permission filtering directly)
	if permissions != 0o777 {
		return filterFilesByPermissionsRemote(conn, foundFiles, permissions)
	}

	return foundFiles, nil
}

// uploadBinaryToRemote uploads a binary to the remote system and makes it executable
func uploadBinaryToRemote(conn shared.Connection, binaryData []byte, remotePath string) error {
	// Get the filesystem interface
	fs := conn.FileSystem()
	if fs == nil {
		return fmt.Errorf("no filesystem available for remote connection")
	}

	// Create the directory if needed
	dir := filepath.Dir(remotePath)
	err := fs.MkdirAll(dir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write the binary file
	err = afero.WriteFile(fs, remotePath, binaryData, 0o755)
	if err != nil {
		return fmt.Errorf("failed to write binary file %s: %w", remotePath, err)
	}

	// Make sure it's executable (some filesystems might not preserve permissions)
	chmodCmd := fmt.Sprintf("chmod +x '%s'", remotePath)
	_, err = conn.RunCommand(chmodCmd)
	if err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	return nil
}

// filterFilesByPermissionsRemote filters files by checking their permissions remotely
func filterFilesByPermissionsRemote(conn shared.Connection, files []string, expectedPerms int64) ([]string, error) {
	// For now, return all files since permission filtering with fd is complex
	// In a full implementation, we could run additional commands to check permissions
	return files, nil
}
