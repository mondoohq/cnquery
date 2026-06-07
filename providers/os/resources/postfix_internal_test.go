// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

func TestPostfixConfigDir(t *testing.T) {
	tests := []struct {
		platform string
		expected string
	}{
		{"debian", "/etc/postfix"},
		{"ubuntu", "/etc/postfix"},
		{"redhat", "/etc/postfix"},
		{"macos", "/etc/postfix"},
		{"freebsd", "/usr/local/etc/postfix"},
		{"dragonflybsd", "/usr/local/etc/postfix"},
		{"netbsd", "/usr/pkg/etc/postfix"},
	}
	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			assert.Equal(t, tt.expected, postfixConfigDir(connWithPlatform(tt.platform)))
		})
	}

	t.Run("nil platform falls back to /etc/postfix", func(t *testing.T) {
		assert.Equal(t, "/etc/postfix", postfixConfigDir(&mockConn{asset: &inventory.Asset{}}))
	})
}

func TestParsePostconf(t *testing.T) {
	out := "inet_interfaces = loopback-only\n" +
		"myhostname = mail.example.com\n" +
		"smtpd_banner = $myhostname ESMTP\n" +
		"# not a real line\n"
	got := parsePostconf(out)
	assert.Equal(t, "loopback-only", got["inet_interfaces"])
	assert.Equal(t, "mail.example.com", got["myhostname"])
	// postconf already expands $vars, so we keep its output verbatim
	assert.Equal(t, "$myhostname ESMTP", got["smtpd_banner"])
}

func TestParsePostfixMainCf(t *testing.T) {
	t.Run("key=value, comments and blank lines", func(t *testing.T) {
		content := "# main.cf\n\ninet_interfaces = localhost\nmydestination = example.com\n"
		got := parsePostfixMainCf(content)
		assert.Equal(t, map[string]any{
			"inet_interfaces": "localhost",
			"mydestination":   "example.com",
		}, got)
	})

	t.Run("continuation lines fold into the previous value", func(t *testing.T) {
		content := "mynetworks = 127.0.0.0/8\n  [::1]/128\n"
		got := parsePostfixMainCf(content)
		assert.Equal(t, "127.0.0.0/8 [::1]/128", got["mynetworks"])
	})

	t.Run("$name interpolation against file values", func(t *testing.T) {
		content := "myhostname = mail.example.com\nmyorigin = $myhostname\nbanner = ${myhostname} ESMTP\n"
		got := parsePostfixMainCf(content)
		assert.Equal(t, "mail.example.com", got["myorigin"])
		assert.Equal(t, "mail.example.com ESMTP", got["banner"])
	})

	t.Run("unknown variables are left untouched", func(t *testing.T) {
		content := "relayhost = $unset_var\n"
		got := parsePostfixMainCf(content)
		assert.Equal(t, "$unset_var", got["relayhost"])
	})

	t.Run("a blank line breaks continuation", func(t *testing.T) {
		// the indented line follows a blank line, so it must NOT fold into
		// mynetworks (a blank line terminates the logical line in Postfix)
		content := "mynetworks = 127.0.0.0/8\n\n  [::1]/128\n"
		got := parsePostfixMainCf(content)
		assert.Equal(t, "127.0.0.0/8", got["mynetworks"])
	})
}

func TestSplitPostfixList(t *testing.T) {
	assert.Equal(t, []any{"127.0.0.1", "::1"}, splitPostfixList("127.0.0.1, ::1"))
	assert.Equal(t, []any{"127.0.0.1", "::1"}, splitPostfixList("127.0.0.1  ::1"))
	assert.Equal(t, []any{"localhost"}, splitPostfixList("localhost"))
	assert.Equal(t, []any{}, splitPostfixList(""))
}

func TestParseMasterCf(t *testing.T) {
	content := "# service type  private unpriv  chroot  wakeup  maxproc command\n" +
		"smtp      inet  n       -       y       -       -       smtpd\n" +
		"pickup    unix  n       -       y       60      1       pickup\n" +
		"submission inet n       -       y       -       -       smtpd\n" +
		"  -o syslog_name=postfix/submission\n" // continuation folds into command

	got := parseMasterCf(content)
	assert.Len(t, got, 3)

	assert.Equal(t, masterCfEntry{
		Service: "smtp", Type: "inet", Private: "n", Unprivileged: "-",
		Chroot: "y", Wakeup: "-", MaxProcesses: "-", Command: "smtpd",
	}, got[0])

	assert.Equal(t, "pickup", got[1].Command)
	assert.Equal(t, "60", got[1].Wakeup)

	// the continuation line is appended to the submission command
	assert.Equal(t, "smtpd -o syslog_name=postfix/submission", got[2].Command)
}
