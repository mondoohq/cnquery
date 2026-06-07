// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPamConfServiceEntryParams(t *testing.T) {
	se := &mqlPamConfServiceEntry{}

	t.Run("key=value and bare flags", func(t *testing.T) {
		got, err := se.params([]any{"use_uid", "group=wheel"})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"use_uid": "", "group": "wheel"}, got)
	})

	t.Run("duplicate keys: last occurrence wins", func(t *testing.T) {
		got, err := se.params([]any{"group=wheel", "group=admin"})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"group": "admin"}, got)
	})

	t.Run("no options yields an empty map", func(t *testing.T) {
		got, err := se.params([]any{})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{}, got)
	})
}
