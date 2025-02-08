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

var (
	longLine  = `1        10        20        30        40        50        60        70        80        90        100`
	shortLine = `0        A         B         C`
)

var test20lines = `Line 1



Line 5

` + longLine + `


Line 10
` + shortLine + `



Line 15




Line 20, done.`

func TestExtractRange(t *testing.T) {
	confAllContent := ExtractConfig{
		MaxLines:        999999,
		MaxColumns:      999999,
		ShowLineNumbers: false,
	}

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
				assert.Equal(t, x.res, r.ExtractString(test3lines, confAllContent))
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
				assert.Equal(t, x.res, r.ExtractString(test3lines, confAllContent))
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
			{0, 99, 0, 0, "Line 1\nLine 2, Col 14\nLine 3, Col....17"},
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
				assert.Equal(t, x.res, r.ExtractString(test3lines, confAllContent))
			})
		}
	})

	t.Run("Limited columns", func(t *testing.T) {
		maxCol := func(maxCol int) ExtractConfig {
			res := confAllContent
			res.MaxColumns = maxCol
			return res
		}

		t.Run("1 line, max 150 cols", func(t *testing.T) {
			r := NewRange().AddLine(uint32(7))
			assert.Equal(t, longLine+"\n", r.ExtractString(test20lines, maxCol(150)))
		})

		t.Run("1 line, max 103 cols", func(t *testing.T) {
			r := NewRange().AddLine(uint32(7))
			assert.Equal(t, longLine+"\n", r.ExtractString(test20lines, maxCol(103)))
		})

		t.Run("1 line max 102 cols", func(t *testing.T) {
			r := NewRange().AddLine(uint32(7))
			assert.Equal(t, longLine[0:98]+"...\n", r.ExtractString(test20lines, maxCol(102)))
		})

		t.Run("1 line max 5 cols", func(t *testing.T) {
			r := NewRange().AddLine(uint32(7))
			assert.Equal(t, longLine[0:1]+"...\n", r.ExtractString(test20lines, maxCol(5)))
		})

		t.Run("1 line max 1 cols", func(t *testing.T) {
			r := NewRange().AddLine(uint32(7))
			assert.Equal(t, "...\n", r.ExtractString(test20lines, maxCol(1)))
		})

		t.Run("1 linerange max 150 cols", func(t *testing.T) {
			r := NewRange().AddLineRange(uint32(7), uint32(8))
			assert.Equal(t, longLine+"\n\n", r.ExtractString(test20lines, maxCol(150)))
		})

		t.Run("1 linerange max 103 cols", func(t *testing.T) {
			r := NewRange().AddLineRange(uint32(7), uint32(8))
			assert.Equal(t, longLine+"\n\n", r.ExtractString(test20lines, maxCol(103)))
		})

		t.Run("1 linerange max 102 cols", func(t *testing.T) {
			r := NewRange().AddLineRange(uint32(7), uint32(8))
			assert.Equal(t, longLine[0:98]+"...\n\n", r.ExtractString(test20lines, maxCol(102)))
		})

		t.Run("1 linerange max 5 cols", func(t *testing.T) {
			r := NewRange().AddLineRange(uint32(7), uint32(8))
			assert.Equal(t, longLine[0:1]+"...\n\n", r.ExtractString(test20lines, maxCol(5)))
		})

		t.Run("1 linerange max 1 cols", func(t *testing.T) {
			r := NewRange().AddLineRange(uint32(7), uint32(8))
			assert.Equal(t, "...\n\n", r.ExtractString(test20lines, maxCol(1)))
		})

		t.Run("1 linerange max 150 cols", func(t *testing.T) {
			r := NewRange().AddLineColumnRange(uint32(7), uint32(8), 10, 1)
			assert.Equal(t, longLine[9:]+"\n\n", r.ExtractString(test20lines, maxCol(150)))
		})

		t.Run("1 linerange max 90 cols", func(t *testing.T) {
			r := NewRange().AddLineColumnRange(uint32(7), uint32(8), 10, 1)
			assert.Equal(t, longLine[9:95]+"...\n\n", r.ExtractString(test20lines, maxCol(90)))
		})

		t.Run("1 linerange max 50 cols (offset 10)", func(t *testing.T) {
			r := NewRange().AddLineColumnRange(uint32(6), uint32(7), 10, 1)
			assert.Equal(t, "1", r.ExtractString(test20lines, maxCol(50)))
		})

		t.Run("1 linerange max 50 cols (offset 1)", func(t *testing.T) {
			r := NewRange().AddLineColumnRange(uint32(6), uint32(7), 1, 150)
			assert.Equal(t, "\n"+longLine[:46]+"...\n", r.ExtractString(test20lines, maxCol(50)))
		})

		t.Run("1 linerange max 50 cols (offset 20)", func(t *testing.T) {
			r := NewRange().AddLineColumnRange(uint32(7), uint32(8), 20, 100)
			assert.Equal(t, longLine[19:65]+"...\n\n", r.ExtractString(test20lines, maxCol(50)))
		})
	})

	t.Run("Limited lines", func(t *testing.T) {
		maxLines := func(maxLines int) ExtractConfig {
			res := confAllContent
			res.MaxLines = maxLines
			return res
		}

		t.Run("30 max lines", func(t *testing.T) {
			r := NewRange().AddLineRange(1, 5)
			assert.Equal(t, "Line 1\n\n\n\nLine 5\n", r.ExtractString(test20lines, maxLines(30)))
		})

		t.Run("7 max lines (line range)", func(t *testing.T) {
			r := NewRange().AddLineRange(1, 20)
			assert.Equal(t, "Line 1\n\n\n...\n\n\nLine 20, done.", r.ExtractString(test20lines, maxLines(7)))
		})

		t.Run("7 max lines (line column range)", func(t *testing.T) {
			r := NewRange().AddLineColumnRange(1, 20, 1, 99)
			assert.Equal(t, "Line 1\n\n\n...\n\n\nLine 20, done.", r.ExtractString(test20lines, maxLines(7)))
		})
	})

	t.Run("Show line numbers", func(t *testing.T) {
		lineNumsCfg := confAllContent
		lineNumsCfg.ShowLineNumbers = true
		lineNumsCfg.LineNumberPadding = 1

		test3linesNumbered := "1: Line 1\n2: Line 2, Col 14\n3: Line 3, Col....17"

		t.Run("1 line", func(t *testing.T) {
			r := NewRange().AddLine(11)
			assert.Equal(t, "11: "+shortLine+"\n", r.ExtractString(test20lines, lineNumsCfg))
		})

		t.Run("3 lines", func(t *testing.T) {
			r := NewRange().AddLineRange(9, 11)
			assert.Equal(t, " 9: \n10: Line 10\n11: "+shortLine+"\n", r.ExtractString(test20lines, lineNumsCfg))
		})

		t.Run("3 lines, start to finish", func(t *testing.T) {
			r := NewRange().AddLineRange(1, 3)
			assert.Equal(t, test3linesNumbered, r.ExtractString(test3lines, lineNumsCfg))
		})

		t.Run("3 lines (colrange)", func(t *testing.T) {
			r := NewRange().AddLineColumnRange(9, 11, 0, 10)
			assert.Equal(t, " 9: \n10: Line 10\n11: "+shortLine[0:10], r.ExtractString(test20lines, lineNumsCfg))
		})

		t.Run("3 lines (colrange), start to finish", func(t *testing.T) {
			r := NewRange().AddLineColumnRange(1, 3, 0, 99)
			assert.Equal(t, test3linesNumbered, r.ExtractString(test3lines, lineNumsCfg))
		})
	})
}
