// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
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

func jobErr(err error) []*jobpool.Job {
	return []*jobpool.Job{{Err: err}}
}
