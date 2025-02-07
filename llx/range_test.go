// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRange(t *testing.T) {
	t.Run("single line", func(t *testing.T) {
		r := RangePrimitive(NewRange().AddLine(12))
		assert.Equal(t, "12", r.LabelV2(nil))
	})

	t.Run("line range", func(t *testing.T) {
		r := RangePrimitive(NewRange().AddLineRange(12, 18))
		assert.Equal(t, "12-18", r.LabelV2(nil))
	})

	t.Run("long line range", func(t *testing.T) {
		r := RangePrimitive(NewRange().AddLineRange(1, 1234567))
		assert.Equal(t, "1-1234567", r.LabelV2(nil))
	})

	t.Run("column range", func(t *testing.T) {
		r := RangePrimitive(NewRange().AddColumnRange(12, 1, 28))
		assert.Equal(t, "12:1-28", r.LabelV2(nil))
	})

	t.Run("line and column range", func(t *testing.T) {
		r := RangePrimitive(NewRange().AddLineColumnRange(12, 12345678, 3, 1234567))
		assert.Equal(t, "12:3-12345678:1234567", r.LabelV2(nil))
	})
}
