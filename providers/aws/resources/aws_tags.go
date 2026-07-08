// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
)

// tagsToStringMap converts a slice of AWS SDK tag structs into a
// map[string]string, using the supplied key and value accessors. The accessors
// absorb the only variation across the ~100 AWS tag shapes: the field names
// (Key/Value vs KMS-style TagKey/TagValue).
//
// Nil policy: an entry whose key is nil is skipped (a tag key is never
// legitimately empty), while a nil value is normalized to the empty string so a
// tag key present on a resource is always represented.
func tagsToStringMap[T any](tags []T, key func(T) *string, value func(T) *string) map[string]string {
	m := make(map[string]string, len(tags))
	for i := range tags {
		k := key(tags[i])
		if k == nil {
			continue
		}
		m[*k] = convert.ToValue(value(tags[i]))
	}
	return m
}

// tagsToMap is tagsToStringMap with a map[string]any result, the shape MQL
// expects for tag fields. It builds the map directly rather than converting a
// map[string]string, avoiding a second allocation and copy.
func tagsToMap[T any](tags []T, key func(T) *string, value func(T) *string) map[string]any {
	m := make(map[string]any, len(tags))
	for i := range tags {
		k := key(tags[i])
		if k == nil {
			continue
		}
		m[*k] = convert.ToValue(value(tags[i]))
	}
	return m
}
