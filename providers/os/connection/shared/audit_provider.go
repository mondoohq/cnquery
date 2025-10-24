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

// validateAndMerge performs STRICT set-based comparison (v4.0)
// Both sources must match exactly - any divergence is a FAILED state
func (p *AuditRuleProvider) validateAndMerge(fs, rt *AuditRuleData) (*AuditRuleData, error) {
	// STRICT VALIDATION: Compare actual rule content as sets (not just counts)

	// Check controls match (set-based comparison)
	if !rulesMatchAsSet(fs.Controls, rt.Controls) {
		return nil, fmt.Errorf("control rules differ between filesystem and runtime (configuration drift detected)")
	}

	// Check files match (set-based comparison)
	if !rulesMatchAsSet(fs.Files, rt.Files) {
		return nil, fmt.Errorf("file rules differ between filesystem and runtime (configuration drift detected)")
	}

	// Check syscalls match (set-based comparison)
	if !rulesMatchAsSet(fs.Syscalls, rt.Syscalls) {
		return nil, fmt.Errorf("syscall rules differ between filesystem and runtime (configuration drift detected)")
	}

	// Both sources match - return runtime data as it's the current state
	// (Runtime is the authoritative source when both match)
	return rt, nil
}

// rulesMatchAsSet performs set-based comparison of rules (order-agnostic)
// Compares actual rule content, not just counts
func rulesMatchAsSet(a, b []interface{}) bool {
	// Length must match first
	if len(a) != len(b) {
		return false
	}

	// Both empty is a match
	if len(a) == 0 {
		return true
	}

	// Convert to sets and compare
	setA := makeRuleSet(a)
	setB := makeRuleSet(b)

	// Check all elements in A exist in B (and vice versa due to length check)
	for rule := range setA {
		if !setB[rule] {
			return false
		}
	}

	return true
}

// makeRuleSet converts a slice of rules into a set (map) for comparison
// We convert each rule to a string representation for set membership testing
func makeRuleSet(rules []interface{}) map[string]bool {
	set := make(map[string]bool, len(rules))
	for _, rule := range rules {
		// Convert rule to string representation
		// This works because our rules are either strings or have string representations
		ruleStr := fmt.Sprintf("%v", rule)
		set[ruleStr] = true
	}
	return set
}
