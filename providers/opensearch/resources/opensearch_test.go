// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "testing"

func TestToStringSlice(t *testing.T) {
	got := toStringSlice([]string{"a", "b"})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("toStringSlice = %v", got)
	}
	// Never nil, so llx.ArrayData receives a real slice.
	if got := toStringSlice(nil); got == nil || len(got) != 0 {
		t.Errorf("toStringSlice(nil) = %v, want empty non-nil", got)
	}
}
