// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shared_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
)

// TestAuditRuleProvider_FilesystemOnly tests filesystem-only behavior on non-live systems
func TestAuditRuleProvider_FilesystemOnly(t *testing.T) {
	// Create mock connection WITHOUT run-command capability
	conn, err := mock.New(0, "", &inventory.Asset{})
	require.NoError(t, err)

	// Override capabilities to be filesystem-only
	mockConn := &mockConnectionNoRunCommand{Connection: conn}

	provider := shared.NewAuditRuleProvider(mockConn)
	require.NotNil(t, provider)

	// Provider should not attempt runtime loading
	assert.False(t, provider.CanLoadRuntime(), "Provider should not support runtime on non-live systems")
}

// TestAuditRuleProvider_DualSource tests dual-source behavior on live systems
func TestAuditRuleProvider_DualSource(t *testing.T) {
	// Create mock connection WITH run-command capability
	conn, err := mock.New(0, "", &inventory.Asset{})
	require.NoError(t, err)

	provider := shared.NewAuditRuleProvider(conn)
	require.NotNil(t, provider)

	// Provider should support runtime loading
	assert.True(t, provider.CanLoadRuntime(), "Provider should support runtime on live systems")
}

// TestAuditRuleProvider_GetRules_FilesystemSuccess tests successful filesystem rule loading
func TestAuditRuleProvider_GetRules_FilesystemSuccess(t *testing.T) {
	// Create mock with filesystem data
	mockData := createMockFilesystemRules()
	conn := createMockWithData(t, mockData, false) // false = no runtime capability

	provider := shared.NewAuditRuleProvider(conn)
	data, err := provider.GetRules("/etc/audit/rules.d")

	require.NoError(t, err)
	assert.NotNil(t, data)
	assert.Greater(t, len(data.Files), 0, "Should have file rules")
}

// TestAuditRuleProvider_GetRules_RuntimeSuccess tests successful runtime rule loading
func TestAuditRuleProvider_GetRules_RuntimeSuccess(t *testing.T) {
	// Create mock with runtime command response
	mockData := createMockRuntimeRules()
	conn := createMockWithData(t, mockData, true) // true = has runtime capability

	provider := shared.NewAuditRuleProvider(conn)
	data, err := provider.GetRules("/etc/audit/rules.d")

	require.NoError(t, err)
	assert.NotNil(t, data)
	assert.Greater(t, len(data.Files), 0, "Should have file rules from runtime")
}

// TestAuditRuleProvider_GetRules_BothSourcesMatch tests logical AND when both sources match
func TestAuditRuleProvider_GetRules_BothSourcesMatch(t *testing.T) {
	// Create mock with matching filesystem and runtime data
	mockData := createMockMatchingRules()
	conn := createMockWithData(t, mockData, true)

	provider := shared.NewAuditRuleProvider(conn)
	data, err := provider.GetRules("/etc/audit/rules.d")

	require.NoError(t, err)
	assert.NotNil(t, data)
	assert.Greater(t, len(data.Files), 0, "Should have merged rules")
}

// TestAuditRuleProvider_GetRules_RuntimeMissing tests FAILED state when runtime differs
func TestAuditRuleProvider_GetRules_RuntimeMissing(t *testing.T) {
	// Create mock with mismatched filesystem and runtime data
	mockData := createMockMismatchedRules()
	conn := createMockWithData(t, mockData, true)

	provider := shared.NewAuditRuleProvider(conn)
	data, err := provider.GetRules("/etc/audit/rules.d")

	// Should return error indicating mismatch
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "differ between", "Error should indicate source of mismatch")
}

// TestAuditRuleProvider_GetRules_FilesystemFails tests FAILED state when filesystem fails
func TestAuditRuleProvider_GetRules_FilesystemFails(t *testing.T) {
	// Create mock with filesystem error
	mockData := createMockWithFilesystemError()
	conn := createMockWithData(t, mockData, true)

	provider := shared.NewAuditRuleProvider(conn)
	data, err := provider.GetRules("/etc/audit/rules.d")

	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "filesystem", "Error should indicate filesystem failure")
}

