// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package slicesx

// IsSubsetOf returns true if every element in sub exists in super.
func IsSubsetOf[T comparable](sub, super []T) bool {
	if len(sub) > len(super) {
		return false
	}
	set := make(map[T]struct{}, len(super))
	for _, v := range super {
		set[v] = struct{}{}
	}
	for _, v := range sub {
		if _, ok := set[v]; !ok {
			return false
		}
	}
	return true
}
