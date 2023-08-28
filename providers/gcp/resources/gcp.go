// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"time"
)

// parseTime parses RFC 3389 timestamps "2019-06-12T21:14:13.190Z"
func parseTime(timestamp string) *time.Time {
	parsedCreated, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return nil
	}
	return &parsedCreated
}
