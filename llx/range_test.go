// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"fmt"
	"strconv"
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
		r := RangePrimitive(NewRange().AddLineColumnRange(12, 3, 12345678, 1234567))
		assert.Equal(t, "12:3-12345678:1234567", r.LabelV2(nil))
	})
}

var test3lines = `Line 1
Line 2, Col 14
Line 3, Col....17`

func TestExtractRange(t *testing.T) {
	t.Run("Lines", func(t *testing.T) {
		lines := []struct {
			line int
			res  string
		}{
			{0, ""},
			{1, "Line 1\n"},
			{2, "Line 2, Col 14\n"},
			{3, "Line 3, Col....17"},
			{4, ""},
		}
		for i := range lines {
			x := lines[i]
			t.Run("line "+strconv.Itoa(x.line), func(t *testing.T) {
				r := NewRange().AddLine(uint32(x.line))
				assert.Equal(t, x.res, r.ExtractString(test3lines, DefaultExtractConfig))
			})
		}
	})

	t.Run("LineRanges", func(t *testing.T) {
		lineRanges := []struct {
			start int
			end   int
			res   string
		}{
			// first line only
			{1, 1, "Line 1\n"},
			// zero line, zero columns
			{0, 1, "Line 1\n"},
			{0, 2, "Line 1\nLine 2, Col 14\n"},
			// multiple normal ranges
			{1, 2, "Line 1\nLine 2, Col 14\n"},
			{1, 3, "Line 1\nLine 2, Col 14\nLine 3, Col....17"},
			{1, 99, "Line 1\nLine 2, Col 14\nLine 3, Col....17"},
		}
		for i := range lineRanges {
			x := lineRanges[i]
			t.Run(fmt.Sprintf("line %d-%d", x.start, x.end), func(t *testing.T) {
				r := NewRange().AddLineRange(uint32(x.start), uint32(x.end))
				assert.Equal(t, x.res, r.ExtractString(test3lines, DefaultExtractConfig))
			})
		}
	})

	t.Run("LineColumnRanges", func(t *testing.T) {
		lineRanges := []struct {
			start int
			end   int
			col1  int
			col2  int
			res   string
		}{
			// first line only
			{1, 1, 1, 4, "Line"},
			{1, 1, 1, 6, "Line 1"},
			{1, 1, 1, 7, "Line 1\n"},
			{1, 1, 1, 99, "Line 1\n"},
			// zero line, zero columns
			{0, 1, 1, 4, "Line"},
			{0, 0, 1, 4, ""},
			{1, 1, 0, 4, "Line"},
			{1, 1, 0, 0, ""},
			{1, 1, 0, 1, "L"},
			{1, 1, 0, 99, "Line 1\n"},
			{0, 1, 1, 6, "Line 1"},
			{0, 1, 99, 6, "Line 1"},
			{0, 2, 1, 0, "Line 1\n"},
			{0, 99, 0, 0, "Line 1\nLine 2, Col 14\nLine 3, Col....17"},
			// multiple normal ranges
			{1, 2, 1, 0, "Line 1\n"},
			{1, 2, 1, 1, "Line 1\nL"},
			{1, 2, 1, 14, "Line 1\nLine 2, Col 14"},
			{1, 2, 1, 15, "Line 1\nLine 2, Col 14\n"},
			{1, 2, 1, 99, "Line 1\nLine 2, Col 14\n"},
			{1, 3, 1, 0, "Line 1\nLine 2, Col 14\n"},
			{1, 3, 1, 1, "Line 1\nLine 2, Col 14\nL"},
			{1, 3, 1, 17, "Line 1\nLine 2, Col 14\nLine 3, Col....17"},
			{1, 3, 1, 99, "Line 1\nLine 2, Col 14\nLine 3, Col....17"},
			{1, 99, 0, 0, "Line 1\nLine 2, Col 14\nLine 3, Col....17"},
			{1, 99, 1, 99, "Line 1\nLine 2, Col 14\nLine 3, Col....17"},
		}
		for i := range lineRanges {
			x := lineRanges[i]
			t.Run(fmt.Sprintf("line %d:%d-%d:%d", x.start, x.col1, x.end, x.col2), func(t *testing.T) {
				r := NewRange().AddLineColumnRange(uint32(x.start), uint32(x.end), uint32(x.col1), uint32(x.col2))
				assert.Equal(t, x.res, r.ExtractString(test3lines, DefaultExtractConfig))
			})
		}
	})
}
