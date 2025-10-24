// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"io"

	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/utils/multierr"
)

// hasRunCommandCapability checks if the connection supports running commands
func (s *mqlAuditdRules) hasRunCommandCapability() bool {
	conn, ok := s.MqlRuntime.Connection.(shared.Connection)
	if !ok {
		return false
	}
	caps := conn.Capabilities()
	return caps.Has(shared.Capability_RunCommand)
}

// loadRuntimeRules executes `auditctl -l` and parses the output
// Returns error if command execution fails or parsing fails
func (s *mqlAuditdRules) loadRuntimeRules() error {
	s.runtimeLock.Lock()
	defer s.runtimeLock.Unlock()

	if s.runtimeLoaded {
		return s.runtimeError
	}
	s.runtimeLoaded = true

	// Check capability
	if !s.hasRunCommandCapability() {
		s.runtimeError = errors.New("runtime rules require run-command capability (system is not live)")
		return s.runtimeError
	}

	// Execute auditctl -l
	conn, ok := s.MqlRuntime.Connection.(shared.Connection)
	if !ok {
		s.runtimeError = errors.New("failed to get connection for runtime rules")
		return s.runtimeError
	}

	cmd, err := conn.RunCommand("auditctl -l")
	if err != nil {
		s.runtimeError = fmt.Errorf("failed to execute auditctl: %w", err)
		return s.runtimeError
	}

	// Check exit status
	if cmd.ExitStatus != 0 {
		stderr, _ := io.ReadAll(cmd.Stderr)
		if len(stderr) > 0 {
			s.runtimeError = fmt.Errorf("auditctl command failed: %s", string(stderr))
		} else {
			s.runtimeError = fmt.Errorf("auditctl command failed with exit code %d", cmd.ExitStatus)
		}
		return s.runtimeError
	}

	// Read stdout
	stdout, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		s.runtimeError = fmt.Errorf("failed to read auditctl output: %w", err)
		return s.runtimeError
	}

	// Parse the output (reuse existing parser)
	var errors multierr.Errors
	s.parse(string(stdout), &errors)

	// Store the parsed data in runtime storage
	s.runtimeData.controls = s.Controls.Data
	s.runtimeData.files = s.Files.Data
	s.runtimeData.syscalls = s.Syscalls.Data

	// Reset the main storage (we'll merge later based on source)
	s.Controls.Data = nil
	s.Files.Data = nil
	s.Syscalls.Data = nil

	if len(errors.Errors) > 0 {
		s.runtimeError = fmt.Errorf("failed to parse runtime audit rules: %w", errors.Deduplicate())
		return s.runtimeError
	}

	return nil
}

// loadFilesystemRules loads rules from filesystem (existing logic)
func (s *mqlAuditdRules) loadFilesystemRules(path string) error {
	s.filesystemLock.Lock()
	defer s.filesystemLock.Unlock()

	if s.filesystemLoaded {
		return s.filesystemError
	}
	s.filesystemLoaded = true

	if path == "" {
		s.filesystemError = errors.New("the path must be non-empty to parse auditd rules")
		return s.filesystemError
	}

	files, err := getSortedPathFiles(s.MqlRuntime, path)
	if err != nil {
		s.filesystemError = err
		return s.filesystemError
	}

	var parseErrors multierr.Errors
	for i := range files {
		file := files[i].(*mqlFile)

		bn := file.GetBasename()
		if !matchesExtension(bn.Data, ".rules") {
			continue
		}

		content := file.GetContent()
		if content.Error != nil {
			s.filesystemError = content.Error
			return s.filesystemError
		}

		s.parse(content.Data, &parseErrors)
	}

	// Store the parsed data in filesystem storage
	s.filesystemData.controls = s.Controls.Data
	s.filesystemData.files = s.Files.Data
	s.filesystemData.syscalls = s.Syscalls.Data

	// Reset the main storage (we'll merge later based on source)
	s.Controls.Data = nil
	s.Files.Data = nil
	s.Syscalls.Data = nil

	s.filesystemError = parseErrors.Deduplicate()
	return s.filesystemError
}

// matchesExtension checks if filename ends with extension
func matchesExtension(filename, ext string) bool {
	if len(filename) < len(ext) {
		return false
	}
	return filename[len(filename)-len(ext):] == ext
}
