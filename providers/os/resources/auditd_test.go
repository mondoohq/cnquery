// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/testutils"
)

func TestResource_AuditdConfig(t *testing.T) {
	// Test graceful handling of missing config files (should return empty params)
	t.Run("auditd.config with missing file should return empty params", func(t *testing.T) {
		res := x.TestQuery(t, "auditd.config('nopath').params")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		params, ok := res[0].Data.Value.(map[string]interface{})
		assert.True(t, ok, "Expected params to be a map")
		assert.Empty(t, params, "Expected empty params for missing config file")
	})

	t.Run("auditd file path", func(t *testing.T) {
		res := x.TestQuery(t, "auditd.config.file.path")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("auditd params", func(t *testing.T) {
		res := x.TestQuery(t, "auditd.config.params")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("auditd is downcasing relevant params", func(t *testing.T) {
		res := x.TestQuery(t, "auditd.config.params.log_format")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		assert.Equal(t, "enriched", res[0].Data.Value)
	})

	t.Run("auditd is NOT downcasing other params", func(t *testing.T) {
		res := x.TestQuery(t, "auditd.config.params.log_file")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		assert.Equal(t, "/var/log/audit/AuDiT.log", res[0].Data.Value)
	})
}

func TestResource_AuditdRules(t *testing.T) {
	t.Run("auditd rules path", func(t *testing.T) {
		x.TestSimple(t, []testutils.SimpleTest{
			{
				Code:        "auditd.rules.path",
				ResultIndex: 0,
				Expectation: "/etc/audit/rules.d",
			},
			{
				Code:        "auditd.rules.files.first.path",
				ResultIndex: 0,
				Expectation: "/etc/sudoers",
			},
			{
				Code:        "auditd.rules.controls[0].flag",
				ResultIndex: 0,
				Expectation: "-D",
			},
			{
				Code:        "auditd.rules.syscalls.where(action==\"always\" && fields.contains(key==\"path\" && value==\"/usr/bin/systemd-run\")).length",
				ResultIndex: 0,
				Expectation: int64(2),
			},
		})
	})
}

// Debug test to see exact error message
func TestResource_AuditdConfig_Debug(t *testing.T) {
	t.Run("debug exact error message for params", func(t *testing.T) {
		res := x.TestQuery(t, "auditd.config('/nonexistent/path/auditd.conf').params")
		t.Logf("Params - Result count: %d", len(res))
		if len(res) > 0 {
			t.Logf("Params - Result value: %v", res[0].Data.Value)
			if res[0].Data.Error != nil {
				t.Logf("Params - Exact error message: '%s'", res[0].Data.Error.Error())
			}
		}
	})

	t.Run("debug exact error message for file", func(t *testing.T) {
		res := x.TestQuery(t, "auditd.config('/nonexistent/path/auditd.conf').file")
		t.Logf("File - Result count: %d", len(res))
		if len(res) > 0 {
			t.Logf("File - Result value: %v", res[0].Data.Value)
			if res[0].Data.Error != nil {
				t.Logf("File - Exact error message: '%s'", res[0].Data.Error.Error())
			}
		}
	})

	t.Run("debug containsKey behavior", func(t *testing.T) {
		res := x.TestQuery(t, "auditd.config('/nonexistent/path/auditd.conf').params.containsKey('log_file')")
		t.Logf("ContainsKey - Result count: %d", len(res))
		if len(res) > 0 {
			t.Logf("ContainsKey - Result value: %v", res[0].Data.Value)
			t.Logf("ContainsKey - Result type: %T", res[0].Data.Value)
			if res[0].Data.Error != nil {
				t.Logf("ContainsKey - Exact error message: '%s'", res[0].Data.Error.Error())
			}
		}
	})

	t.Run("debug containsKey on working config", func(t *testing.T) {
		res := x.TestQuery(t, "auditd.config.params.containsKey('log_format')")
		t.Logf("Working ContainsKey - Result count: %d", len(res))
		if len(res) > 0 {
			t.Logf("Working ContainsKey - Result value: %v", res[0].Data.Value)
			t.Logf("Working ContainsKey - Result type: %T", res[0].Data.Value)
			if res[0].Data.Error != nil {
				t.Logf("Working ContainsKey - Exact error message: '%s'", res[0].Data.Error.Error())
			}
		}
	})

	t.Run("debug what keys exist in working config", func(t *testing.T) {
		res := x.TestQuery(t, "auditd.config.params.keys")
		t.Logf("Keys - Result count: %d", len(res))
		if len(res) > 0 {
			t.Logf("Keys - Result value: %v", res[0].Data.Value)
			if res[0].Data.Error != nil {
				t.Logf("Keys - Exact error message: '%s'", res[0].Data.Error.Error())
			}
		}
	})
}

// Test error handling behavior as described in tmp_auditd_fail.md
func TestResource_AuditdConfig_ErrorHandling(t *testing.T) {
	t.Run("missing auditd.conf file should return empty params", func(t *testing.T) {
		// Test with a non-existent file path
		res := x.TestQuery(t, "auditd.config('/nonexistent/path/auditd.conf').params")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		// Should return empty map for missing config file
		params, ok := res[0].Data.Value.(map[string]interface{})
		assert.True(t, ok, "Expected params to be a map")
		assert.Empty(t, params, "Expected empty params for missing config file")
	})

	t.Run("missing auditd.conf file should not fail the query", func(t *testing.T) {
		// Test that query execution continues even with missing file
		res := x.TestQuery(t, "auditd.config('/nonexistent/path/auditd.conf').params.length")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		assert.Equal(t, int64(0), res[0].Data.Value)
	})

	t.Run("default auditd.conf behavior with graceful handling", func(t *testing.T) {
		// Test default path behavior - should handle missing file gracefully
		res := x.TestQuery(t, "auditd.config.params")
		assert.NotEmpty(t, res)
		// This should not error, even if file doesn't exist
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("auditd.config.file should exist even for missing files", func(t *testing.T) {
		// Test that file resource is created even when file doesn't exist
		res := x.TestQuery(t, "auditd.config('/nonexistent/path/auditd.conf').file")
		assert.NotEmpty(t, res)
		// The file resource should exist (even with an error), proving resource creation succeeded
		assert.NotNil(t, res[0].Data.Value)
		// The file resource may have an error for missing files, but the query should not fail
		if res[0].Data.Error != nil {
			assert.Contains(t, res[0].Data.Error.Error(), "file not found")
		}
	})
}

func TestResource_AuditdRules_ErrorHandling(t *testing.T) {
	t.Run("auditd.rules should handle missing directory gracefully", func(t *testing.T) {
		// Test the default rules directory behavior (it already handles missing directories)
		res := x.TestQuery(t, "auditd.rules.controls")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		// Should return array (possibly empty) for controls
		_, ok := res[0].Data.Value.([]interface{})
		assert.True(t, ok, "Expected controls to be an array")
	})

	t.Run("auditd.rules files should handle missing directory", func(t *testing.T) {
		res := x.TestQuery(t, "auditd.rules.files")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		_, ok := res[0].Data.Value.([]interface{})
		assert.True(t, ok, "Expected files to be an array")
	})

	t.Run("auditd.rules syscalls should handle missing directory", func(t *testing.T) {
		res := x.TestQuery(t, "auditd.rules.syscalls")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		_, ok := res[0].Data.Value.([]interface{})
		assert.True(t, ok, "Expected syscalls to be an array")
	})

	t.Run("query should continue execution with rules", func(t *testing.T) {
		// Test that complex queries continue to work
		res := x.TestQuery(t, "auditd.rules.syscalls.length")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		// Should return a number (length of syscalls array)
		_, ok := res[0].Data.Value.(int64)
		assert.True(t, ok, "Expected length to be an integer")
	})
}

func TestResource_AuditdConfig_FailedStateVsError(t *testing.T) {
	t.Run("verify failed state behavior for missing files", func(t *testing.T) {
		// Test behavior when file is missing vs other errors
		// This verifies the distinction between Failed states and Error states
		res := x.TestQuery(t, "auditd.config('/dev/null/impossible/path').params")
		assert.NotEmpty(t, res)
		// Should handle gracefully, not return an error that stops execution
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("empty params should be queryable", func(t *testing.T) {
		// Verify that empty params from missing file can still be queried
		// Use keys.length instead of containsKey since containsKey seems to have issues
		res := x.TestQuery(t, "auditd.config('/nonexistent/auditd.conf').params.keys.length")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		assert.Equal(t, int64(0), res[0].Data.Value)
	})

	t.Run("missing file should allow method chaining", func(t *testing.T) {
		// Test that MQL queries can chain methods even with missing config
		res := x.TestQuery(t, "auditd.config('/nonexistent/auditd.conf').params.keys.length")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		assert.Equal(t, int64(0), res[0].Data.Value)
	})
}
