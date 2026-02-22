// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

func TestSudoersPathsForPlatform(t *testing.T) {
	tests := []struct {
		platform string
		expected []string
	}{
		{"freebsd", []string{"/usr/local/etc/sudoers"}},
		{"dragonflybsd", []string{"/usr/local/etc/sudoers"}},
		{"openbsd", []string{"/usr/local/etc/sudoers"}},
		{"netbsd", []string{"/usr/pkg/etc/sudoers"}},
		{"aix", []string{"/opt/freeware/etc/sudoers"}},
		{"debian", []string{"/etc/sudoers"}},
		{"ubuntu", []string{"/etc/sudoers"}},
		{"redhat", []string{"/etc/sudoers"}},
		{"macos", []string{"/etc/sudoers"}},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			assert.Equal(t, tt.expected, sudoersPathsForPlatform(connWithPlatform(tt.platform)))
		})
	}

	t.Run("nil platform", func(t *testing.T) {
		conn := &mockConn{asset: &inventory.Asset{}}
		assert.Equal(t, []string{"/etc/sudoers"}, sudoersPathsForPlatform(conn))
	})
}
