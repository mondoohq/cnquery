// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sudoers_test

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v12/providers/os/resources/sudoers"
)

func TestParseLine_BasicUserSpec(t *testing.T) {
	line := "root ALL=(ALL:ALL) ALL"
	parsed := sudoers.ToParsedLine(sudoers.ParseLine(line))

	require.NotNil(t, parsed)
	assert.Equal(t, "user_spec", parsed.EntryType)
	assert.Equal(t, []string{"root"}, parsed.Users)
	assert.Equal(t, []string{"ALL"}, parsed.Hosts)
	assert.Equal(t, []string{"ALL"}, parsed.RunasUsers)
	assert.Equal(t, []string{"ALL"}, parsed.RunasGroups)
	assert.Equal(t, []string{"ALL"}, parsed.Commands)
}

func TestParseLine_GroupSpec(t *testing.T) {
	line := "%sudo ALL=(ALL:ALL) ALL"
	parsed := sudoers.ToParsedLine(sudoers.ParseLine(line))

	require.NotNil(t, parsed)
	assert.Equal(t, "user_spec", parsed.EntryType)
	assert.Equal(t, []string{"%sudo"}, parsed.Users)
	assert.Equal(t, []string{"ALL"}, parsed.Hosts)
	assert.Equal(t, []string{"ALL"}, parsed.RunasUsers)
	assert.Equal(t, []string{"ALL"}, parsed.RunasGroups)
}

func TestParseLine_NoPassword(t *testing.T) {
	line := "john ALL=(ALL) NOPASSWD: ALL"
	parsed := sudoers.ToParsedLine(sudoers.ParseLine(line))

	require.NotNil(t, parsed)
	assert.Equal(t, "user_spec", parsed.EntryType)
	assert.Equal(t, []string{"john"}, parsed.Users)
	assert.Equal(t, []string{"ALL"}, parsed.Hosts)
	assert.Equal(t, []string{"ALL"}, parsed.RunasUsers)
	assert.Empty(t, parsed.RunasGroups)
	assert.Equal(t, []string{"NOPASSWD"}, parsed.Tags)
	assert.Equal(t, []string{"ALL"}, parsed.Commands)
}

func TestParseLine_SpecificCommands(t *testing.T) {
	line := "jane ALL=(root) NOPASSWD: /usr/bin/systemctl restart nginx, /usr/bin/systemctl reload nginx"
	parsed := sudoers.ToParsedLine(sudoers.ParseLine(line))

	require.NotNil(t, parsed)
	assert.Equal(t, "user_spec", parsed.EntryType)
	assert.Equal(t, []string{"jane"}, parsed.Users)
	assert.Equal(t, []string{"ALL"}, parsed.Hosts)
	assert.Equal(t, []string{"root"}, parsed.RunasUsers)
	assert.Equal(t, []string{"NOPASSWD"}, parsed.Tags)
	assert.Equal(t, []string{
		"/usr/bin/systemctl restart nginx",
		"/usr/bin/systemctl reload nginx",
	}, parsed.Commands)
}

func TestParseLine_MultipleTags(t *testing.T) {
	line := "bob ALL=(ALL) NOPASSWD: SETENV: /usr/bin/docker"
	parsed := sudoers.ToParsedLine(sudoers.ParseLine(line))

	require.NotNil(t, parsed)
	assert.Equal(t, "user_spec", parsed.EntryType)
	assert.Equal(t, []string{"bob"}, parsed.Users)
	assert.Equal(t, []string{"NOPASSWD", "SETENV"}, parsed.Tags)
	assert.Equal(t, []string{"/usr/bin/docker"}, parsed.Commands)
}

func TestParseLine_MultipleUsers(t *testing.T) {
	line := "john, jane, bob ALL=(ALL) ALL"
	parsed := sudoers.ToParsedLine(sudoers.ParseLine(line))

	require.NotNil(t, parsed)
	assert.Equal(t, "user_spec", parsed.EntryType)
	assert.Equal(t, []string{"john", "jane", "bob"}, parsed.Users)
	assert.Equal(t, []string{"ALL"}, parsed.Hosts)
}

