// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/cockroachdb/errors"
	"github.com/google/go-github/v62/github"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

	return github.NewClient(newGithubRetryableClient(&http.Client{Transport: itr})), nil
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
	return github.NewClient(newGithubRetryableClient(tc)), nil
}

func newGithubRetryableClient(httpClient *http.Client) *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 5
	retryClient.Logger = &zeroLogAdapter{}

	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	retryClient.HTTPClient = httpClient

	retryClient.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		// Default Retry Policy would not retry on 403 (adding 429 for good measure)
		if resp.StatusCode == 403 || resp.StatusCode == 429 {
			// Primary and Secondary rate limit
			if resp.Header.Get("x-ratelimit-remaining") == "0" {
				return true, nil // Should be retried after the rate limit reset (duration handled by Backoff)
			}
		}
		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}
	retryClient.Backoff = func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
		if resp.StatusCode == 403 || resp.StatusCode == 429 {
			// Secondary limit
			if resp.Header.Get("retry-after") != "" {
				sec, err := strconv.ParseInt(resp.Header.Get("retry-after"), 10, 64) // retry-after	- The number of seconds to wait before making a follow-up request
				if err != nil {                                                      // Must be impossible to hit errors here, but just in case
					return time.Second * 8
				}
				return time.Second * time.Duration(sec)
			}
			// Primary and Secondary rate limit
			if resp.Header.Get("x-ratelimit-remaining") == "0" {
				unix, err := strconv.ParseInt(resp.Header.Get("x-ratelimit-reset"), 10, 64) // x-ratelimit-reset	- The time at which the current rate limit window resets, in UTC epoch seconds
				if err != nil {                                                             // Must be impossible to hit errors here, but just in case
					return time.Second * 8
				}
				tm := time.Unix(unix, 0)
				return tm.Sub(time.Now().UTC()) // time.Until might not use UTC, depending on the server configuration
			}
		}

		return retryablehttp.DefaultBackoff(min, max, attemptNum, resp)
	}

	return retryClient.StandardClient()
}

// zeroLogAdapter is the adapter for retryablehttp is outputting debug messages
type zeroLogAdapter struct{}

func (l *zeroLogAdapter) Msg(msg string, keysAndValues ...interface{}) {
	var e *zerolog.Event
	// retry messages should only go to debug
	e = log.Debug()
	for i := 0; i < len(keysAndValues); i += 2 {
		e = e.Interface(keysAndValues[i].(string), keysAndValues[i+1])
	}
	e.Msg(msg)
}

func (l *zeroLogAdapter) Error(msg string, keysAndValues ...interface{}) {
	l.Msg(msg, keysAndValues...)
}

func (l *zeroLogAdapter) Info(msg string, keysAndValues ...interface{}) {
	l.Msg(msg, keysAndValues...)
}

func (l *zeroLogAdapter) Debug(msg string, keysAndValues ...interface{}) {
	l.Msg(msg, keysAndValues...)
}

func (l *zeroLogAdapter) Warn(msg string, keysAndValues ...interface{}) {
	l.Msg(msg, keysAndValues...)
}
