// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

// Pointer helpers shared across the resource tests.
//
// The OCI SDK models optional values as pointers, so a test that wants to
// distinguish "absent" from "the zero value" needs to hand a real pointer to
// the mapper. These live here rather than beside any one test so a new test
// does not have to invent a differently-named copy.

func intPtr(i int) *int       { return &i }
func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
func int64Ptr(i int64) *int64 { return &i }
