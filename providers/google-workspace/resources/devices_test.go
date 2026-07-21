// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEpochMillisToTime(t *testing.T) {
	t.Run("zero is treated as unset", func(t *testing.T) {
		require.Nil(t, epochMillisToTime(0))
	})

	t.Run("negative is treated as unset", func(t *testing.T) {
		require.Nil(t, epochMillisToTime(-1))
	})

	t.Run("positive epoch-millis converts to the matching instant", func(t *testing.T) {
		// Chrome OS auto-update expiration is reported in epoch milliseconds.
		want := time.Date(2027, time.June, 1, 0, 0, 0, 0, time.UTC)
		got := epochMillisToTime(want.UnixMilli())
		require.NotNil(t, got)
		require.True(t, got.UTC().Equal(want), "expected %s, got %s", want, got.UTC())
	})

	t.Run("sub-second millis are preserved", func(t *testing.T) {
		got := epochMillisToTime(1_700_000_000_123)
		require.NotNil(t, got)
		require.Equal(t, int64(1_700_000_000_123), got.UnixMilli())
	})
}
