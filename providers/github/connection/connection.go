// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/cockroachdb/errors"
	"github.com/google/go-github/v61/github"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
	"golang.org/x/oauth2"
)

const (
	OPTION_REPOS               = "repos"
	OPTION_REPOS_EXCLUDE       = "repos-exclude"
	OPTION_APP_ID              = "app-id"
	OPTION_APP_INSTALLATION_ID = "app-installation-id"
	OPTION_APP_PRIVATE_KEY     = "app-private-key"
	OPTION_ENTERPRISE_URL      = "enterprise-url"
)

type GithubConnection struct {
	plugin.Connection
	asset  *inventory.Asset
	client *github.Client
}

func NewGithubConnection(id uint32, asset *inventory.Asset) (*GithubConnection, error) {
	conf := asset.Connections[0]

	var client *github.Client
	var err error
	appIdStr := conf.Options[OPTION_APP_ID]
	if appIdStr != "" {
		client, err = newGithubAppClient(conf)
	} else {
		client, err = newGithubTokenClient(conf)
	}
	if err != nil {
		return nil, err
	}

	if enterpriseUrl := conf.Options[OPTION_ENTERPRISE_URL]; enterpriseUrl != "" {
		parsedUrl, err := url.Parse(enterpriseUrl)
		if err != nil {
			return nil, err
		}

		baseUrl := parsedUrl.JoinPath("api/v3/")
		uploadUrl := parsedUrl.JoinPath("api/uploads/")
		client, err = client.WithEnterpriseURLs(baseUrl.String(), uploadUrl.String())
		if err != nil {
			return nil, err
		}
	}

	// perform a quick call to verify the token's validity.
	_, resp, err := client.Meta.Zen(context.Background())
	if err != nil {
		if resp != nil && resp.StatusCode == 401 {
			return nil, errors.New("invalid GitHub token provided. check the value passed with the --token flag or the GITHUB_TOKEN environment variable")
		}
		return nil, err
	}
	return &GithubConnection{
		Connection: plugin.NewConnection(id, asset),
		asset:      asset,
		client:     client,
	}, nil
}

func (c *GithubConnection) Name() string {
	return "github"
}

func (c *GithubConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *GithubConnection) Client() *github.Client {
	return c.client
}

func newGithubAppClient(conf *inventory.Config) (*github.Client, error) {
	appIdStr := conf.Options[OPTION_APP_ID]
	if appIdStr == "" {
		return nil, errors.New("app-id is required for GitHub App authentication")
	}
	appId, err := strconv.ParseInt(appIdStr, 10, 32)
	if err != nil {
		return nil, err
	}

	appInstallationIdStr := conf.Options[OPTION_APP_INSTALLATION_ID]
	if appInstallationIdStr == "" {
		return nil, errors.New("app-installation-id is required for GitHub App authentication")
	}
	appInstallationId, err := strconv.ParseInt(appInstallationIdStr, 10, 32)
	if err != nil {
		return nil, err
	}

	var itr *ghinstallation.Transport
	if pk := conf.Options[OPTION_APP_PRIVATE_KEY]; pk != "" {
		itr, err = ghinstallation.NewKeyFromFile(http.DefaultTransport, appId, appInstallationId, pk)
	} else {
		for _, cred := range conf.Credentials {
			switch cred.Type {
			case vault.CredentialType_private_key:
				itr, err = ghinstallation.New(http.DefaultTransport, appId, appInstallationId, cred.Secret)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	if err != nil {
		return nil, err
	}

	if itr == nil {
		return nil, errors.New("app-private-key is required for GitHub App authentication")
	}

	return github.NewClient(&http.Client{Transport: itr}), nil
}

func newGithubTokenClient(conf *inventory.Config) (*github.Client, error) {
	token := ""
	for i := range conf.Credentials {
		cred := conf.Credentials[i]
		switch cred.Type {
		case vault.CredentialType_password:
			token = string(cred.Secret)
		}
	}

	if token == "" {
		token = conf.Options["token"]
		if token == "" {
			token = os.Getenv("GITHUB_TOKEN")
		}
	}

	// if we still have no token, error out
	if token == "" {
		return nil, errors.New("a valid GitHub token is required, pass --token '<yourtoken>' or set GITHUB_TOKEN environment variable")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc), nil
}