func TestParseLine_SpecificHost(t *testing.T) {
	line := "admin webserver01, webserver02=(ALL) ALL"
	parsed := sudoers.ToParsedLine(sudoers.ParseLine(line))

	require.NotNil(t, parsed)
	assert.Equal(t, "user_spec", parsed.EntryType)
	assert.Equal(t, []string{"admin"}, parsed.Users)
	assert.Equal(t, []string{"webserver01", "webserver02"}, parsed.Hosts)
}

func TestParseLine_RunAsGroupOnly(t *testing.T) {
	line := "developer ALL=(ALL) ALL"
	parsed := sudoers.ToParsedLine(sudoers.ParseLine(line))

	require.NotNil(t, parsed)
	assert.Equal(t, "user_spec", parsed.EntryType)
	assert.Equal(t, []string{"developer"}, parsed.Users)
	assert.Equal(t, []string{"ALL"}, parsed.RunasUsers)
	assert.Empty(t, parsed.RunasGroups)
}

func TestParseLine_ComplexCommand(t *testing.T) {
	line := "backup ALL=(root) NOPASSWD: /usr/bin/rsync -av /data/ /backup/"
	parsed := sudoers.ToParsedLine(sudoers.ParseLine(line))

	require.NotNil(t, parsed)
	assert.Equal(t, "user_spec", parsed.EntryType)
	assert.Equal(t, []string{"backup"}, parsed.Users)
	assert.Equal(t, []string{"root"}, parsed.RunasUsers)
	assert.Equal(t, []string{"NOPASSWD"}, parsed.Tags)
	assert.Equal(t, 1, len(parsed.Commands))
	assert.Contains(t, parsed.Commands[0], "rsync")
}

func TestParseLine_EmptyLine(t *testing.T) {
	line := ""
	parsed := sudoers.ParseLine(line)
	assert.Nil(t, parsed)
}

func TestParseLine_CommentLine(t *testing.T) {
	line := "# This is a comment"
	parsed := sudoers.ParseLine(line)
	assert.Nil(t, parsed)
}

func TestParseLine_NoExecTag(t *testing.T) {
	line := "test ALL=(ALL) NOEXEC: /usr/bin/vim"
	parsed := sudoers.ToParsedLine(sudoers.ParseLine(line))

	require.NotNil(t, parsed)
	assert.Equal(t, []string{"NOEXEC"}, parsed.Tags)
}

func TestParseLine_LogInputOutputTags(t *testing.T) {
	line := "admin ALL=(ALL) LOG_INPUT: LOG_OUTPUT: /usr/bin/bash"
	parsed := sudoers.ToParsedLine(sudoers.ParseLine(line))

	require.NotNil(t, parsed)
	assert.Equal(t, []string{"LOG_INPUT", "LOG_OUTPUT"}, parsed.Tags)
}

func TestParseLine_MailTag(t *testing.T) {
	line := "user ALL=(ALL) MAIL: /usr/bin/command"
	parsed := sudoers.ToParsedLine(sudoers.ParseLine(line))

	require.NotNil(t, parsed)
	assert.Equal(t, []string{"MAIL"}, parsed.Tags)
}

func TestParseLine_NegatedUser(t *testing.T) {
	line := "!root ALL=(ALL) ALL"
	parsed := sudoers.ToParsedLine(sudoers.ParseLine(line))

	require.NotNil(t, parsed)
	assert.Equal(t, []string{"!root"}, parsed.Users)
}

func TestParseLine_NegatedCommand(t *testing.T) {
	line := "john ALL=(ALL) ALL, !/usr/bin/su"
	parsed := sudoers.ToParsedLine(sudoers.ParseLine(line))

	require.NotNil(t, parsed)
	assert.Equal(t, 2, len(parsed.Commands))
	assert.Contains(t, parsed.Commands, "ALL")
	assert.Contains(t, parsed.Commands, "!/usr/bin/su")
}

func TestParseLine_WildcardCommand(t *testing.T) {
	line := "john ALL=(ALL) /usr/bin/*"
	parsed := sudoers.ToParsedLine(sudoers.ParseLine(line))

	require.NotNil(t, parsed)
	assert.Equal(t, []string{"/usr/bin/*"}, parsed.Commands)
}

