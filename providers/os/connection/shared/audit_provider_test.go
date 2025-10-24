// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shared_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
)

// TC-1: Non-Live System - Filesystem Only
func TestAuditRuleProvider_TC1_NonLive_FilesystemOnly(t *testing.T) {
	// Create mock connection WITHOUT run-command capability
	conn, err := mock.New(0, "", &inventory.Asset{})
	require.NoError(t, err)

	// Override capabilities to be filesystem-only
	mockConn := &mockConnectionNoRunCommand{Connection: conn}

	provider := shared.NewAuditRuleProvider(mockConn)
	require.NotNil(t, provider)

	// Provider should not attempt runtime loading
	assert.False(t, provider.CanLoadRuntime(), "Provider should not support runtime on non-live systems")

	// Filesystem data should pass through unchanged
	fsData := &shared.AuditRuleData{
		Controls: []interface{}{"control1"},
		Files:    []interface{}{"file1", "file2"},
		Syscalls: []interface{}{"syscall1"},
	}

	result, err := provider.GetRules(fsData)
	require.NoError(t, err)
	assert.Equal(t, fsData, result, "Should return filesystem data unchanged on non-live")
}

// TC-2: Live System - Perfect Match (same count and content)
func TestAuditRuleProvider_TC2_Live_PerfectMatch(t *testing.T) {
	// Mock data with matching rules
	fsData := &shared.AuditRuleData{
		Controls: []interface{}{"control1", "control2"},
		Files:    []interface{}{"file1", "file2", "file3"},
		Syscalls: []interface{}{},
	}

	rtData := &shared.AuditRuleData{
		Controls: []interface{}{"control1", "control2"},    // Same rules
		Files:    []interface{}{"file1", "file2", "file3"}, // Same rules
		Syscalls: []interface{}{},
	}

	conn := &mockConnectionWithRuntime{
		runtimeData: rtData,
		runtimeErr:  nil,
	}

	provider := shared.NewAuditRuleProvider(conn)
	provider.SetParser(createMockParser(rtData, nil))

	result, err := provider.GetRules(fsData)
	require.NoError(t, err, "Should PASS when both sources match")
	assert.NotNil(t, result)
}

// TC-3: Live System - Drift (Runtime Missing Rules)
func TestAuditRuleProvider_TC3_Live_DriftRuntimeMissing(t *testing.T) {
	// Filesystem has rules, runtime has 0 (configuration not loaded)
	fsData := &shared.AuditRuleData{
		Controls: []interface{}{"control1", "control2", "control3", "control4", "control5"},
		Files:    []interface{}{},
		Syscalls: []interface{}{},
	}

	rtData := &shared.AuditRuleData{
		Controls: []interface{}{}, // Empty - drift!
		Files:    []interface{}{},
		Syscalls: []interface{}{},
	}

	conn := &mockConnectionWithRuntime{
		runtimeData: rtData,
		runtimeErr:  nil,
	}

	provider := shared.NewAuditRuleProvider(conn)
	provider.SetParser(createMockParser(rtData, nil))

	result, err := provider.GetRules(fsData)
	assert.Error(t, err, "Should FAIL when runtime has fewer rules (drift)")
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "differ", "Error should mention drift/differ")
}

// TC-4: Live System - Drift (Extra Runtime Rules)
func TestAuditRuleProvider_TC4_Live_DriftExtraRuntime(t *testing.T) {
	// Filesystem has 10 rules, runtime has 12 (extra rules added)
	fsData := &shared.AuditRuleData{
		Controls: []interface{}{},
		Files:    []interface{}{"f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10"},
		Syscalls: []interface{}{},
	}

	rtData := &shared.AuditRuleData{
		Controls: []interface{}{},
		Files:    []interface{}{"f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12"}, // Extra rules
		Syscalls: []interface{}{},
	}

	conn := &mockConnectionWithRuntime{
		runtimeData: rtData,
		runtimeErr:  nil,
	}

	provider := shared.NewAuditRuleProvider(conn)
	provider.SetParser(createMockParser(rtData, nil))

	result, err := provider.GetRules(fsData)
	assert.Error(t, err, "Should FAIL when runtime has extra rules (drift)")
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "differ", "Error should mention drift/differ")
}

// TC-5: Live System - Runtime Error (Permission Denied)
func TestAuditRuleProvider_TC5_Live_RuntimePermissionDenied(t *testing.T) {
	fsData := &shared.AuditRuleData{
		Controls: []interface{}{"control1"},
		Files:    []interface{}{},
		Syscalls: []interface{}{},
	}

	permErr := fmt.Errorf("auditctl command failed: You must be root to run this command")

	conn := &mockConnectionWithRuntime{
		runtimeData: nil,
		runtimeErr:  permErr,
	}

	provider := shared.NewAuditRuleProvider(conn)
	provider.SetParser(createMockParser(nil, permErr))

	result, err := provider.GetRules(fsData)
	assert.Error(t, err, "Should FAIL when runtime has permission error")
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "runtime", "Error should mention runtime failure")
}

// TC-6: Live System - Runtime Error (Command Not Found) - GRACEFUL FALLBACK
func TestAuditRuleProvider_TC6_Live_RuntimeCommandNotFound(t *testing.T) {
	fsData := &shared.AuditRuleData{
		Controls: []interface{}{"control1"},
		Files:    []interface{}{"file1"},
		Syscalls: []interface{}{},
	}

	cmdNotFoundErr := fmt.Errorf("auditctl command not found: command not found")

	conn := &mockConnectionWithRuntime{
		runtimeData: nil,
		runtimeErr:  cmdNotFoundErr,
	}

	provider := shared.NewAuditRuleProvider(conn)
	provider.SetParser(createMockParser(nil, cmdNotFoundErr))

	result, err := provider.GetRules(fsData)
	require.NoError(t, err, "Should gracefully fallback to filesystem when auditctl not installed")
	assert.Equal(t, fsData, result, "Should return filesystem data when command not found")
}

