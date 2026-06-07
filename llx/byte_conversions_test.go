// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBytes2time(t *testing.T) {
	t.Run("round-trips a value encoded by TimePrimitive", func(t *testing.T) {
		want := time.Unix(1717689600, 123)
		got := bytes2time(TimePrimitive(&want).Value)
		assert.Equal(t, want.UnixNano(), got.UnixNano())
	})

	t.Run("falls back to zero time on a short buffer", func(t *testing.T) {
		// A valid encoding is 12 bytes; shorter buffers must not panic on the
		// b[0:8]/b[8:12] slices but fall back to the zero time.
		for n := range 12 {
			require.NotPanics(t, func() {
				assert.Equal(t, time.Unix(0, 0), bytes2time(make([]byte, n)), "length %d", n)
			})
		}
	})
}
