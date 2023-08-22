// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"time"

	"github.com/google/go-github/v49/github"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/github/connection"
)

func githubProvider(t plugin.Connection) (*connection.GithubConnection, error) {
	gt, ok := t.(*connection.GithubConnection)
	if !ok {
		return nil, errors.New("github resource is not supported on this provider")
	}
	return gt, nil
}

func githubTimestamp(ts *github.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	return &ts.Time
}

const (
	paginationPerPage = 100
)