func TestParseDefaultsLine_Global(t *testing.T) {
	scope, target, parameter, value, operation, negated := sudoers.ParseDefaultsLine("Defaults env_reset")

	assert.Equal(t, "global", scope)
	assert.Equal(t, "", target)
	assert.Equal(t, "env_reset", parameter)
	assert.Equal(t, "", value)
	assert.Equal(t, "", operation)
	assert.False(t, negated)
}

func TestParseDefaultsLine_WithValue(t *testing.T) {
	scope, target, parameter, value, operation, negated := sudoers.ParseDefaultsLine("Defaults secure_path=\"/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\"")

	assert.Equal(t, "global", scope)
	assert.Equal(t, "", target)
	assert.Equal(t, "secure_path", parameter)
	assert.Equal(t, "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin", value)
	assert.Equal(t, "=", operation)
	assert.False(t, negated)
}

func TestParseDefaultsLine_UserScope(t *testing.T) {
	scope, target, parameter, value, operation, negated := sudoers.ParseDefaultsLine("Defaults:john !requiretty")

	assert.Equal(t, "user", scope)
	assert.Equal(t, "john", target)
	assert.Equal(t, "requiretty", parameter)
	assert.Equal(t, "", value)
	assert.Equal(t, "", operation)
	assert.True(t, negated)
}

func TestParseDefaultsLine_HostScope(t *testing.T) {
	scope, target, parameter, value, operation, negated := sudoers.ParseDefaultsLine("Defaults@webserver env_keep += \"FOO\"")

	assert.Equal(t, "host", scope)
	assert.Equal(t, "webserver", target)
	assert.Equal(t, "env_keep", parameter)
	assert.Equal(t, "FOO", value)
	assert.Equal(t, "+=", operation)
	assert.False(t, negated)
}

func TestParseDefaultsLine_RunasScope(t *testing.T) {
	scope, target, parameter, value, operation, negated := sudoers.ParseDefaultsLine("Defaults>root env_reset")

	assert.Equal(t, "runas", scope)
	assert.Equal(t, "root", target)
	assert.Equal(t, "env_reset", parameter)
	assert.Equal(t, "", value)
	assert.Equal(t, "", operation)
	assert.False(t, negated)
}

func TestParseDefaultsLine_CommandScope(t *testing.T) {
	scope, target, parameter, value, operation, negated := sudoers.ParseDefaultsLine("Defaults!/usr/bin/su !authenticate")

	assert.Equal(t, "command", scope)
	assert.Equal(t, "/usr/bin/su", target)
	assert.Equal(t, "authenticate", parameter)
	assert.Equal(t, "", value)
	assert.Equal(t, "", operation)
	assert.True(t, negated)
}

func TestSmartSplit_BasicSplit(t *testing.T) {
	result := sudoers.SmartSplit("user host command")
	assert.Equal(t, []string{"user", "host", "command"}, result)
}

func TestSmartSplit_WithQuotes(t *testing.T) {
	result := sudoers.SmartSplit(`user host "command with spaces"`)
	assert.Equal(t, []string{"user", "host", `"command with spaces"`}, result)
}

func TestSmartSplit_WithEscapes(t *testing.T) {
	result := sudoers.SmartSplit(`user host command\ with\ escape`)
	assert.Equal(t, []string{"user", "host", `command\ with\ escape`}, result)
}

func TestSplitCommands_SingleCommand(t *testing.T) {
	result := sudoers.SplitCommands("/usr/bin/systemctl restart nginx")
	assert.Equal(t, []string{"/usr/bin/systemctl restart nginx"}, result)
}

func TestSplitCommands_MultipleCommands(t *testing.T) {
	result := sudoers.SplitCommands("/usr/bin/systemctl restart nginx, /usr/bin/systemctl reload nginx")
	assert.Equal(t, []string{
		"/usr/bin/systemctl restart nginx",
		"/usr/bin/systemctl reload nginx",
	}, result)
}

func TestSplitCommands_WithQuotedComma(t *testing.T) {
	result := sudoers.SplitCommands(`/usr/bin/echo "hello, world", /usr/bin/echo goodbye`)
	assert.Equal(t, []string{
		`/usr/bin/echo "hello, world"`,
		"/usr/bin/echo goodbye",
	}, result)
}

