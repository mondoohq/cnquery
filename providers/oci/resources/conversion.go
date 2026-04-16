// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
)

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func boolValue(s *bool) bool {
	if s == nil {
		return false
	}
	return *s
}

func int64Value(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func intValue(i *int) int64 {
	if i == nil {
		return 0
	}
	return int64(*i)
}

// isOcid returns true if the string looks like a valid OCI resource identifier.
// OCI uses placeholder values like "ORACLE_MANAGED_KEY" for system-managed
// resources; those should not be resolved via init lookups.
func isOcid(s string) bool {
	return strings.HasPrefix(s, "ocid1.")
}

func jobErr(err error) []*jobpool.Job {
	return []*jobpool.Job{{Err: err}}
}
