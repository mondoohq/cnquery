// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// content() must skip files it cannot read instead of swallowing the error and
// appending an empty block for them.
func TestRsyslogConfContentSkipsUnreadableFiles(t *testing.T) {
	good := &mqlFile{}
	good.Content = plugin.TValue[string]{Data: "good content", State: plugin.StateIsSet}

	unreadable := &mqlFile{}
	unreadable.Content = plugin.TValue[string]{
		Error: errors.New("permission denied"),
		State: plugin.StateIsSet | plugin.StateIsNull,
	}

	s := &mqlRsyslogConf{}
	out, err := s.content([]any{unreadable, good})
	require.NoError(t, err)

	// The unreadable file is skipped entirely; only the readable file's content
	// is emitted (previously a spurious leading blank line was appended).
	assert.Equal(t, "good content\n", out)
}
