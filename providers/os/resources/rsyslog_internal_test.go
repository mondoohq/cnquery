// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
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
