// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"time"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
)

// parseTime parses RFC 3389 timestamps "2019-06-12T21:14:13.190Z"
func parseTime(timestamp string) *time.Time {
	parsedCreated, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return nil
	}
	return &parsedCreated
}

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

func jobErr(err error) []*jobpool.Job {
	return []*jobpool.Job{{Err: err}}
}
