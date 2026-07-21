// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/mock"
	"go.mondoo.com/mql/v13/utils/multierr"
	"go.mondoo.com/mql/v13/utils/syncx"
)

func newAuditdRulesTestRuntime(t *testing.T) *plugin.Runtime {
	t.Helper()

	conn, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:    "oraclelinux",
			Version: "8",
			Family:  []string{"oraclelinux", "linux"},
		},
	})
	require.NoError(t, err)

	return &plugin.Runtime{
		Connection: conn,
		Resources:  &syncx.Map[plugin.Resource]{},
	}
}

func TestAuditdRulesMissingPathReturnsEmptyLists(t *testing.T) {
	runtime := newAuditdRulesTestRuntime(t)
	rules := &mqlAuditdRules{MqlRuntime: runtime}

	controls := rules.GetControls()
	require.NoError(t, controls.Error)
	assert.Empty(t, controls.Data)
	assert.Equal(t, plugin.StateIsSet, controls.State)

	files := rules.GetFiles()
	require.NoError(t, files.Error)
	assert.Empty(t, files.Data)
	assert.Equal(t, plugin.StateIsSet, files.State)

	syscalls := rules.GetSyscalls()
	require.NoError(t, syscalls.Error)
	assert.Empty(t, syscalls.Data)
	assert.Equal(t, plugin.StateIsSet, syscalls.State)
}

func TestAuditdSyscallRuleParsing(t *testing.T) {
	runtime := newAuditdRulesTestRuntime(t)
	rules := &mqlAuditdRules{MqlRuntime: runtime}

	content := `
# 64-bit DAC permission modification rule
-a always,exit -F arch=b64 -S chmod,fchmod,chown -F auid>=1000 -F auid!=unset -k perm_mod
# 32-bit variant using the raw unset sentinel
-a always,exit -F arch=b32 -S chmod -F auid>=1000 -F auid!=4294967295 -k perm_mod
# rule without any arch or auid filters
-a always,exit -S execve -k exec
`

	var errs multierr.Errors
	rules.parse(content, &errs)
	require.NoError(t, errs.Deduplicate())
	require.Len(t, rules.Syscalls.Data, 3)

	t.Run("b64 rule derives arch, auidMin and excludesUnsetAuid", func(t *testing.T) {
		r := rules.Syscalls.Data[0].(*mqlAuditdRuleSyscall)
		assert.Equal(t, "b64", r.Arch.Data)
		assert.Equal(t, []any{"chmod", "fchmod", "chown"}, r.Syscalls.Data)
		assert.Equal(t, int64(1000), r.AuidMin.Data)
		assert.Equal(t, plugin.StateIsSet, r.AuidMin.State)
		assert.True(t, r.ExcludesUnsetAuid.Data)
		assert.Equal(t, "perm_mod", r.Keyname.Data)
	})

	t.Run("b32 rule treats the 4294967295 sentinel as unset", func(t *testing.T) {
		r := rules.Syscalls.Data[1].(*mqlAuditdRuleSyscall)
		assert.Equal(t, "b32", r.Arch.Data)
		assert.Equal(t, int64(1000), r.AuidMin.Data)
		assert.True(t, r.ExcludesUnsetAuid.Data)
	})

	t.Run("rule without arch/auid filters leaves fields empty and auidMin null", func(t *testing.T) {
		r := rules.Syscalls.Data[2].(*mqlAuditdRuleSyscall)
		assert.Equal(t, "", r.Arch.Data)
		assert.False(t, r.ExcludesUnsetAuid.Data)
		assert.NotEqual(t, 0, r.AuidMin.State&plugin.StateIsNull)
	})
}

func TestAuditdSyscallAuidGreaterThan(t *testing.T) {
	runtime := newAuditdRulesTestRuntime(t)
	rules := &mqlAuditdRules{MqlRuntime: runtime}

	// `auid>999` is the strict-greater-than spelling of `auid>=1000`; auidMin
	// reports the effective lower bound of 1000 either way.
	content := "-a always,exit -F arch=b64 -S execve -F auid>999 -k t\n"

	var errs multierr.Errors
	rules.parse(content, &errs)
	require.NoError(t, errs.Deduplicate())
	require.Len(t, rules.Syscalls.Data, 1)

	r := rules.Syscalls.Data[0].(*mqlAuditdRuleSyscall)
	assert.Equal(t, int64(1000), r.AuidMin.Data)
}

func TestAuditdSyscallRuleRepeatedFlags(t *testing.T) {
	runtime := newAuditdRulesTestRuntime(t)
	rules := &mqlAuditdRules{MqlRuntime: runtime}

	// Syscalls may be supplied via repeated -S flags as well as comma lists;
	// both forms must accumulate into a single flat syscalls list.
	content := "-a always,exit -F arch=b64 -S open -S openat,creat -k access\n"

	var errs multierr.Errors
	rules.parse(content, &errs)
	require.NoError(t, errs.Deduplicate())
	require.Len(t, rules.Syscalls.Data, 1)

	r := rules.Syscalls.Data[0].(*mqlAuditdRuleSyscall)
	assert.Equal(t, []any{"open", "openat", "creat"}, r.Syscalls.Data)
}

