// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResource_RsyslogConf(t *testing.T) {
	t.Run("files includes main conf and .d fragments", func(t *testing.T) {
		res := x.TestQuery(t, "rsyslog.conf.files.length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		assert.Equal(t, int64(2), res[0].Data.Value)
	})

	t.Run("content aggregates all files", func(t *testing.T) {
		res := x.TestQuery(t, "rsyslog.conf.content")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		content := res[0].Data.Value.(string)
		// Main conf content
		assert.Contains(t, content, "$ModLoad imuxsock")
		// Fragment content from rsyslog.d/50-default.conf
		assert.Contains(t, content, "kern.* /var/log/kern.log")
	})

	t.Run("settings strips comments and blanks", func(t *testing.T) {
		res := x.TestQuery(t, "rsyslog.conf.settings")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		settings := res[0].Data.Value.([]any)
		assert.Greater(t, len(settings), 0)
		// Verify comment-only and blank lines are excluded
		for _, s := range settings {
			line := s.(string)
			assert.NotEmpty(t, line)
			assert.NotEqual(t, "#", string(line[0]), "settings should not contain comment lines")
		}
	})

	t.Run("settings contains expected directives", func(t *testing.T) {
		res := x.TestQuery(t, "rsyslog.conf.settings")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		settings := res[0].Data.Value.([]any)
		// Check for a known directive from the main conf
		found := false
		for _, s := range settings {
			if s.(string) == "$ModLoad imuxsock" {
				found = true
				break
			}
		}
		assert.True(t, found, "settings should contain '$ModLoad imuxsock'")
	})

	t.Run("path returns default", func(t *testing.T) {
		res := x.TestQuery(t, "rsyslog.conf.path")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		assert.Equal(t, "/etc/rsyslog.conf", res[0].Data.Value)
	})
}
