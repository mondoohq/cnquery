// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-github/v67/github"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
)

func TestGithubNoConnection(t *testing.T) {
	os.Setenv("GITHUB_TOKEN", "")
	_, err := NewGithubConnection(0, &inventory.Asset{})
	require.Error(t, err)
}

func TestGithubValidConnection(t *testing.T) {
	_, err := NewGithubConnection(0, &inventory.Asset{
		Connections: []*inventory.Config{{
			Credentials: []*vault.Credential{{
				Type:   vault.CredentialType_password,
				Secret: []byte("super_secret"),
			},
			},
		},
		},
	})
	require.NoError(t, err)
}

func TestGithubNeedsFix(t *testing.T) {
	t.Skip()
	p, err := NewGithubConnection(0, &inventory.Asset{})
	orgName := "mondoohq"
	client := p.Client()
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