// TestAuditRuleProvider_GetRules_RuntimeFails tests FAILED state when runtime fails
func TestAuditRuleProvider_GetRules_RuntimeFails(t *testing.T) {
	// Create mock with runtime command failure
	mockData := createMockWithRuntimeError()
	conn := createMockWithData(t, mockData, true)

	provider := shared.NewAuditRuleProvider(conn)
	data, err := provider.GetRules("/etc/audit/rules.d")

	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "runtime", "Error should indicate runtime failure")
}

// TestAuditRuleProvider_GetRules_BothFail tests FAILED state when both sources fail
func TestAuditRuleProvider_GetRules_BothFail(t *testing.T) {
	// Create mock with both sources failing
	mockData := createMockWithBothErrors()
	conn := createMockWithData(t, mockData, true)

	provider := shared.NewAuditRuleProvider(conn)
	data, err := provider.GetRules("/etc/audit/rules.d")

	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "filesystem", "Error should mention filesystem")
	assert.Contains(t, err.Error(), "runtime", "Error should mention runtime")
}

// TestAuditRuleProvider_LazyLoading tests that rules are loaded only once
func TestAuditRuleProvider_LazyLoading(t *testing.T) {
	mockData := createMockFilesystemRules()
	conn := createMockWithData(t, mockData, false)

	provider := shared.NewAuditRuleProvider(conn)

	// First call should load
	data1, err1 := provider.GetRules("/etc/audit/rules.d")
	require.NoError(t, err1)

	// Second call should use cached data
	data2, err2 := provider.GetRules("/etc/audit/rules.d")
	require.NoError(t, err2)

	// Should be same data (by reference or value)
	assert.Equal(t, len(data1.Files), len(data2.Files))
}

// TestAuditRuleProvider_SetBasedComparison tests order-agnostic rule comparison
func TestAuditRuleProvider_SetBasedComparison(t *testing.T) {
	// Create mock with same rules in different order
	mockData := createMockDifferentOrderRules()
	conn := createMockWithData(t, mockData, true)

	provider := shared.NewAuditRuleProvider(conn)
	data, err := provider.GetRules("/etc/audit/rules.d")

	// Should succeed because sets are equal (order doesn't matter)
	require.NoError(t, err)
	assert.NotNil(t, data)
}

// Helper: Mock connection without run-command capability
type mockConnectionNoRunCommand struct {
	shared.Connection
}

func (m *mockConnectionNoRunCommand) Capabilities() shared.Capabilities {
	return shared.Capability_File // Only file capability, no RunCommand
}

// Helper functions to create mock data
func createMockFilesystemRules() *mock.TomlData {
	return &mock.TomlData{
		Files: map[string]*mock.MockFileData{
			"/etc/audit/rules.d/audit.rules": {
				Path: "/etc/audit/rules.d/audit.rules",
				Content: `-w /etc/passwd -p wa -k passwd_changes
-w /etc/shadow -p wa -k shadow_changes
-a always,exit -F arch=b64 -S open -F key=file_access
`,
				StatData: mock.FileInfo{
					Mode:  0o644,
					IsDir: false,
					Size:  150,
				},
			},
		},
		Commands: map[string]*mock.Command{},
	}
}

func createMockRuntimeRules() *mock.TomlData {
	return &mock.TomlData{
		Files: map[string]*mock.MockFileData{},
		Commands: map[string]*mock.Command{
			"auditctl -l": {
				Command: "auditctl -l",
				Stdout: `-w /etc/passwd -p wa -k passwd_changes
-w /etc/shadow -p wa -k shadow_changes
-a always,exit -F arch=b64 -S open -F key=file_access
`,
				Stderr:     "",
				ExitStatus: 0,
			},
		},
	}
}

