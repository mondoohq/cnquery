// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSudoersLine_BasicUserSpec(t *testing.T) {
	line := "root ALL=(ALL:ALL) ALL"
	parsed := parseSudoersLine(line)

	require.NotNil(t, parsed)
	assert.Equal(t, "user_spec", parsed.entryType)
	assert.Equal(t, []string{"root"}, parsed.users)
	assert.Equal(t, []string{"ALL"}, parsed.hosts)
	assert.Equal(t, []string{"ALL"}, parsed.runasUsers)
	assert.Equal(t, []string{"ALL"}, parsed.runasGroups)
	assert.Equal(t, []string{"ALL"}, parsed.commands)
}

func TestParseSudoersLine_GroupSpec(t *testing.T) {
	line := "%sudo ALL=(ALL:ALL) ALL"
	parsed := parseSudoersLine(line)

	require.NotNil(t, parsed)
	assert.Equal(t, "user_spec", parsed.entryType)
	assert.Equal(t, []string{"%sudo"}, parsed.users)
	assert.Equal(t, []string{"ALL"}, parsed.hosts)
	assert.Equal(t, []string{"ALL"}, parsed.runasUsers)
	assert.Equal(t, []string{"ALL"}, parsed.runasGroups)
}

func TestParseSudoersLine_NoPassword(t *testing.T) {
	line := "john ALL=(ALL) NOPASSWD: ALL"
	parsed := parseSudoersLine(line)

	require.NotNil(t, parsed)
	assert.Equal(t, "user_spec", parsed.entryType)
	assert.Equal(t, []string{"john"}, parsed.users)
	assert.Equal(t, []string{"ALL"}, parsed.hosts)
	assert.Equal(t, []string{"ALL"}, parsed.runasUsers)
	assert.Empty(t, parsed.runasGroups)
	assert.Equal(t, []string{"NOPASSWD"}, parsed.tags)
	assert.Equal(t, []string{"ALL"}, parsed.commands)
}

func TestParseSudoersLine_SpecificCommands(t *testing.T) {
	line := "jane ALL=(root) NOPASSWD: /usr/bin/systemctl restart nginx, /usr/bin/systemctl reload nginx"
	parsed := parseSudoersLine(line)

	require.NotNil(t, parsed)
	assert.Equal(t, "user_spec", parsed.entryType)
	assert.Equal(t, []string{"jane"}, parsed.users)
	assert.Equal(t, []string{"ALL"}, parsed.hosts)
	assert.Equal(t, []string{"root"}, parsed.runasUsers)
	assert.Equal(t, []string{"NOPASSWD"}, parsed.tags)
	assert.Equal(t, []string{
		"/usr/bin/systemctl restart nginx",
		"/usr/bin/systemctl reload nginx",
	}, parsed.commands)
}

func TestParseSudoersLine_MultipleTags(t *testing.T) {
	line := "bob ALL=(ALL) NOPASSWD: SETENV: /usr/bin/docker"
	parsed := parseSudoersLine(line)

	require.NotNil(t, parsed)
	assert.Equal(t, "user_spec", parsed.entryType)
	assert.Equal(t, []string{"bob"}, parsed.users)
	assert.Equal(t, []string{"NOPASSWD", "SETENV"}, parsed.tags)
	assert.Equal(t, []string{"/usr/bin/docker"}, parsed.commands)
}

func TestParseSudoersLine_MultipleUsers(t *testing.T) {
	line := "john, jane, bob ALL=(ALL) ALL"
	parsed := parseSudoersLine(line)

	require.NotNil(t, parsed)
	assert.Equal(t, "user_spec", parsed.entryType)
	assert.Equal(t, []string{"john", "jane", "bob"}, parsed.users)
	assert.Equal(t, []string{"ALL"}, parsed.hosts)
}

func TestParseSudoersLine_SpecificHost(t *testing.T) {
	line := "admin webserver01, webserver02=(ALL) ALL"
	parsed := parseSudoersLine(line)

	require.NotNil(t, parsed)
	assert.Equal(t, "user_spec", parsed.entryType)
	assert.Equal(t, []string{"admin"}, parsed.users)
	assert.Equal(t, []string{"webserver01", "webserver02"}, parsed.hosts)
}

func TestParseSudoersLine_RunAsGroupOnly(t *testing.T) {
	line := "developer ALL=(ALL) ALL"
	parsed := parseSudoersLine(line)

	require.NotNil(t, parsed)
	assert.Equal(t, "user_spec", parsed.entryType)
	assert.Equal(t, []string{"developer"}, parsed.users)
	assert.Equal(t, []string{"ALL"}, parsed.runasUsers)
	assert.Empty(t, parsed.runasGroups)
}

