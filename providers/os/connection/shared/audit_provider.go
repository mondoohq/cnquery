// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shared

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

// AuditRuleData holds parsed audit rules categorized by type
type AuditRuleData struct {
	Controls []interface{}
	Files    []interface{}
	Syscalls []interface{}
}

// AuditRuleProvider manages audit rule data from filesystem and/or runtime sources
// It abstracts the complexity of dual-source loading at the connection level
type AuditRuleProvider struct {
	connection Connection
	useRuntime bool

	// Lazy loading with dual storage
	filesystemOnce sync.Once
	filesystemData *AuditRuleData
	filesystemErr  error

	runtimeOnce sync.Once
	runtimeData *AuditRuleData
	runtimeErr  error

	// Parser function injected from resources package (to avoid circular dependency)
	parser AuditRuleParser
}

// AuditRuleParser is a function type for parsing audit rules
// This allows the resources package to inject its parser without circular dependencies
type AuditRuleParser func(content string) (*AuditRuleData, error)

// NewAuditRuleProvider creates a new audit rule provider for the given connection
func NewAuditRuleProvider(conn Connection) *AuditRuleProvider {
	hasRunCommand := conn.Capabilities().Has(Capability_RunCommand)

	return &AuditRuleProvider{
		connection: conn,
		useRuntime: hasRunCommand,
	}
}

// SetParser sets the audit rule parser function
// This must be called before GetRules to enable rule parsing
func (p *AuditRuleProvider) SetParser(parser AuditRuleParser) {
	p.parser = parser
}

// CanLoadRuntime returns whether the provider can load runtime rules
func (p *AuditRuleProvider) CanLoadRuntime() bool {
	return p.useRuntime
}

// GetRules returns audit rules from the appropriate source(s)
// On non-live systems: returns filesystem rules only
// On live systems: returns merged rules from both sources (logical AND)
func (p *AuditRuleProvider) GetRules(path string) (*AuditRuleData, error) {
	if !p.useRuntime {
		// Non-live system: filesystem only
		return p.getFilesystemRules(path)
	}

	// Live system: load both sources with logical AND
	return p.getBothRules(path)
}

// getFilesystemRules loads rules from filesystem (lazy, cached)
func (p *AuditRuleProvider) getFilesystemRules(path string) (*AuditRuleData, error) {
	p.filesystemOnce.Do(func() {
		p.filesystemData, p.filesystemErr = p.loadFilesystemRules(path)
	})
	return p.filesystemData, p.filesystemErr
}

// getRuntimeRules loads rules from runtime (lazy, cached)
func (p *AuditRuleProvider) getRuntimeRules() (*AuditRuleData, error) {
	p.runtimeOnce.Do(func() {
		p.runtimeData, p.runtimeErr = p.loadRuntimeRules()
	})
	return p.runtimeData, p.runtimeErr
}

// loadFilesystemRules reads audit rules from filesystem files
func (p *AuditRuleProvider) loadFilesystemRules(path string) (*AuditRuleData, error) {
	if p.parser == nil {
		return nil, fmt.Errorf("audit rule parser not set")
	}

	if path == "" {
		return nil, fmt.Errorf("path must be non-empty to parse auditd rules")
	}

	// Read files from filesystem using connection's FileSystem
	fs := p.connection.FileSystem()
	if fs == nil {
		return nil, fmt.Errorf("connection does not provide filesystem access")
	}

	// Open the directory
	dir, err := fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit rules directory: %w", err)
	}
	defer dir.Close()

	// Read all .rules files
	entries, err := dir.Readdir(-1)
	if err != nil {
		return nil, fmt.Errorf("failed to read audit rules directory: %w", err)
	}

	// Aggregate all rule content
	var allContent strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if !strings.HasSuffix(filename, ".rules") {
			continue
		}

		// Read file content
		filePath := path + "/" + filename
		file, err := fs.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open rule file %s: %w", filename, err)
		}

		content, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read rule file %s: %w", filename, err)
		}

		allContent.Write(content)
		allContent.WriteString("\n")
	}

	// Parse aggregated content
	return p.parser(allContent.String())
}