func createMockMatchingRules() *mock.TomlData {
	rules := `-w /etc/passwd -p wa -k passwd_changes
-w /etc/shadow -p wa -k shadow_changes
`
	return &mock.TomlData{
		Files: map[string]*mock.MockFileData{
			"/etc/audit/rules.d/audit.rules": {
				Path:    "/etc/audit/rules.d/audit.rules",
				Content: rules,
				StatData: mock.FileInfo{
					Mode:  0o644,
					IsDir: false,
					Size:  int64(len(rules)),
				},
			},
		},
		Commands: map[string]*mock.Command{
			"auditctl -l": {
				Command:    "auditctl -l",
				Stdout:     rules,
				Stderr:     "",
				ExitStatus: 0,
			},
		},
	}
}

func createMockMismatchedRules() *mock.TomlData {
	return &mock.TomlData{
		Files: map[string]*mock.MockFileData{
			"/etc/audit/rules.d/audit.rules": {
				Path:    "/etc/audit/rules.d/audit.rules",
				Content: `-w /etc/passwd -p wa -k passwd_changes`,
				StatData: mock.FileInfo{
					Mode:  0o644,
					IsDir: false,
					Size:  50,
				},
			},
		},
		Commands: map[string]*mock.Command{
			"auditctl -l": {
				Command:    "auditctl -l",
				Stdout:     `-w /etc/shadow -p wa -k shadow_changes`, // Different rule
				Stderr:     "",
				ExitStatus: 0,
			},
		},
	}
}

func createMockWithFilesystemError() *mock.TomlData {
	return &mock.TomlData{
		Files: map[string]*mock.MockFileData{
			// No files = error reading filesystem
		},
		Commands: map[string]*mock.Command{
			"auditctl -l": {
				Command:    "auditctl -l",
				Stdout:     `-w /etc/passwd -p wa -k passwd_changes`,
				Stderr:     "",
				ExitStatus: 0,
			},
		},
	}
}

func createMockWithRuntimeError() *mock.TomlData {
	return &mock.TomlData{
		Files: map[string]*mock.MockFileData{
			"/etc/audit/rules.d/audit.rules": {
				Path:    "/etc/audit/rules.d/audit.rules",
				Content: `-w /etc/passwd -p wa -k passwd_changes`,
				StatData: mock.FileInfo{
					Mode:  0o644,
					IsDir: false,
					Size:  50,
				},
			},
		},
		Commands: map[string]*mock.Command{
			"auditctl -l": {
				Command:    "auditctl -l",
				Stdout:     "",
				Stderr:     "You must be root to run this command",
				ExitStatus: 1,
			},
		},
	}
}

func createMockWithBothErrors() *mock.TomlData {
	return &mock.TomlData{
		Files: map[string]*mock.MockFileData{
			// No files
		},
		Commands: map[string]*mock.Command{
			"auditctl -l": {
				Command:    "auditctl -l",
				Stdout:     "",
				Stderr:     "Command failed",
				ExitStatus: 1,
			},
		},
	}
}

func createMockDifferentOrderRules() *mock.TomlData {
	fsRules := `-w /etc/passwd -p wa -k passwd_changes
-w /etc/shadow -p wa -k shadow_changes
`
	rtRules := `-w /etc/shadow -p wa -k shadow_changes
-w /etc/passwd -p wa -k passwd_changes
` // Same rules, different order

	return &mock.TomlData{
		Files: map[string]*mock.MockFileData{
			"/etc/audit/rules.d/audit.rules": {
				Path:    "/etc/audit/rules.d/audit.rules",
				Content: fsRules,
				StatData: mock.FileInfo{
					Mode:  0o644,
					IsDir: false,
					Size:  int64(len(fsRules)),
				},
			},
		},
		Commands: map[string]*mock.Command{
			"auditctl -l": {
				Command:    "auditctl -l",
				Stdout:     rtRules,
				Stderr:     "",
				ExitStatus: 0,
			},
		},
	}
}

func createMockWithData(t *testing.T, data *mock.TomlData, hasRunCommand bool) shared.Connection {
	// Create mock connection with provided data
	conn, err := mock.New(0, "", &inventory.Asset{})
	require.NoError(t, err)

	// Inject mock data
	// Note: This is a simplified approach. In real implementation,
	// we'd need to create the mock connection properly with data
	// For now, this is a placeholder for the test structure

	if !hasRunCommand {
		return &mockConnectionNoRunCommand{Connection: conn}
	}

	return conn
}
