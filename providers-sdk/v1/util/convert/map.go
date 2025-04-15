// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package convert

// MapToInterfaceMap converts a map[string]T to map[string]any
func MapToInterfaceMap[T any](m map[string]T) map[string]any {
	res := make(map[string]any)
	for k, v := range m {
		res[k] = v
	}
	return res
}

func PtrMapStrToStr(data map[string]*string) map[string]string {
	m := make(map[string]string)
	for key := range data {
		if data[key] != nil {
			m[key] = *data[key]
		}
	}
	return m
}

func PtrMapStrToInterface(data map[string]*string) map[string]any {
	m := make(map[string]any)
	for key := range data {
		if data[key] != nil {
			m[key] = *data[key]
		}
	}
	return m
}
