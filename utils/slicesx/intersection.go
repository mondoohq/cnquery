// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package slicesx

func Intersection[T any](a, b []T, compareFn func(T, T) bool) []T {
	intersection := []T{}
	for _, itemA := range a {
		for _, itemB := range b {
			if compareFn(itemA, itemB) {
				intersection = append(intersection, itemA)
				break
			}
		}
	}
	return intersection
}