func TestParseDefaultsLine_Global(t *testing.T) {
	scope, target, parameter, value, operation, negated := parseDefaultsLine("Defaults env_reset")

	assert.Equal(t, "global", scope)
	assert.Equal(t, "", target)
	assert.Equal(t, "env_reset", parameter)
	assert.Equal(t, "", value)
	assert.Equal(t, "", operation)
	assert.False(t, negated)
}

func TestParseDefaultsLine_WithValue(t *testing.T) {
	scope, target, parameter, value, operation, negated := parseDefaultsLine("Defaults secure_path=\"/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\"")

	assert.Equal(t, "global", scope)
	assert.Equal(t, "", target)
	assert.Equal(t, "secure_path", parameter)
	assert.Equal(t, "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin", value)
	assert.Equal(t, "=", operation)
	assert.False(t, negated)
}

func TestParseDefaultsLine_UserScope(t *testing.T) {
	scope, target, parameter, value, operation, negated := parseDefaultsLine("Defaults:john !requiretty")

	assert.Equal(t, "user", scope)
	assert.Equal(t, "john", target)
	assert.Equal(t, "requiretty", parameter)
	assert.Equal(t, "", value)
	assert.Equal(t, "", operation)
	assert.True(t, negated)
}

func TestParseDefaultsLine_HostScope(t *testing.T) {
	scope, target, parameter, value, operation, negated := parseDefaultsLine("Defaults@webserver env_keep += \"FOO\"")

	assert.Equal(t, "host", scope)
	assert.Equal(t, "webserver", target)
	assert.Equal(t, "env_keep", parameter)
	assert.Equal(t, "FOO", value)
	assert.Equal(t, "+=", operation)
	assert.False(t, negated)
}

func TestParseDefaultsLine_RunasScope(t *testing.T) {
	scope, target, parameter, value, operation, negated := parseDefaultsLine("Defaults>root env_reset")

	assert.Equal(t, "runas", scope)
	assert.Equal(t, "root", target)
	assert.Equal(t, "env_reset", parameter)
	assert.Equal(t, "", value)
	assert.Equal(t, "", operation)
	assert.False(t, negated)
}

func TestParseDefaultsLine_CommandScope(t *testing.T) {
	scope, target, parameter, value, operation, negated := parseDefaultsLine("Defaults!/usr/bin/su !authenticate")

	assert.Equal(t, "command", scope)
	assert.Equal(t, "/usr/bin/su", target)
	assert.Equal(t, "authenticate", parameter)
	assert.Equal(t, "", value)
	assert.Equal(t, "", operation)
	assert.True(t, negated)
}

func TestParseSudoersLine_ComplexCommand(t *testing.T) {
	line := "backup ALL=(root) NOPASSWD: /usr/bin/rsync -av /data/ /backup/"
	parsed := parseSudoersLine(line)

	require.NotNil(t, parsed)
	assert.Equal(t, "user_spec", parsed.entryType)
	assert.Equal(t, []string{"backup"}, parsed.users)
	assert.Equal(t, []string{"root"}, parsed.runasUsers)
	assert.Equal(t, []string{"NOPASSWD"}, parsed.tags)
	assert.Equal(t, 1, len(parsed.commands))
	assert.Contains(t, parsed.commands[0], "rsync")
}

func TestParseSudoersLine_EmptyLine(t *testing.T) {
	line := ""
	parsed := parseSudoersLine(line)

	// Empty lines should return nil
	assert.Nil(t, parsed)
}

func TestParseSudoersLine_CommentLine(t *testing.T) {
	line := "# This is a comment"
	parsed := parseSudoersLine(line)

	// parseSudoersLine doesn't filter comments, that's done in parseSudoersContent
	// But it should still parse as an invalid entry
	assert.Nil(t, parsed)
}

func TestSmartSplit_BasicSplit(t *testing.T) {
	result := smartSplit("user host command")
	assert.Equal(t, []string{"user", "host", "command"}, result)
}

func TestSmartSplit_WithQuotes(t *testing.T) {
	result := smartSplit(`user host "command with spaces"`)
	assert.Equal(t, []string{"user", "host", `"command with spaces"`}, result)
}

func TestSmartSplit_WithEscapes(t *testing.T) {
	result := smartSplit(`user host command\ with\ escape`)
	assert.Equal(t, []string{"user", "host", `command\ with\ escape`}, result)
}