func TestSplitAndTrim_BasicSplit(t *testing.T) {
	result := sudoers.SplitAndTrim("john, jane, bob", ",")
	assert.Equal(t, []string{"john", "jane", "bob"}, result)
}

func TestSplitAndTrim_WithExtraSpaces(t *testing.T) {
	result := sudoers.SplitAndTrim("  john  ,  jane  ,  bob  ", ",")
	assert.Equal(t, []string{"john", "jane", "bob"}, result)
}

func TestSplitAndTrim_EmptyString(t *testing.T) {
	result := sudoers.SplitAndTrim("", ",")
	assert.Empty(t, result)
}

func TestSplitAndTrim_SingleItem(t *testing.T) {
	result := sudoers.SplitAndTrim("john", ",")
	assert.Equal(t, []string{"john"}, result)
}

func TestIncludeRegex_AtInclude(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"@include /etc/sudoers.local", "/etc/sudoers.local"},
		{"@include /path/to/file", "/path/to/file"},
		{"@include   /path/with/spaces  ", "/path/with/spaces"},
	}

	for _, tt := range tests {
		matches := sudoers.IncludeRegex.FindStringSubmatch(tt.line)
		require.NotNil(t, matches, "line: %s", tt.line)
		assert.Equal(t, tt.expected, strings.TrimSpace(matches[1]))
	}
}

func TestIncludeRegex_HashInclude(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"#include /etc/sudoers.local", "/etc/sudoers.local"},
		{"#include /path/to/file", "/path/to/file"},
	}

	for _, tt := range tests {
		matches := sudoers.IncludeRegex.FindStringSubmatch(tt.line)
		require.NotNil(t, matches, "line: %s", tt.line)
		assert.Equal(t, tt.expected, strings.TrimSpace(matches[1]))
	}
}

func TestIncludedirRegex_AtIncludedir(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"@includedir /etc/sudoers.d", "/etc/sudoers.d"},
		{"@includedir /path/to/dir", "/path/to/dir"},
		{"@includedir   /path/with/spaces  ", "/path/with/spaces"},
	}

	for _, tt := range tests {
		matches := sudoers.IncludedirRegex.FindStringSubmatch(tt.line)
		require.NotNil(t, matches, "line: %s", tt.line)
		assert.Equal(t, tt.expected, strings.TrimSpace(matches[1]))
	}
}

func TestIncludedirRegex_HashIncludedir(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"#includedir /etc/sudoers.d", "/etc/sudoers.d"},
		{"#includedir /path/to/dir", "/path/to/dir"},
	}

	for _, tt := range tests {
		matches := sudoers.IncludedirRegex.FindStringSubmatch(tt.line)
		require.NotNil(t, matches, "line: %s", tt.line)
		assert.Equal(t, tt.expected, strings.TrimSpace(matches[1]))
	}
}

func TestIncludeRegex_NotMatchingLines(t *testing.T) {
	lines := []string{
		"# This is a comment about include",
		"Defaults env_reset",
		"root ALL=(ALL) ALL",
		"@includedir /etc/sudoers.d", // includedir should not match include
	}

	for _, line := range lines {
		matches := sudoers.IncludeRegex.FindStringSubmatch(line)
		assert.Nil(t, matches, "line should not match: %s", line)
	}
}

func TestIncludedirRegex_NotMatchingLines(t *testing.T) {
	lines := []string{
		"# This is a comment about includedir",
		"Defaults env_reset",
		"root ALL=(ALL) ALL",
		"@include /etc/sudoers.local", // include should not match includedir
	}

	for _, line := range lines {
		matches := sudoers.IncludedirRegex.FindStringSubmatch(line)
		assert.Nil(t, matches, "line should not match: %s", line)
	}
}

