// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "testing"

func TestDerefFloat(t *testing.T) {
	if got := derefFloat(nil); got != 0 {
		t.Fatalf("nil pointer: got %v, want 0", got)
	}
	f := 3.5
	if got := derefFloat(&f); got != 3.5 {
		t.Fatalf("got %v, want 3.5", got)
	}
}
