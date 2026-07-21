// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

func TestRsyslogConfPath(t *testing.T) {
	tests := []struct {
		platform string
		expected string
	}{
		{"freebsd", "/usr/local/etc/rsyslog.conf"},
		{"dragonflybsd", "/usr/local/etc/rsyslog.conf"},
		{"openbsd", "/usr/local/etc/rsyslog.conf"},
		{"netbsd", "/usr/pkg/etc/rsyslog.conf"},
		{"debian", "/etc/rsyslog.conf"},
		{"ubuntu", "/etc/rsyslog.conf"},
		{"redhat", "/etc/rsyslog.conf"},
		{"macos", "/etc/rsyslog.conf"},
		{"aix", "/etc/rsyslog.conf"},
		{"solaris", "/etc/rsyslog.conf"},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			assert.Equal(t, tt.expected, rsyslogConfPath(connWithPlatform(tt.platform)))
		})
	}

	t.Run("nil platform", func(t *testing.T) {
		conn := &mockConn{asset: &inventory.Asset{}}
		assert.Equal(t, "/etc/rsyslog.conf", rsyslogConfPath(conn))
	})
}

func TestStripRsyslogComment(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no comment", "$ModLoad imuxsock", "$ModLoad imuxsock"},
		{"trailing comment", "$ModLoad imuxsock # load unix socket input", "$ModLoad imuxsock "},
		{"whole line comment", "# this is a comment", ""},
		{"comment in double-quoted string is preserved", `$Template foo,"hash#tag"`, `$Template foo,"hash#tag"`},
		{"comment in single-quoted string is preserved", `$Template foo,'hash#tag'`, `$Template foo,'hash#tag'`},
		{"comment after closing quote is stripped", `$Template foo,"value" # comment`, `$Template foo,"value" `},
		{"blank line", "", ""},
		{"escaped hash is NOT special (rsyslog rule, not shell)", `key=val#after`, `key=val`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, stripRsyslogComment(tt.in))
		})
	}
}

func TestRsyslogConfParams(t *testing.T) {
	s := &mqlRsyslogConf{}

	content := strings.Join([]string{
		"# rsyslog configuration",
		"$FileCreateMode 0640",
		"$FileOwner   syslog",  // extra spacing is trimmed
		"$DirCreateMode\t0755", // tab separator
		"$FileGroup adm # inline comment is stripped",
		"module(load=\"imuxsock\")", // modern syntax is ignored
		"*.info /var/log/messages",  // selector lines are ignored
		"$FileCreateMode 0600",      // duplicate: last occurrence wins
		"$ActionResumeRetryCount",   // bare directive with no value is skipped
	}, "\n")

	got, err := s.params(content)
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{
		"FileCreateMode": "0600",
		"FileOwner":      "syslog",
		"DirCreateMode":  "0755",
		"FileGroup":      "adm",
	}, got)
}

