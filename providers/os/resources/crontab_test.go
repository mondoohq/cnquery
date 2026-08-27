// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// A bare `crontab.entry` leaves `file` unset, so id() must report the missing
// file rather than dereferencing a nil *mqlFile.
func TestCrontabEntryID(t *testing.T) {
	e := &mqlCrontabEntry{}
	// GetFile short-circuits on an already-resolved field, so mark it set with a
	// nil value -- the shape the runtime produces for a bare entry.
	e.File.State = plugin.StateIsSet | plugin.StateIsNull

	id, err := e.id()
	require.Error(t, err)
	assert.Empty(t, id)
	assert.Contains(t, err.Error(), "missing file")
}
