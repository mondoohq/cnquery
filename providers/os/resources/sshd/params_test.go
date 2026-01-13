// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sshd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSHParser(t *testing.T) {
	raw, err := os.ReadFile("./testdata/sshd_config")
	require.NoError(t, err)

	sshParams, err := ParseBlocks("./testdata/sshd_config", string(raw))
	if err != nil {
		t.Fatalf("cannot request file %v", err)
	}

	assert.NotNil(t, sshParams, "params are not nil")

	// check result for multiple host-keys
	assert.Equal(t, "/etc/ssh/ssh_host_rsa_key,/etc/ssh/ssh_host_ecdsa_key,/etc/ssh/ssh_host_ed25519_key", sshParams[0].Params["HostKey"])
	assert.Equal(t, "yes", sshParams[0].Params["X11Forwarding"])
	assert.Equal(t, "60", sshParams[0].Params["LoginGraceTime"])
}

func TestSSHParseCaseInsensitive(t *testing.T) {
	raw, err := os.ReadFile("./testdata/case_insensitive")
	require.NoError(t, err)

	sshParams, err := ParseBlocks("./testdata/case_insensitive", string(raw))
	if err != nil {
		t.Fatalf("cannot request file %v", err)
	}

	assert.NotNil(t, sshParams, "params are not nil")

	assert.Equal(t, "22", sshParams[0].Params["Port"])
	assert.Equal(t, "any", sshParams[0].Params["AddressFamily"])
	assert.Equal(t, "0.0.0.0", sshParams[0].Params["ListenAddress"])
}

func TestSSHParseWithGlob(t *testing.T) {
	// Test that glob patterns in Include directives are expanded correctly
	// and that each file maintains its own context

	fileContent := func(path string) (string, error) {
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}

	globExpand := func(glob string) ([]string, error) {
		// For this test we handle the known patterns explicitly.
		var paths []string
		switch glob {
		case "conf.d/*.conf":
			paths = []string{
				"./testdata/conf.d/01_mondoo.conf",
				"./testdata/conf.d/02_security.conf",
			}
		case "subdir/01_*.conf":
			paths = []string{"./testdata/subdir/01_custom.conf"}
		case "subdir/02_*.conf":
			paths = []string{"./testdata/subdir/02_additional.conf"}
		case "./testdata/sshd_config_with_include":
			paths = []string{"./testdata/sshd_config_with_include"}
		default:
			if _, err := os.Stat(glob); err == nil {
				paths = []string{glob}
			}
		}
		return paths, nil
	}

	blocks, err := ParseBlocksWithGlob("./testdata/sshd_config_with_include", fileContent, globExpand)
	require.NoError(t, err)
	assert.NotNil(t, blocks)

	// Verify that we have blocks from multiple files
	// The main file should have blocks, and included files should also contribute
	assert.Greater(t, len(blocks), 0, "should have at least one block")

	// Check that params from included files are present
	// Find the default block (empty criteria)
	var defaultBlock *MatchBlock
	for _, block := range blocks {
		if block.Criteria == "" {
			defaultBlock = block
			break
		}
	}
	require.NotNil(t, defaultBlock, "should have a default block")

	// Verify params from main file
	assert.Equal(t, "22", defaultBlock.Params["Port"])
	assert.Equal(t, "any", defaultBlock.Params["AddressFamily"])
	assert.Equal(t, "yes", defaultBlock.Params["UsePAM"])
	assert.Equal(t, "yes", defaultBlock.Params["X11Forwarding"])

	// Verify params from included files (conf.d/*.conf)
	assert.Equal(t, "no", defaultBlock.Params["PermitRootLogin"])
	assert.Equal(t, "yes", defaultBlock.Params["PasswordAuthentication"])
	assert.Equal(t, "3", defaultBlock.Params["MaxAuthTries"])
	assert.Equal(t, "30", defaultBlock.Params["LoginGraceTime"])

	// Verify params from subdirectory files
	assert.Contains(t, defaultBlock.Params["Ciphers"], "aes256-gcm@openssh.com")
	assert.Contains(t, defaultBlock.Params["MACs"], "hmac-sha2-256-etm@openssh.com")

	// Verify that blocks have correct file paths in their context
	// Check that we have blocks from different files
	filePaths := make(map[string]bool)
	for _, block := range blocks {
		filePaths[block.Context.Path] = true
	}

	// Should have blocks from main file and included files
	assert.Greater(t, len(filePaths), 1, "should have blocks from multiple files")

	// Verify Match blocks from included files
	var sftpBlock *MatchBlock
	var adminBlock *MatchBlock
	for _, block := range blocks {
		if block.Criteria == "Group sftp-users" {
			sftpBlock = block
		}
		if block.Criteria == "User admin" {
			adminBlock = block
		}
	}

	require.NotNil(t, sftpBlock, "should have sftp-users match block")
	assert.Equal(t, "no", sftpBlock.Params["AllowTcpForwarding"])
	assert.Contains(t, sftpBlock.Context.Path, "01_mondoo.conf")

	require.NotNil(t, adminBlock, "should have admin user match block")
	assert.Equal(t, "yes", adminBlock.Params["PermitRootLogin"])
	assert.Equal(t, "no", adminBlock.Params["PasswordAuthentication"])
	assert.Contains(t, adminBlock.Context.Path, "02_security.conf")
}

func TestSSHParseIncludeInsideMatchBlock(t *testing.T) {
	// Test edge case: Include directive inside a Match block
	// The included file has a different Match block - both should be present
	// This verifies that Match blocks from included files are always added
	// to the global map, regardless of where the Include directive appears.

	fileContent := func(path string) (string, error) {
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}

	globExpand := func(glob string) ([]string, error) {
		// Handle the nested include file
		if glob == "match_include_nested.conf" {
			return []string{"./testdata/match_include_nested.conf"}, nil
		}
		// For other paths, check if file exists
		if _, err := os.Stat(glob); err == nil {
			return []string{glob}, nil
		}
		return []string{glob}, nil
	}

	mainConfigPath := "./testdata/match_include_main.conf"
	blocks, err := ParseBlocksWithGlob(mainConfigPath, fileContent, globExpand)
	require.NoError(t, err)
	assert.NotNil(t, blocks)

	// Find both Match blocks
	var specialBlock *MatchBlock
	var adminBlock *MatchBlock
	var defaultBlock *MatchBlock

	for _, block := range blocks {
		switch block.Criteria {
		case "Group special":
			specialBlock = block
		case "User admin":
			adminBlock = block
		case "":
			defaultBlock = block
		}
	}

	// Verify default block exists
	require.NotNil(t, defaultBlock, "should have default block")
	assert.Equal(t, "22", defaultBlock.Params["Port"])

	// Verify the Match block containing the Include exists
	require.NotNil(t, specialBlock, "should have 'Group special' match block")
	assert.Equal(t, "30", specialBlock.Params["ClientAliveInterval"])

	// Verify the Match block from the included file exists (this is the edge case)
	// Before the fix, this block would be filtered out because Include was inside
	// a Match block with different criteria.
	require.NotNil(t, adminBlock, "should have 'User admin' match block from included file")
	assert.Equal(t, "yes", adminBlock.Params["PermitRootLogin"])
	assert.Equal(t, "no", adminBlock.Params["PasswordAuthentication"])
	assert.Contains(t, adminBlock.Context.Path, "match_include_nested.conf")
}
