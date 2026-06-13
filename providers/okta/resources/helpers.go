// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

// The v5 Okta SDK is OpenAPI-generated and models almost every scalar as a
// pointer so it can distinguish "unset" from a zero value. The previous v2
// types were plain values, so these helpers dereference v5 pointers back to the
// zero-value semantics the resource mappers (and existing MQL queries) expect.

func oktaStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func oktaBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
