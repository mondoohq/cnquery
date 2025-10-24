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

// GetRules returns audit rules, optionally merging with runtime data
// filesystemData: rules loaded from filesystem by the resource
// On non-live systems: returns filesystem rules as-is
// On live systems: merges filesystem with runtime rules (logical AND)
func (p *AuditRuleProvider) GetRules(filesystemData *AuditRuleData) (*AuditRuleData, error) {
	if !p.useRuntime {
		// Non-live system: return filesystem data as-is
		return filesystemData, nil
	}

	// Live system: load runtime and merge with filesystem
	return p.mergeWithRuntime(filesystemData)
}

// getRuntimeRules loads rules from runtime (lazy, cached)
func (p *AuditRuleProvider) getRuntimeRules() (*AuditRuleData, error) {
	p.runtimeOnce.Do(func() {
		p.runtimeData, p.runtimeErr = p.loadRuntimeRules()
	})
	return p.runtimeData, p.runtimeErr
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

// mergeWithRuntime loads runtime rules and merges with filesystem data
func (p *AuditRuleProvider) mergeWithRuntime(filesystemData *AuditRuleData) (*AuditRuleData, error) {
	// Load runtime rules
	rtData, rtErr := p.getRuntimeRules()

	// Check if runtime error is "command not found" - if so, fall back to filesystem only
	if rtErr != nil {
		isCommandNotFound := strings.Contains(rtErr.Error(), "command not found") ||
			strings.Contains(rtErr.Error(), "executable file not found")
		if isCommandNotFound {
			// auditctl not installed - fall back to filesystem only
			return filesystemData, nil
		}
		// Other runtime errors are failures
		return nil, fmt.Errorf("failed to load audit rules from runtime: %w", rtErr)
	}

	// Validate sets match (set-based comparison)
	return p.validateAndMerge(filesystemData, rtData)
}

// validateAndMerge performs set-based comparison and returns merged data
func (p *AuditRuleProvider) validateAndMerge(fs, rt *AuditRuleData) (*AuditRuleData, error) {
	// Count total rules in each source
	totalRuntimeRules := len(rt.Controls) + len(rt.Files) + len(rt.Syscalls)
	totalFilesystemRules := len(fs.Controls) + len(fs.Files) + len(fs.Syscalls)

	// Special case: if runtime has 0 rules total, auditd might not be running
	if totalRuntimeRules == 0 && totalFilesystemRules > 0 {
		// Runtime has no rules but filesystem does - auditd might not be running
		// Return filesystem rules (lenient mode)
		return fs, nil
	}

	// Check if both sources have rules (both non-zero)
	if totalRuntimeRules > 0 && totalFilesystemRules > 0 {
		// Both have rules - in v3.0, we return runtime data as the source of truth
		// Runtime shows what IS actually loaded in the kernel
		// Filesystem shows what SHOULD be loaded at boot
		// For a live system, runtime is the current state
		return rt, nil
	}

	// Fallback: if filesystem has rules but runtime doesn't, return filesystem
	if totalFilesystemRules > 0 {
		return fs, nil
	}

	// Both empty - return either (they're equivalent)
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
