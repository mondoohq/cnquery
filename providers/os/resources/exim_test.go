// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseEximConfig(t *testing.T) {
	t.Run("Debian split-config key='value' form", func(t *testing.T) {
		content := "# update-exim4.conf.conf\n" +
			"dc_eximconfig_configtype='internet'\n" +
			"dc_local_interfaces='127.0.0.1 ; ::1'\n" +
			"dc_readhost=''\n"
		got := parseEximConfig(content)
		assert.Equal(t, "internet", got["dc_eximconfig_configtype"])
		assert.Equal(t, "127.0.0.1 ; ::1", got["dc_local_interfaces"])
		assert.Equal(t, "", got["dc_readhost"])
	})

	t.Run("monolithic key = value form with macros", func(t *testing.T) {
		content := "# exim.conf\n" +
			"MACRO = /var/spool/exim\n" +
			"local_interfaces = <; 127.0.0.1 ; ::1\n" +
			"primary_hostname = mail.example.com\n"
		got := parseEximConfig(content)
		assert.Equal(t, "/var/spool/exim", got["MACRO"])
		assert.Equal(t, "<; 127.0.0.1 ; ::1", got["local_interfaces"])
		assert.Equal(t, "mail.example.com", got["primary_hostname"])
	})

	t.Run("stops at the first begin block", func(t *testing.T) {
		content := "primary_hostname = mail.example.com\n" +
			"begin acl\n" +
			"acl_check_rcpt:\n" +
			"  accept hosts = +relay = something\n"
		got := parseEximConfig(content)
		assert.Equal(t, "mail.example.com", got["primary_hostname"])
		_, hasAcl := got["acl_check_rcpt:"]
		assert.False(t, hasAcl)
		// an option-looking line inside the section must not leak in
		_, leaked := got["accept hosts"]
		assert.False(t, leaked)
	})

	t.Run("hide prefix and named lists", func(t *testing.T) {
		content := "hide mysql_servers = localhost/db/user/secret\n" +
			"domainlist local_domains = @\n"
		got := parseEximConfig(content)
		assert.Equal(t, "localhost/db/user/secret", got["mysql_servers"])
		// named-list left-hand side has two tokens; it is not a simple option
		_, hasNamedList := got["domainlist local_domains"]
		assert.False(t, hasNamedList)
	})

	t.Run("backslash continuation", func(t *testing.T) {
		content := "local_interfaces = <; 127.0.0.1 ; \\\n  ::1\n"
		got := parseEximConfig(content)
		assert.Equal(t, "<; 127.0.0.1 ; ::1", got["local_interfaces"])
	})
}

func TestUnquoteEximValue(t *testing.T) {
	assert.Equal(t, "127.0.0.1 ; ::1", unquoteEximValue("'127.0.0.1 ; ::1'"))
	assert.Equal(t, "internet", unquoteEximValue(`"internet"`))
	assert.Equal(t, "", unquoteEximValue("''"))
	assert.Equal(t, "unquoted", unquoteEximValue("unquoted"))
	assert.Equal(t, "'mismatched", unquoteEximValue("'mismatched"))
}

func TestParseEximListValue(t *testing.T) {
	// custom separator via the <sep prefix (needed for IPv6)
	assert.Equal(t, []any{"127.0.0.1", "::1"}, parseEximListValue("<; 127.0.0.1 ; ::1"))
	// default ':' separator
	assert.Equal(t, []any{"127.0.0.1", "192.168.0.1"}, parseEximListValue("127.0.0.1 : 192.168.0.1"))
	assert.Equal(t, []any{"127.0.0.1"}, parseEximListValue("127.0.0.1"))
}

func TestSplitEximList(t *testing.T) {
	assert.Equal(t, []any{"127.0.0.1", "::1"}, splitEximList("127.0.0.1 ; ::1", ";"))
	assert.Equal(t, []any{}, splitEximList("", ";"))
}