func TestParseUserSpecs(t *testing.T) {
	content := `# Sample sudoers file
root ALL=(ALL:ALL) ALL
%sudo ALL=(ALL:ALL) ALL
john ALL=(ALL) NOPASSWD: ALL
`
	specs := sudoers.ParseUserSpecs("/etc/sudoers", content)

	require.Len(t, specs, 3)

	assert.Equal(t, "/etc/sudoers", specs[0].File)
	assert.Equal(t, 2, specs[0].LineNumber)
	assert.Equal(t, []string{"root"}, specs[0].Users)

	assert.Equal(t, 3, specs[1].LineNumber)
	assert.Equal(t, []string{"%sudo"}, specs[1].Users)

	assert.Equal(t, 4, specs[2].LineNumber)
	assert.Equal(t, []string{"john"}, specs[2].Users)
	assert.Equal(t, []string{"NOPASSWD"}, specs[2].Tags)
}

func TestParseDefaults(t *testing.T) {
	content := `# Sample sudoers file
Defaults env_reset
Defaults secure_path="/usr/local/sbin:/usr/local/bin"
Defaults:john !requiretty
`
	defaults := sudoers.ParseDefaults("/etc/sudoers", content)

	require.Len(t, defaults, 3)

	assert.Equal(t, "global", defaults[0].Scope)
	assert.Equal(t, "env_reset", defaults[0].Parameter)

	assert.Equal(t, "global", defaults[1].Scope)
	assert.Equal(t, "secure_path", defaults[1].Parameter)
	assert.Equal(t, "/usr/local/sbin:/usr/local/bin", defaults[1].Value)

	assert.Equal(t, "user", defaults[2].Scope)
	assert.Equal(t, "john", defaults[2].Target)
	assert.True(t, defaults[2].Negated)
}

func TestParseAliases(t *testing.T) {
	content := `# Sample sudoers file
User_Alias ADMINS = john, jane, bob
Host_Alias WEBSERVERS = web1, web2, web3
Cmnd_Alias SERVICES = /usr/bin/systemctl, /usr/bin/service
`
	aliases := sudoers.ParseAliases("/etc/sudoers", content)

	require.Len(t, aliases, 3)

	assert.Equal(t, "user", aliases[0].Type)
	assert.Equal(t, "ADMINS", aliases[0].Name)
	assert.Equal(t, []string{"john", "jane", "bob"}, aliases[0].Members)

	assert.Equal(t, "host", aliases[1].Type)
	assert.Equal(t, "WEBSERVERS", aliases[1].Name)

	assert.Equal(t, "cmnd", aliases[2].Type)
	assert.Equal(t, "SERVICES", aliases[2].Name)
}

func TestParseUserSpecs_LineContinuation(t *testing.T) {
	content := `john ALL=(ALL) NOPASSWD: \
    /usr/bin/systemctl restart nginx, \
    /usr/bin/systemctl reload nginx
`
	specs := sudoers.ParseUserSpecs("/etc/sudoers", content)

	require.Len(t, specs, 1)
	assert.Equal(t, 1, specs[0].LineNumber)
	assert.Equal(t, []string{"john"}, specs[0].Users)
	assert.Len(t, specs[0].Commands, 2)
}

// Tests using TOML-based mock connection

func TestSudoersParser_MainConfig(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/linux.toml"))
	require.NoError(t, err)

	f, err := conn.FileSystem().Open("/etc/sudoers")
	require.NoError(t, err)
	defer f.Close()

	content, err := io.ReadAll(f)
	require.NoError(t, err)

	// Test parsing user specs
	specs := sudoers.ParseUserSpecs("/etc/sudoers", string(content))
	require.NotEmpty(t, specs)

	// First entry: root ALL=(ALL:ALL) ALL
	var rootSpec *sudoers.UserSpec
	for i := range specs {
		if len(specs[i].Users) > 0 && specs[i].Users[0] == "root" {
			rootSpec = &specs[i]
			break
		}
	}
	require.NotNil(t, rootSpec, "root user spec should exist")
	assert.Equal(t, "/etc/sudoers", rootSpec.File)
	assert.Equal(t, []string{"ALL"}, rootSpec.Hosts)
	assert.Equal(t, []string{"ALL"}, rootSpec.RunasUsers)
	assert.Equal(t, []string{"ALL"}, rootSpec.RunasGroups)
	assert.Equal(t, []string{"ALL"}, rootSpec.Commands)

	// %admin ALL=(ALL) ALL
	var adminSpec *sudoers.UserSpec
	for i := range specs {
		if len(specs[i].Users) > 0 && specs[i].Users[0] == "%admin" {
			adminSpec = &specs[i]
			break
		}
	}
	require.NotNil(t, adminSpec, "%admin user spec should exist")
	assert.Equal(t, []string{"ALL"}, adminSpec.Hosts)

	// ADMINS ALL=(ALL) NOPASSWD: SHUTDOWN
	var adminsSpec *sudoers.UserSpec
	for i := range specs {
		if len(specs[i].Users) > 0 && specs[i].Users[0] == "ADMINS" {
			adminsSpec = &specs[i]
			break
		}
	}
	require.NotNil(t, adminsSpec, "ADMINS user spec should exist")
	assert.Equal(t, []string{"NOPASSWD"}, adminsSpec.Tags)
	assert.Equal(t, []string{"SHUTDOWN"}, adminsSpec.Commands)
}

