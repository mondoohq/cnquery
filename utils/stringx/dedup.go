// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stringx

func DedupStringArray(arr []string) []string {
	strMap := map[string]struct{}{}

	for i := range arr {
		strMap[arr[i]] = struct{}{}
	}

	rval := []string{}
	for p := range strMap {
		rval = append(rval, p)
	}
	return rval
}
