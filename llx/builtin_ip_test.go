// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateMask(t *testing.T) {
	tests := []struct {
		mask     int
		offset   int
		maxBytes int
		res      []byte
	}{
		{0, 0, 1, []byte{0x00}},
		{1, 0, 1, []byte{0x80}},
		{5, 0, 1, []byte{0xf8}},
		{8, 0, 1, []byte{0xff}},
		{9, 0, 1, []byte{0xff}},
		{9, 0, 2, []byte{0xff, 0x80}},
		{4, 4, 1, []byte{0x0f}},
		{7, 1, 1, []byte{0x7f}},
		{5, 3, 1, []byte{0x1f}},
		{6, 3, 2, []byte{0x1f, 0x80}},
		{6, 3, 1, []byte{0x1f}},
		{16, 48, 16, []byte{0, 0, 0, 0, 0, 0, 0xff, 0xff, 0, 0, 0, 0, 0, 0, 0, 0}},
	}

	for i := range tests {
		cur := tests[i]
		t.Run(fmt.Sprintf("bits=%d off=%d max=%d", cur.mask, cur.offset, cur.maxBytes), func(t *testing.T) {
			res := createMask(cur.mask, cur.offset, cur.maxBytes)
			assert.Equal(t, cur.res, res)
		})
	}
}