func TestNormalizeAuditdConfigFields(t *testing.T) {
	t.Run("lowercases keys and downcases enum values", func(t *testing.T) {
		res, err := normalizeAuditdConfigFields(map[string]any{
			"Log_Format": "ENABLED",
			"Log_File":   "/var/log/audit/audit.log",
		})
		require.NoError(t, err)
		assert.Equal(t, "enabled", res["log_format"])
		// log_file is not a downcase keyword, so its value is preserved verbatim.
		assert.Equal(t, "/var/log/audit/audit.log", res["log_file"])
	})

	t.Run("reports the offending field name for non-string values", func(t *testing.T) {
		_, err := normalizeAuditdConfigFields(map[string]any{
			"broken": map[string]any{"nested": "value"},
		})
		require.Error(t, err)
		// The error must name the field that failed (`broken`), not an empty string.
		assert.Contains(t, err.Error(), "broken")
	})
}

func TestAuditdSyscallRuleMalformedAction(t *testing.T) {
	runtime := newAuditdRulesTestRuntime(t)
	rules := &mqlAuditdRules{MqlRuntime: runtime}

	// A malformed `-a` rule carries no comma (the list segment is missing).
	// Parsing must not panic; the action is captured and the list is empty.
	content := "-a always -S execve -k t\n"

	var errs multierr.Errors
	require.NotPanics(t, func() {
		rules.parse(content, &errs)
	})
	require.NoError(t, errs.Deduplicate())
	require.Len(t, rules.Syscalls.Data, 1)

	r := rules.Syscalls.Data[0].(*mqlAuditdRuleSyscall)
	assert.Equal(t, "always", r.Action.Data)
	assert.Equal(t, "", r.List.Data)
}

func TestAuditdControlRuleParsing(t *testing.T) {
	runtime := newAuditdRulesTestRuntime(t)
	rules := &mqlAuditdRules{MqlRuntime: runtime}

	content := `
# comment line, must be skipped

-D
-b 8192
-f 1
--backlog_wait_time 60000
`

	var errs multierr.Errors
	rules.parse(content, &errs)
	require.NoError(t, errs.Deduplicate())
	require.Len(t, rules.Controls.Data, 4)
	assert.Empty(t, rules.Syscalls.Data)
	assert.Empty(t, rules.Files.Data)

	got := map[string]string{}
	for _, raw := range rules.Controls.Data {
		c := raw.(*mqlAuditdRuleControl)
		got[c.Flag.Data] = c.Value.Data
	}
	assert.Equal(t, map[string]string{
		"-D":                  "",
		"-b":                  "8192",
		"-f":                  "1",
		"--backlog_wait_time": "60000",
	}, got)
}

func TestAuditdFileRuleParsing(t *testing.T) {
	runtime := newAuditdRulesTestRuntime(t)
	rules := &mqlAuditdRules{MqlRuntime: runtime}

	content := `
-w /etc/shadow -p wa -k identity
-w /etc/passwd
`

	var errs multierr.Errors
	rules.parse(content, &errs)
	require.NoError(t, errs.Deduplicate())
	require.Len(t, rules.Files.Data, 2)

	shadow := rules.Files.Data[0].(*mqlAuditdRuleFile)
	assert.Equal(t, "/etc/shadow", shadow.Path.Data)
	assert.Equal(t, "wa", shadow.Permissions.Data)
	assert.Equal(t, "identity", shadow.Keyname.Data)

	// A watch with no -p/-k still parses, with empty permissions and keyname.
	passwd := rules.Files.Data[1].(*mqlAuditdRuleFile)
	assert.Equal(t, "/etc/passwd", passwd.Path.Data)
	assert.Equal(t, "", passwd.Permissions.Data)
	assert.Equal(t, "", passwd.Keyname.Data)
}

func TestAuditdSyscallFieldOperatorParsing(t *testing.T) {
	runtime := newAuditdRulesTestRuntime(t)
	rules := &mqlAuditdRules{MqlRuntime: runtime}

	// Exercises the reOperator ordering (>= must win over =/>) and a key-only
	// field with no operator.
	content := "-a always,exit -F arch=b64 -F auid>=1000 -F success!=0 -F exit -S execve -k t\n"

	var errs multierr.Errors
	rules.parse(content, &errs)
	require.NoError(t, errs.Deduplicate())
	require.Len(t, rules.Syscalls.Data, 1)

	got := map[string][2]string{} // key -> {op, value}
	keyOnly := []string{}
	for _, raw := range rules.Syscalls.Data[0].(*mqlAuditdRuleSyscall).Fields.Data {
		f := raw.(map[string]any)
		key, _ := f["key"].(string)
		op, hasOp := f["op"].(string)
		if !hasOp {
			keyOnly = append(keyOnly, key)
			continue
		}
		val, _ := f["value"].(string)
		got[key] = [2]string{op, val}
	}

	assert.Equal(t, [2]string{"=", "b64"}, got["arch"])
	assert.Equal(t, [2]string{">=", "1000"}, got["auid"])
	assert.Equal(t, [2]string{"!=", "0"}, got["success"])
	assert.Contains(t, keyOnly, "exit")
}