// TC-7: Set-Based Comparison - Same Content Different Order
func TestAuditRuleProvider_TC7_SetBased_DifferentOrder(t *testing.T) {
	// Filesystem: [A, B, C]
	// Runtime: [C, A, B] (same rules, different order)
	fsData := &shared.AuditRuleData{
		Controls: []interface{}{},
		Files:    []interface{}{"ruleA", "ruleB", "ruleC"},
		Syscalls: []interface{}{},
	}

	rtData := &shared.AuditRuleData{
		Controls: []interface{}{},
		Files:    []interface{}{"ruleC", "ruleA", "ruleB"}, // Different order
		Syscalls: []interface{}{},
	}

	conn := &mockConnectionWithRuntime{
		runtimeData: rtData,
		runtimeErr:  nil,
	}

	provider := shared.NewAuditRuleProvider(conn)
	provider.SetParser(createMockParser(rtData, nil))

	result, err := provider.GetRules(fsData)
	require.NoError(t, err, "Should PASS - order doesn't matter (set semantics)")
	assert.NotNil(t, result)
}

// TC-8: Set-Based Comparison - Different Content Same Count
func TestAuditRuleProvider_TC8_SetBased_DifferentContent(t *testing.T) {
	// Filesystem: 3 rules [A, B, C]
	// Runtime: 3 rules [A, B, X] (different content)
	fsData := &shared.AuditRuleData{
		Controls: []interface{}{},
		Files:    []interface{}{},
		Syscalls: []interface{}{"syscallA", "syscallB", "syscallC"},
	}

	rtData := &shared.AuditRuleData{
		Controls: []interface{}{},
		Files:    []interface{}{},
		Syscalls: []interface{}{"syscallA", "syscallB", "syscallX"}, // Different content
	}

	conn := &mockConnectionWithRuntime{
		runtimeData: rtData,
		runtimeErr:  nil,
	}

	provider := shared.NewAuditRuleProvider(conn)
	provider.SetParser(createMockParser(rtData, nil))

	result, err := provider.GetRules(fsData)
	assert.Error(t, err, "Should FAIL - content differs even though count is same")
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "differ", "Error should mention drift")
}

// TC-9: Both Empty
func TestAuditRuleProvider_TC9_BothEmpty(t *testing.T) {
	// Both filesystem and runtime have 0 rules (valid state)
	fsData := &shared.AuditRuleData{
		Controls: []interface{}{},
		Files:    []interface{}{},
		Syscalls: []interface{}{},
	}

	rtData := &shared.AuditRuleData{
		Controls: []interface{}{},
		Files:    []interface{}{},
		Syscalls: []interface{}{},
	}

	conn := &mockConnectionWithRuntime{
		runtimeData: rtData,
		runtimeErr:  nil,
	}

	provider := shared.NewAuditRuleProvider(conn)
	provider.SetParser(createMockParser(rtData, nil))

	result, err := provider.GetRules(fsData)
	require.NoError(t, err, "Should PASS - both empty is valid (no audit rules configured)")
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result.Controls))
	assert.Equal(t, 0, len(result.Files))
	assert.Equal(t, 0, len(result.Syscalls))
}

// Helper: Mock connection without run-command capability
type mockConnectionNoRunCommand struct {
	shared.Connection
}

func (m *mockConnectionNoRunCommand) Capabilities() shared.Capabilities {
	return shared.Capability_File // Only file capability, no RunCommand
}

// Helper: Mock connection with run-command capability that returns test data
type mockConnectionWithRuntime struct {
	runtimeData *shared.AuditRuleData
	runtimeErr  error
}

func (m *mockConnectionWithRuntime) RunCommand(cmd string) (*shared.Command, error) {
	if m.runtimeErr != nil {
		return &shared.Command{ExitStatus: 1}, m.runtimeErr
	}
	return &shared.Command{
		ExitStatus: 0,
		Stdout:     &bytes.Buffer{},
	}, nil
}

func (m *mockConnectionWithRuntime) Capabilities() shared.Capabilities {
	return shared.Capability_RunCommand | shared.Capability_File
}

func (m *mockConnectionWithRuntime) FileInfo(path string) (shared.FileInfoDetails, error) {
	return shared.FileInfoDetails{}, nil
}

func (m *mockConnectionWithRuntime) FileSystem() afero.Fs {
	return afero.NewMemMapFs()
}

func (m *mockConnectionWithRuntime) Name() string {
	return "mock-runtime"
}

func (m *mockConnectionWithRuntime) Type() shared.ConnectionType {
	return shared.Type_Local
}

func (m *mockConnectionWithRuntime) Asset() *inventory.Asset {
	return &inventory.Asset{}
}

func (m *mockConnectionWithRuntime) UpdateAsset(asset *inventory.Asset) {}

func (m *mockConnectionWithRuntime) AuditRuleProvider() *shared.AuditRuleProvider {
	return nil // Will be created by test
}

func (m *mockConnectionWithRuntime) Close() {}

func (m *mockConnectionWithRuntime) Identifier() (string, error) {
	return "mock-runtime", nil
}

func (m *mockConnectionWithRuntime) ParentID() uint32 {
	return 0
}

func (m *mockConnectionWithRuntime) ID() uint32 {
	return 1
}

// Helper function to create a mock parser that returns provided data
func createMockParser(data *shared.AuditRuleData, err error) shared.AuditRuleParser {
	return func(content string) (*shared.AuditRuleData, error) {
		if err != nil {
			return nil, err
		}
		return data, nil
	}
}
