// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package mqlx

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToPrimitiveUintOverflow(t *testing.T) {
	// uint64 within int64 range converts fine.
	p, err := ToPrimitive(uint64(42))
	require.NoError(t, err)
	assert.Equal(t, int64(42), p.RawData().Value)

	// uint64 above math.MaxInt64 must error rather than silently wrap.
	_, err = ToPrimitive(uint64(math.MaxInt64) + 1)
	require.ErrorContains(t, err, "overflows int64")

	// uint (64-bit on most platforms) is guarded the same way.
	_, err = ToPrimitive(uint(math.MaxUint64))
	require.ErrorContains(t, err, "overflows int64")
}
