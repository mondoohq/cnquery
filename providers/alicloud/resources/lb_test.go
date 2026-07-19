// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLbStrHelpers covers the ALB/NLB slice-flattening helpers that build the
// typed security-group and vSwitch cross-references, ensuring nil and empty
// entries are dropped so a resolver is never handed a blank id.
func TestLbStrHelpers(t *testing.T) {
	t.Run("albStrList drops nil and empty", func(t *testing.T) {
		assert.Equal(t, []string{}, albStrList(nil))
		assert.Equal(t, []string{"sg-a", "sg-b"}, albStrList([]*string{strp("sg-a"), strp(""), nil, strp("sg-b")}))
	})
	t.Run("nlbStrList drops nil and empty", func(t *testing.T) {
		assert.Equal(t, []string{}, nlbStrList(nil))
		assert.Equal(t, []string{"vsw-a"}, nlbStrList([]*string{nil, strp(""), strp("vsw-a")}))
	})
	t.Run("nlbStrSlice returns []any of non-empty strings", func(t *testing.T) {
		assert.Equal(t, []any{}, nlbStrSlice(nil))
		assert.Equal(t, []any{"cert-1", "cert-2"}, nlbStrSlice([]*string{strp("cert-1"), strp(""), strp("cert-2")}))
	})
}
