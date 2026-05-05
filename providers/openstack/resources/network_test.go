// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSegmentationID(t *testing.T) {
	t.Run("empty string returns 0", func(t *testing.T) {
		assert.Equal(t, int64(0), parseSegmentationID(""))
	})
	t.Run("decimal digits parse normally", func(t *testing.T) {
		assert.Equal(t, int64(0), parseSegmentationID("0"))
		assert.Equal(t, int64(1), parseSegmentationID("1"))
		assert.Equal(t, int64(100), parseSegmentationID("100"))
		assert.Equal(t, int64(4094), parseSegmentationID("4094"))
		assert.Equal(t, int64(16777215), parseSegmentationID("16777215"))
	})
	t.Run("non-digit characters return 0", func(t *testing.T) {
		assert.Equal(t, int64(0), parseSegmentationID("vlan-100"))
		assert.Equal(t, int64(0), parseSegmentationID("100x"))
		assert.Equal(t, int64(0), parseSegmentationID("+100"))
		assert.Equal(t, int64(0), parseSegmentationID("-100"))
		assert.Equal(t, int64(0), parseSegmentationID(" 100"))
	})
}
