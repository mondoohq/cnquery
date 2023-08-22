// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stringx

func Intersection(a, b []string) []string {
	entriesMap := map[string]struct{}{}
	res := []string{}

	for i := range a {
		entriesMap[a[i]] = struct{}{}
	}

	for i := range b {
		if _, ok := entriesMap[b[i]]; ok {
			res = append(res, b[i])
		}
	}
	return res
}
