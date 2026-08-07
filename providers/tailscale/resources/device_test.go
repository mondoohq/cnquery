// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tsclient "tailscale.com/client/tailscale/v2"
)

func TestLastSeenTime(t *testing.T) {
	t.Run("nil while connected to control", func(t *testing.T) {
		// Tailscale omits lastSeen for a device that currently holds a
		// connection. Reporting null keeps a dormant-device query from
		// matching a device that is online right now.
		assert.Nil(t, lastSeenTime(nil))
	})

	t.Run("returns the underlying instant", func(t *testing.T) {
		want := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
		got := lastSeenTime(&tsclient.Time{Time: want})
		require.NotNil(t, got)
		assert.Equal(t, want, *got)
	})

	t.Run("zero time is carried through, not treated as absent", func(t *testing.T) {
		got := lastSeenTime(&tsclient.Time{})
		require.NotNil(t, got)
		assert.True(t, got.IsZero())
	})
}
