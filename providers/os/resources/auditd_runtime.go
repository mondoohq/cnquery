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

	// Initialize empty slices for parsing
	s.runtimeData.controls = make([]interface{}, 0)
	s.runtimeData.files = make([]interface{}, 0)
	s.runtimeData.syscalls = make([]interface{}, 0)

	// Parse the output directly into runtime storage
	var errors multierr.Errors
	parseIntoSlices(s, string(stdout),
		&s.runtimeData.controls,
		&s.runtimeData.files,
		&s.runtimeData.syscalls,
		&errors)

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

	// Initialize empty slices for parsing
	s.filesystemData.controls = make([]interface{}, 0)
	s.filesystemData.files = make([]interface{}, 0)
	s.filesystemData.syscalls = make([]interface{}, 0)

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

		// Parse directly into filesystem storage
		parseIntoSlices(s, content.Data,
			&s.filesystemData.controls,
			&s.filesystemData.files,
			&s.filesystemData.syscalls,
			&parseErrors)
	}

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

// parseIntoSlices parses audit rule content into provided slices
// This is a wrapper around the existing parse() method that directs output to custom slices
func parseIntoSlices(s *mqlAuditdRules, content string, controls, files, syscalls *[]interface{}, errors *multierr.Errors) {
	// Temporarily swap the TValue fields with our target slices
	oldControls := s.Controls.Data
	oldFiles := s.Files.Data
	oldSyscalls := s.Syscalls.Data

	s.Controls.Data = *controls
	s.Files.Data = *files
	s.Syscalls.Data = *syscalls

	// Call the existing parse method
	s.parse(content, errors)

	// Extract the results
	*controls = s.Controls.Data
	*files = s.Files.Data
	*syscalls = s.Syscalls.Data

	// Restore original values
	s.Controls.Data = oldControls
	s.Files.Data = oldFiles
	s.Syscalls.Data = oldSyscalls
}