func TestParseRsyslogIncludes(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "empty",
			content: "",
			want:    nil,
		},
		{
			name:    "only directives unrelated to includes",
			content: "$ModLoad imuxsock\n$ActionFileDefaultTemplate foo\n",
			want:    nil,
		},
		{
			name:    "legacy $IncludeConfig with glob",
			content: "$IncludeConfig /etc/rsyslog.d/*.conf\n",
			want:    []string{"/etc/rsyslog.d/*.conf"},
		},
		{
			name:    "legacy directive is case-insensitive",
			content: "$includeconfig /etc/rsyslog.d/a.conf\n$INCLUDECONFIG /etc/rsyslog.d/b.conf\n",
			want:    []string{"/etc/rsyslog.d/a.conf", "/etc/rsyslog.d/b.conf"},
		},
		{
			name:    "legacy with trailing inline comment",
			content: "$IncludeConfig /etc/rsyslog.d/*.conf # load fragments\n",
			want:    []string{"/etc/rsyslog.d/*.conf"},
		},
		{
			name:    "legacy with quoted value",
			content: `$IncludeConfig "/etc/rsyslog.d/*.conf"` + "\n",
			want:    []string{"/etc/rsyslog.d/*.conf"},
		},
		{
			name:    "modern include() with file=",
			content: `include(file="/etc/rsyslog.d/*.conf")` + "\n",
			want:    []string{"/etc/rsyslog.d/*.conf"},
		},
		{
			name:    "modern include() with extra params",
			content: `include(file="/etc/rsyslog.d/*.conf" mode="optional")` + "\n",
			want:    []string{"/etc/rsyslog.d/*.conf"},
		},
		{
			name:    "modern include() with single-quoted value",
			content: `include(file='/etc/rsyslog.d/*.conf')` + "\n",
			want:    []string{"/etc/rsyslog.d/*.conf"},
		},
		{
			name:    "modern include() with unquoted value",
			content: `include(file=/etc/rsyslog.d/local.conf)` + "\n",
			want:    []string{"/etc/rsyslog.d/local.conf"},
		},
		{
			name:    "modern include() with text= is skipped",
			content: `include(text="ruleset(name=\"foo\") { /* ... */ }")` + "\n",
			want:    nil,
		},
		{
			name: "mixed legacy + modern + ignored directives",
			content: `# rsyslog.conf
$ModLoad imuxsock

$IncludeConfig /etc/rsyslog.d/00-local.conf
include(file="/etc/rsyslog.d/*.conf")
$ActionFileDefaultTemplate RSYSLOG_TraditionalFileFormat
`,
			want: []string{"/etc/rsyslog.d/00-local.conf", "/etc/rsyslog.d/*.conf"},
		},
		{
			name: "duplicates collapse, source order preserved",
			content: `$IncludeConfig /a.conf
$IncludeConfig /b.conf
$IncludeConfig /a.conf
include(file="/b.conf")
`,
			want: []string{"/a.conf", "/b.conf"},
		},
		{
			name:    "false positive guard: $IncludeConfigSomething is not a match",
			content: "$IncludeConfigSomething /tmp/x.conf\n",
			want:    nil,
		},
		{
			name:    "false positive guard: includes inside a comment are ignored",
			content: "# example: $IncludeConfig /etc/rsyslog.d/*.conf\n",
			want:    nil,
		},
		{
			name:    "comment inside quoted include arg is preserved",
			content: `include(file="/etc/rsyslog.d/has#hash.conf")` + "\n",
			want:    []string{"/etc/rsyslog.d/has#hash.conf"},
		},
		{
			name: "modern include() multi-line block (Ansible-style)",
			content: `include(
    file="/etc/rsyslog.d/*.conf"
)
`,
			want: []string{"/etc/rsyslog.d/*.conf"},
		},
		{
			name: "modern include() multi-line with mode after",
			content: `include(
    file="/etc/rsyslog.d/*.conf"
    mode="optional"
)
`,
			want: []string{"/etc/rsyslog.d/*.conf"},
		},
		{
			name: "modern include() opens and closes mid-line",
			content: `include( file="/etc/rsyslog.d/a.conf" )
include(file="/etc/rsyslog.d/b.conf"
)
`,
			want: []string{"/etc/rsyslog.d/a.conf", "/etc/rsyslog.d/b.conf"},
		},
		{
			// Unterminated blocks have no closing `)`, so the anchored
			// regex won't match. Returning nil is correct — rsyslog itself
			// would reject this config at load time. We surface nothing
			// rather than guessing at a partial parse.
			name: "unterminated include() block returns nothing",
			content: `include(
    file="/etc/rsyslog.d/orphan.conf"
`,
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRsyslogIncludes(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCoalesceIncludeBlocks(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "blank lines outside blocks are dropped",
			in:   "$ModLoad imuxsock\n\n\n$IncludeConfig /etc/rsyslog.d/x.conf\n",
			want: []string{"$ModLoad imuxsock", "$IncludeConfig /etc/rsyslog.d/x.conf"},
		},
		{
			name: "comments are stripped before coalescing",
			in:   "include( # opens\n  file=\"/a.conf\" # path\n) # closes\n",
			want: []string{`include( file="/a.conf" )`},
		},
		{
			name: "parens inside quotes do not affect block tracking",
			in:   `include(file="/a.conf"  text=")")` + "\n",
			want: []string{`include(file="/a.conf"  text=")")`},
		},
		{
			name: "non-include line with stray paren is not coalesced",
			in:   "$Template foo,\"(literal)\"\n$IncludeConfig /a.conf\n",
			want: []string{`$Template foo,"(literal)"`, "$IncludeConfig /a.conf"},
		},
		{
			name: "blank lines INSIDE a block are kept as separators",
			in:   "include(\n\n    file=\"/a.conf\"\n\n)\n",
			want: []string{`include(  file="/a.conf"  )`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := coalesceIncludeBlocks(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCountUnquotedParens(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"include(", 1},
		{`include(file="/a")`, 0},
		{`include(file="/a"`, 1},
		{"))))", -4},
		{`"()()"`, 0},
		{`'()'`, 0},
		{`(text=")")`, 0},
		{`(text="(")`, 0},
		{`'(' "(" (`, 1}, // only the third `(` is unquoted
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, countUnquotedParens(tt.in))
		})
	}
}

func TestResolveRsyslogInclude(t *testing.T) {
	tests := []struct {
		name      string
		parentDir string
		pattern   string
		wantDir   string
		wantGlob  string
	}{
		{
			name:      "absolute path with glob",
			parentDir: "/etc",
			pattern:   "/etc/rsyslog.d/*.conf",
			wantDir:   "/etc/rsyslog.d",
			wantGlob:  "*.conf",
		},
		{
			name:      "absolute path with no glob",
			parentDir: "/etc",
			pattern:   "/etc/rsyslog.d/00-local.conf",
			wantDir:   "/etc/rsyslog.d",
			wantGlob:  "00-local.conf",
		},
		{
			name:      "relative path is anchored at parent dir",
			parentDir: "/etc/rsyslog.d",
			pattern:   "local.conf",
			wantDir:   "/etc/rsyslog.d",
			wantGlob:  "local.conf",
		},
		{
			name:      "relative path with subdir",
			parentDir: "/etc/rsyslog.d",
			pattern:   "extras/local.conf",
			wantDir:   "/etc/rsyslog.d/extras",
			wantGlob:  "local.conf",
		},
		{
			name:      "relative path with glob",
			parentDir: "/etc/rsyslog.d",
			pattern:   "extras/*.conf",
			wantDir:   "/etc/rsyslog.d/extras",
			wantGlob:  "*.conf",
		},
		{
			name:      "glob metacharacters are left intact for the matcher",
			parentDir: "/etc",
			pattern:   "/etc/rsyslog.d/[0-9]?-local.conf",
			wantDir:   "/etc/rsyslog.d",
			wantGlob:  "[0-9]?-local.conf",
		},
		{
			name:      "parent traversal is cleaned",
			parentDir: "/etc/rsyslog.d",
			pattern:   "../rsyslog.extra/*.conf",
			wantDir:   "/etc/rsyslog.extra",
			wantGlob:  "*.conf",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, glob := resolveRsyslogInclude(tt.parentDir, tt.pattern)
			assert.Equal(t, tt.wantDir, dir)
			assert.Equal(t, tt.wantGlob, glob)
		})
	}
}
