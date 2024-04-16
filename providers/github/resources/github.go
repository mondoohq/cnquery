// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"github.com/google/go-github/v59/github"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/memoize"
	"go.mondoo.com/cnquery/v11/providers/github/connection"
)

// We use a global MQL resource for the connection to store the memoizer.
// In this way we can cache requests for things that aren't directly attached to an MQL resource.
// For example, we don't want to fetch the same users multiple times, for that we can use the memoizer.
type mqlGithubInternal struct {
	memoize *memoize.Memoizer
}

func getUser(ctx context.Context, runtime *plugin.Runtime, conn *connection.GithubConnection, user string) (*github.User, error) {
	obj, err := CreateResource(runtime, "github", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	g := obj.(*mqlGithub)
	if g.memoize == nil {
		g.memoize = memoize.NewMemoizer(30*time.Minute, 1*time.Hour)
	}
	res, err, _ := g.memoize.Memoize("user", func() (interface{}, error) {
		log.Debug().Msgf("fetching user %s", user)
		user, _, err := conn.Client().Users.Get(ctx, user)
		return user, err
	})
	if err != nil {
		return nil, err
	}
	return res.(*github.User), nil
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