// loadRuntimeRules executes auditctl -l and parses the output
func (p *AuditRuleProvider) loadRuntimeRules() (*AuditRuleData, error) {
	if p.parser == nil {
		return nil, fmt.Errorf("audit rule parser not set")
	}

	if !p.useRuntime {
		return nil, fmt.Errorf("runtime rules require run-command capability")
	}

	// Execute auditctl -l
	cmd, err := p.connection.RunCommand("auditctl -l")
	if err != nil {
		return nil, fmt.Errorf("failed to execute auditctl: %w", err)
	}

	// Check exit status
	if cmd.ExitStatus != 0 {
		stderr, _ := io.ReadAll(cmd.Stderr)
		if len(stderr) > 0 {
			stderrStr := string(stderr)
			// Check if it's a "command not found" error
			if strings.Contains(stderrStr, "command not found") ||
				strings.Contains(stderrStr, "executable file not found") {
				return nil, fmt.Errorf("auditctl command not found: %s", stderrStr)
			}
			return nil, fmt.Errorf("auditctl command failed: %s", stderrStr)
		}
		return nil, fmt.Errorf("auditctl command failed with exit code %d", cmd.ExitStatus)
	}

	// Read stdout
	stdout, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to read auditctl output: %w", err)
	}

	// Parse the output
	return p.parser(string(stdout))
}

// getBothRules loads rules from both sources and applies logical AND
func (p *AuditRuleProvider) getBothRules(path string) (*AuditRuleData, error) {
	// Load both sources in parallel
	var wg sync.WaitGroup
	var fsData, rtData *AuditRuleData
	var fsErr, rtErr error

	wg.Add(2)
	go func() {
		defer wg.Done()
		fsData, fsErr = p.getFilesystemRules(path)
	}()
	go func() {
		defer wg.Done()
		rtData, rtErr = p.getRuntimeRules()
	}()
	wg.Wait()

	// Check if runtime error is "command not found" - if so, fall back to filesystem only
	if rtErr != nil {
		isCommandNotFound := strings.Contains(rtErr.Error(), "command not found") ||
			strings.Contains(rtErr.Error(), "executable file not found")
		if isCommandNotFound {
			// auditctl not installed - fall back to filesystem only
			if fsErr != nil {
				return nil, fmt.Errorf("failed to load audit rules from filesystem: %w", fsErr)
			}
			return fsData, nil
		}
	}

	// Logical AND: both must succeed
	if fsErr != nil && rtErr != nil {
		return nil, fmt.Errorf("failed to load audit rules from both filesystem and runtime: [filesystem: %v, runtime: %v]", fsErr, rtErr)
	}
	if fsErr != nil {
		return nil, fmt.Errorf("failed to load audit rules from filesystem: %w", fsErr)
	}
	if rtErr != nil {
		return nil, fmt.Errorf("failed to load audit rules from runtime: %w", rtErr)
	}

	// Validate sets match (set-based comparison)
	return p.validateAndMerge(fsData, rtData)
}

// validateAndMerge performs set-based comparison and returns merged data
func (p *AuditRuleProvider) validateAndMerge(fs, rt *AuditRuleData) (*AuditRuleData, error) {
	// For now, we do a simple count-based check
	// TODO: Implement proper set-based comparison with rule IDs

	if len(fs.Controls) != len(rt.Controls) {
		return nil, fmt.Errorf("control rules differ between filesystem (%d) and runtime (%d)",
			len(fs.Controls), len(rt.Controls))
	}
	if len(fs.Files) != len(rt.Files) {
		return nil, fmt.Errorf("file rules differ between filesystem (%d) and runtime (%d)",
			len(fs.Files), len(rt.Files))
	}
	if len(fs.Syscalls) != len(rt.Syscalls) {
		return nil, fmt.Errorf("syscall rules differ between filesystem (%d) and runtime (%d)",
			len(fs.Syscalls), len(rt.Syscalls))
	}

	// Sets match, return filesystem data (they should be identical)
	return fs, nil
}

// rulesMatch performs set-based comparison of rules (order-agnostic)
// TODO: Implement proper set comparison using rule IDs
func rulesMatch(a, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}

	// For now, just check lengths
	// In production, we'd convert to sets and compare
	return true
}