func TestSudoersParser_Defaults(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/linux.toml"))
	require.NoError(t, err)

	f, err := conn.FileSystem().Open("/etc/sudoers")
	require.NoError(t, err)
	defer f.Close()

	content, err := io.ReadAll(f)
	require.NoError(t, err)

	defaults := sudoers.ParseDefaults("/etc/sudoers", string(content))
	require.NotEmpty(t, defaults)

	// Find env_reset default
	var envReset *sudoers.Default
	for i := range defaults {
		if defaults[i].Parameter == "env_reset" {
			envReset = &defaults[i]
			break
		}
	}
	require.NotNil(t, envReset, "env_reset default should exist")
	assert.Equal(t, "global", envReset.Scope)
	assert.Equal(t, "", envReset.Target)
	assert.False(t, envReset.Negated)

	// Find secure_path default
	var securePath *sudoers.Default
	for i := range defaults {
		if defaults[i].Parameter == "secure_path" {
			securePath = &defaults[i]
			break
		}
	}
	require.NotNil(t, securePath, "secure_path default should exist")
	assert.Equal(t, "global", securePath.Scope)
	assert.Contains(t, securePath.Value, "/usr/local/sbin")
	assert.Equal(t, "=", securePath.Operation)

	// Find user-scoped default: Defaults:root !requiretty
	var rootDefault *sudoers.Default
	for i := range defaults {
		if defaults[i].Scope == "user" && defaults[i].Target == "root" {
			rootDefault = &defaults[i]
			break
		}
	}
	require.NotNil(t, rootDefault, "root user-scoped default should exist")
	assert.Equal(t, "requiretty", rootDefault.Parameter)
	assert.True(t, rootDefault.Negated)

	// Find host-scoped default: Defaults@webservers log_output
	var hostDefault *sudoers.Default
	for i := range defaults {
		if defaults[i].Scope == "host" && defaults[i].Target == "webservers" {
			hostDefault = &defaults[i]
			break
		}
	}
	require.NotNil(t, hostDefault, "webservers host-scoped default should exist")
	assert.Equal(t, "log_output", hostDefault.Parameter)
}

