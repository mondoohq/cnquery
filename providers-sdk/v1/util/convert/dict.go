// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package convert

import "encoding/json"

// TODO: These functions are very heavyweight and prime candidates to
// be replaced by better alternatives.

// JsonToDict converts a raw golang object (typically loaded from JSON)
// into a `dict` type
func JsonToDict(v interface{}) (map[string]interface{}, error) {
	res := make(map[string]interface{})

	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(data), &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// JsonToDictSlice converts a raw golang object (typically loaded from JSON)
// into an array of `dict` types
func JsonToDictSlice(v interface{}) ([]interface{}, error) {
	res := []interface{}{}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(data), &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// DictToTypedMap converts a `dict` into a `map[string]T` safely.
// It discards anything that can't be converted to `T`.
func DictToTypedMap[T any](d interface{}) map[string]T {
	m := make(map[string]T)
	dict, ok := d.(map[string]interface{})
	if ok {
		for k, v := range dict {
			if t, ok := v.(T); ok {
				m[k] = t
			}
		}
	}
	return m
}
