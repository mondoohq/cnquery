// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"testing"

	"github.com/muesli/reflow/ansi"
	"github.com/stretchr/testify/assert"
)

func TestAssetName(t *testing.T) {
	data := []byte{77, 97, 110, 97, 103, 101, 100, 226, 128, 153, 115, 32, 86, 105, 114, 116, 117, 97, 108, 32, 77, 97, 99, 104, 105, 110, 101}
	str := string(data)
	assert.Equal(t, "Managed’s Virtual Machine", str)
	assert.True(t, len(RemoveNonASCII(str)) <= ansi.PrintableRuneWidth(str))
}
