// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build debugtest
// +build debugtest

package github

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-github/v55/github"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
)

func TestGithub(t *testing.T) {
	os.Setenv("GITHUB_TOKEN", "")
	p, err := NewGithubConnection(0, &inventory.Config{})
	require.NoError(t, err)

	client := p.Client()

	orgName := "mondoohq"
	ctx := context.Background()
	org, _, err := client.Organizations.Get(ctx, orgName)
	require.NoError(t, err)
	require.NotNil(t, org)

	owners, _, err := client.Organizations.ListMembers(context.Background(), orgName, &github.ListMembersOptions{
		Role: "admin",
	})
	require.NoError(t, err)
	require.NotNil(t, owners)

	members, _, err := client.Organizations.ListMembers(context.Background(), orgName, nil)
	require.NoError(t, err)
	require.NotNil(t, members)

	// list public repositories for org "github"
	opt := &github.RepositoryListByOrgOptions{Type: "all"}
	repos, _, err := client.Repositories.ListByOrg(context.Background(), orgName, opt)
	require.NoError(t, err)
	require.NotNil(t, repos)

	apps, _, err := client.Organizations.ListInstallations(context.Background(), orgName, &github.ListOptions{})
	require.NoError(t, err)
	require.NotNil(t, apps)
}
