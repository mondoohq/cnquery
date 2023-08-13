// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package convert

// MapToInterfaceMap converts a map[string]T to map[string]interface{}
func MapToInterfaceMap[T any](m map[string]T) map[string]interface{} {
	res := make(map[string]interface{})
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

func PtrMapStrToInterface(data map[string]*string) map[string]interface{} {
	m := make(map[string]interface{})
	for key := range data {
		if data[key] != nil {
			m[key] = *data[key]
		}
	}
	return m
}
