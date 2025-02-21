// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mapx

// Merge merges two maps of type `map[K]T` giving preference to the first map.
func Merge[K comparable, V any](m1, m2 map[K]V) map[K]V {
	merged := make(map[K]V)

	// store all key:value's from the second map
	for key, value := range m2 {
		merged[key] = value
	}

	// iterate over the first map to give it preference
	for key, value := range m1 {
		merged[key] = value
	}

	return merged
}
