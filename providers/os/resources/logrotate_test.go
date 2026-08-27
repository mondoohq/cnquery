// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// A bare `logrotate.entry` leaves `file` unset, so id() must report the missing
// file rather than dereferencing a nil *mqlFile.
func TestLogrotateEntryID(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		id, err := (&mqlLogrotateEntry{}).id()
		require.Error(t, err)
		assert.Empty(t, id)
		assert.Contains(t, err.Error(), "missing file")
	})

	t.Run("with file", func(t *testing.T) {
		f := &mqlFile{}
		f.Path.Data = "/etc/logrotate.conf"
		f.Path.State = plugin.StateIsSet

		e := &mqlLogrotateEntry{}
		e.File.Data = f
		e.File.State = plugin.StateIsSet
		e.LineNumber.Data = 7
		e.LineNumber.State = plugin.StateIsSet
		e.Path.Data = "/var/log/syslog"
		e.Path.State = plugin.StateIsSet

		id, err := e.id()
		require.NoError(t, err)
		assert.Equal(t, "/etc/logrotate.conf:7:/var/log/syslog", id)
	})
}