func TestSplitCommands_SingleCommand(t *testing.T) {
	result := splitCommands("/usr/bin/systemctl restart nginx")
	assert.Equal(t, []string{"/usr/bin/systemctl restart nginx"}, result)
}

func TestSplitCommands_MultipleCommands(t *testing.T) {
	result := splitCommands("/usr/bin/systemctl restart nginx, /usr/bin/systemctl reload nginx")
	assert.Equal(t, []string{
		"/usr/bin/systemctl restart nginx",
		"/usr/bin/systemctl reload nginx",
	}, result)
}

func TestSplitCommands_WithQuotedComma(t *testing.T) {
	result := splitCommands(`/usr/bin/echo "hello, world", /usr/bin/echo goodbye`)
	assert.Equal(t, []string{
		`/usr/bin/echo "hello, world"`,
		"/usr/bin/echo goodbye",
	}, result)
}

func TestSplitAndTrim_BasicSplit(t *testing.T) {
	result := splitAndTrim("john, jane, bob", ",")
	assert.Equal(t, []string{"john", "jane", "bob"}, result)
}

func TestSplitAndTrim_WithExtraSpaces(t *testing.T) {
	result := splitAndTrim("  john  ,  jane  ,  bob  ", ",")
	assert.Equal(t, []string{"john", "jane", "bob"}, result)
}

func TestSplitAndTrim_EmptyString(t *testing.T) {
	result := splitAndTrim("", ",")
	assert.Empty(t, result)
}

func TestSplitAndTrim_SingleItem(t *testing.T) {
	result := splitAndTrim("john", ",")
	assert.Equal(t, []string{"john"}, result)
}

func TestParseSudoersLine_NoExecTag(t *testing.T) {
	line := "test ALL=(ALL) NOEXEC: /usr/bin/vim"
	parsed := parseSudoersLine(line)

	require.NotNil(t, parsed)
	assert.Equal(t, []string{"NOEXEC"}, parsed.tags)
}

func TestParseSudoersLine_LogInputOutputTags(t *testing.T) {
	line := "admin ALL=(ALL) LOG_INPUT: LOG_OUTPUT: /usr/bin/bash"
	parsed := parseSudoersLine(line)

	require.NotNil(t, parsed)
	assert.Equal(t, []string{"LOG_INPUT", "LOG_OUTPUT"}, parsed.tags)
}

func TestParseSudoersLine_MailTag(t *testing.T) {
	line := "user ALL=(ALL) MAIL: /usr/bin/command"
	parsed := parseSudoersLine(line)

	require.NotNil(t, parsed)
	assert.Equal(t, []string{"MAIL"}, parsed.tags)
}

func TestParseSudoersLine_MultipleUserSpecs(t *testing.T) {
	// Test that multiple user spec lines parse correctly
	lines := []string{
		"root    ALL=(ALL:ALL) ALL",
		"%sudo   ALL=(ALL:ALL) ALL",
		"john    ALL=(ALL) NOPASSWD: ALL",
	}

	for _, line := range lines {
		parsed := parseSudoersLine(line)
		assert.NotNil(t, parsed)
		assert.Equal(t, "user_spec", parsed.entryType)
	}
}

func TestParseSudoersLine_LineContinuation(t *testing.T) {
	// Line continuations are handled in parseSudoersContent
	// This test verifies that a concatenated line parses correctly
	line := "john ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart nginx"
	parsed := parseSudoersLine(line)

	require.NotNil(t, parsed)
	assert.Equal(t, "user_spec", parsed.entryType)
}

func TestParseSudoersLine_NegatedUser(t *testing.T) {
	line := "!root ALL=(ALL) ALL"
	parsed := parseSudoersLine(line)

	require.NotNil(t, parsed)
	assert.Equal(t, []string{"!root"}, parsed.users)
}

func TestParseSudoersLine_NegatedCommand(t *testing.T) {
	line := "john ALL=(ALL) ALL, !/usr/bin/su"
	parsed := parseSudoersLine(line)

	require.NotNil(t, parsed)
	assert.Equal(t, 2, len(parsed.commands))
	assert.Contains(t, parsed.commands, "ALL")
	assert.Contains(t, parsed.commands, "!/usr/bin/su")
}

func TestParseSudoersLine_WildcardCommand(t *testing.T) {
	line := "john ALL=(ALL) /usr/bin/*"
	parsed := parseSudoersLine(line)

	require.NotNil(t, parsed)
	assert.Equal(t, []string{"/usr/bin/*"}, parsed.commands)
}
