// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"time"

	"github.com/google/go-github/v57/github"
)

func githubTimestamp(ts *github.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	return &ts.Time
}

const (
	paginationPerPage = 100
)