func TestSudoersParser_Aliases(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/linux.toml"))
	require.NoError(t, err)

	f, err := conn.FileSystem().Open("/etc/sudoers")
	require.NoError(t, err)
	defer f.Close()

	content, err := io.ReadAll(f)
	require.NoError(t, err)

	aliases := sudoers.ParseAliases("/etc/sudoers", string(content))
	require.NotEmpty(t, aliases)

	// Find Host_Alias WEBSERVERS
	var webserversAlias *sudoers.Alias
	for i := range aliases {
		if aliases[i].Name == "WEBSERVERS" {
			webserversAlias = &aliases[i]
			break
		}
	}
	require.NotNil(t, webserversAlias, "WEBSERVERS alias should exist")
	assert.Equal(t, "host", webserversAlias.Type)
	assert.Equal(t, []string{"www1", "www2", "www3"}, webserversAlias.Members)

	// Find User_Alias ADMINS
	var adminsAlias *sudoers.Alias
	for i := range aliases {
		if aliases[i].Name == "ADMINS" {
			adminsAlias = &aliases[i]
			break
		}
	}
	require.NotNil(t, adminsAlias, "ADMINS alias should exist")
	assert.Equal(t, "user", adminsAlias.Type)
	assert.Equal(t, []string{"alice", "bob", "charlie"}, adminsAlias.Members)

	// Find Cmnd_Alias SHUTDOWN
	var shutdownAlias *sudoers.Alias
	for i := range aliases {
		if aliases[i].Name == "SHUTDOWN" {
			shutdownAlias = &aliases[i]
			break
		}
	}
	require.NotNil(t, shutdownAlias, "SHUTDOWN alias should exist")
	assert.Equal(t, "cmnd", shutdownAlias.Type)
	assert.Contains(t, shutdownAlias.Members, "/sbin/halt")
	assert.Contains(t, shutdownAlias.Members, "/sbin/shutdown")
	assert.Contains(t, shutdownAlias.Members, "/sbin/reboot")

	// Find Runas_Alias DBA
	var dbaAlias *sudoers.Alias
	for i := range aliases {
		if aliases[i].Name == "DBA" {
			dbaAlias = &aliases[i]
			break
		}
	}
	require.NotNil(t, dbaAlias, "DBA alias should exist")
	assert.Equal(t, "runas", dbaAlias.Type)
	assert.Equal(t, []string{"postgres", "mysql"}, dbaAlias.Members)
}

func TestSudoersParser_DropInFile(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/linux.toml"))
	require.NoError(t, err)

	f, err := conn.FileSystem().Open("/etc/sudoers.d/10-developers")
	require.NoError(t, err)
	defer f.Close()

	content, err := io.ReadAll(f)
	require.NoError(t, err)

	specs := sudoers.ParseUserSpecs("/etc/sudoers.d/10-developers", string(content))
	require.Len(t, specs, 2)

	// %developers ALL=(ALL) NOPASSWD: /usr/bin/docker, /usr/bin/docker-compose
	assert.Equal(t, "/etc/sudoers.d/10-developers", specs[0].File)
	assert.Equal(t, []string{"%developers"}, specs[0].Users)
	assert.Equal(t, []string{"NOPASSWD"}, specs[0].Tags)
	assert.Contains(t, specs[0].Commands, "/usr/bin/docker")
	assert.Contains(t, specs[0].Commands, "/usr/bin/docker-compose")

	// developer1 ALL=(www-data) NOPASSWD: ALL
	assert.Equal(t, []string{"developer1"}, specs[1].Users)
	assert.Equal(t, []string{"www-data"}, specs[1].RunasUsers)
	assert.Equal(t, []string{"ALL"}, specs[1].Commands)
}

func TestSudoersParser_MonitoringDropIn(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/linux.toml"))
	require.NoError(t, err)

	f, err := conn.FileSystem().Open("/etc/sudoers.d/99-monitoring")
	require.NoError(t, err)
	defer f.Close()

	content, err := io.ReadAll(f)
	require.NoError(t, err)

	specs := sudoers.ParseUserSpecs("/etc/sudoers.d/99-monitoring", string(content))
	require.Len(t, specs, 2)

	// nagios ALL=(root) NOPASSWD: /usr/lib/nagios/plugins/*, /usr/bin/systemctl status *
	assert.Equal(t, []string{"nagios"}, specs[0].Users)
	assert.Equal(t, []string{"root"}, specs[0].RunasUsers)
	assert.Equal(t, []string{"NOPASSWD"}, specs[0].Tags)

	// zabbix ALL=(root) NOPASSWD: /usr/sbin/dmidecode, /usr/bin/lsof
	assert.Equal(t, []string{"zabbix"}, specs[1].Users)
	assert.Contains(t, specs[1].Commands, "/usr/sbin/dmidecode")
	assert.Contains(t, specs[1].Commands, "/usr/bin/lsof")
}

func TestSudoersParser_DirectoryExists(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/linux.toml"))
	require.NoError(t, err)

	stat, err := conn.FileSystem().Stat("/etc/sudoers.d")
	require.NoError(t, err)
	assert.True(t, stat.IsDir())
}
