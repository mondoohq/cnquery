// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"github.com/google/go-github/v67/github"
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

var (
	cacheExpirationTime = 24 * time.Hour
	cacheCleanupTime    = 48 * time.Hour
)

func getUser(ctx context.Context, runtime *plugin.Runtime, conn *connection.GithubConnection, user string) (*github.User, error) {
	obj, err := CreateResource(runtime, "github", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	g := obj.(*mqlGithub)
	if g.memoize == nil {
		g.memoize = memoize.NewMemoizer(cacheExpirationTime, cacheCleanupTime)
	}

	res, err, _ := g.memoize.Memoize("user-"+user, func() (interface{}, error) {
		log.Debug().Msgf("fetching user %s", user)
		user, _, err := conn.Client().Users.Get(ctx, user)
		return user, err
	})
	if err != nil {
		return nil, err
	}
	return res.(*github.User), nil
}

func getOrg(ctx context.Context, runtime *plugin.Runtime, conn *connection.GithubConnection, name string) (*github.Organization, error) {
	obj, err := CreateResource(runtime, "github", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	g := obj.(*mqlGithub)
	if g.memoize == nil {
		g.memoize = memoize.NewMemoizer(cacheExpirationTime, cacheCleanupTime)
	}
	res, err, _ := g.memoize.Memoize("org-"+name, func() (interface{}, error) {
		log.Debug().Msgf("fetching organization %s", name)
		org, _, err := conn.Client().Organizations.Get(ctx, name)
		return org, err
	})
	if err != nil {
		return nil, err
	}
	return res.(*github.Organization), nil
}

func githubTimestamp(ts *github.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	return &ts.Time
}

const (
	paginationPerPage = 100
	workers           = 10
)
