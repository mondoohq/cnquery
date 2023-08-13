// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package convert

func SliceAnyToInterface[T any](s []T) []interface{} {
	res := make([]interface{}, len(s))
	for i, v := range s {
		res[i] = v
	}
	return res
}

func SliceStrPtrToStr(s []*string) []string {
	res := make([]string, len(s))
	for i, v := range s {
		res[i] = *v
	}
	return res
}

func SliceStrPtrToInterface(s []*string) []interface{} {
	res := make([]interface{}, len(s))
	for i, v := range s {
		res[i] = *v
	}
	return res
}
