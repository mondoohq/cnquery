// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package filteropts provides shared helpers for parsing the raw --filters
// key/value options that providers use to narrow discovery.
package filteropts

import "strings"

// ParseCsvSliceOpt returns the comma-separated values for the given key as a
// slice. Empty keys or values are skipped, and a non-nil empty slice is
// returned when there is nothing to parse.
//
// Example:
//
//	key  = "regions"
//	opts = {"regions": "us-east-1,us-west-2"}
//	returns []string{"us-east-1", "us-west-2"}
func ParseCsvSliceOpt(opts map[string]string, key string) []string {
	res := []string{}
	for k, v := range opts {
		if k == "" || v == "" {
			continue
		}
		if k == key {
			res = append(res, strings.Split(v, ",")...)
		}
	}
	return res
}
