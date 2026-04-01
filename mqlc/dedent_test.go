// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package mqlc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDedent(t *testing.T) {
	content := "    a\n  b\n"
	assert.Equal(t, "  a\nb\n", Dedent(content))
}
