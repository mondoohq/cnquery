// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/cockroachdb/errors"
	"github.com/google/go-github/v72/github"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/logger/zerologadapter"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
	"golang.org/x/oauth2"
)

const (
	OPTION_TOKEN               = "token"
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
	ctx    context.Context

	// Used to avoid verifying a client with the same options more than once
	Hash uint64
}

type githubConnectionOptions struct {
	// Maps to OPTION_ENTERPRISE_URL
	EnterpriseURL string

	// Maps to OPTION_APP_ID
	AppID string
	// Maps to OPTION_APP_INSTALLATION_ID
	AppInstallationID string
	// Maps to OPTION_APP_PRIVATE_KEY
	AppPrivateKeyFile string
	AppPrivateKey     []byte
	// Maps to OPTION_TOKEN or the environment variable GITHUB_TOKEN
	Token string
}

func connectionOptionsFromConfigOptions(conf *inventory.Config) (opts githubConnectionOptions) {
	if conf == nil {
		return
	}

	opts.AppID = conf.Options[OPTION_APP_ID]
	opts.AppInstallationID = conf.Options[OPTION_APP_INSTALLATION_ID]
	opts.AppPrivateKeyFile = conf.Options[OPTION_APP_PRIVATE_KEY]
	opts.EnterpriseURL = conf.Options[OPTION_ENTERPRISE_URL]
	opts.Token = conf.Options[OPTION_TOKEN]

	for _, cred := range conf.Credentials {
		switch cred.Type {

		case vault.CredentialType_private_key:
			if opts.AppPrivateKeyFile == "" {
				opts.AppPrivateKey = cred.Secret
			}

		case vault.CredentialType_password:
			if opts.Token == "" {
				opts.Token = string(cred.Secret)
			}
		}
	}

	return
}

func NewGithubConnection(id uint32, asset *inventory.Asset) (*GithubConnection, error) {
	if asset.Connections == nil {
		return nil, errors.New("no connection details for the asset")
	}
	connectionOpts := connectionOptionsFromConfigOptions(asset.Connections[0])

	var client *github.Client
	var err error
	if connectionOpts.AppID != "" {
		client, err = newGithubAppClient(connectionOpts)
	} else {
		client, err = newGithubTokenClient(connectionOpts)
	}
	if err != nil {
		return nil, err
	}

	if connectionOpts.EnterpriseURL != "" {
		parsedUrl, err := url.Parse(connectionOpts.EnterpriseURL)
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

	// set the context so github client can handle backoff
	// (default behaviour is to send fake 403 response bypassing the retry logic)
	ctx := context.WithValue(context.Background(), github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)

	// store the hash of the config options used to generate this client
	hash, err := hashstructure.Hash(connectionOpts, hashstructure.FormatV2, nil)

	return &GithubConnection{
		Connection: plugin.NewConnection(id, asset),
		asset:      asset,
		client:     client,
		ctx:        ctx,
		Hash:       hash,
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

func (c *GithubConnection) Context() context.Context {
	return c.ctx
}

func (c *GithubConnection) Verify() error {
	// perform a quick call to verify the token's validity.
	_, resp, err := c.client.Meta.Zen(c.ctx)
	if err != nil {
		if resp != nil && resp.StatusCode == 401 {
			return errors.New(
				"invalid GitHub token provided. check the value passed with the --token flag or the GITHUB_TOKEN environment variable",
			)
		}
		return err
	}
	return nil
}

func newGithubAppClient(opts githubConnectionOptions) (*github.Client, error) {
	if opts.AppID == "" {
		return nil, errors.New("app-id is required for GitHub App authentication")
	}
	appId, err := strconv.ParseInt(opts.AppID, 10, 32)
	if err != nil {
		return nil, err
	}

	if opts.AppInstallationID == "" {
		return nil, errors.New("app-installation-id is required for GitHub App authentication")
	}
	appInstallationId, err := strconv.ParseInt(opts.AppInstallationID, 10, 32)
	if err != nil {
		return nil, err
	}

	var itr *ghinstallation.Transport
	if opts.AppPrivateKeyFile != "" {
		itr, err = ghinstallation.NewKeyFromFile(http.DefaultTransport, appId, appInstallationId, opts.AppPrivateKeyFile)
	} else {
		itr, err = ghinstallation.New(http.DefaultTransport, appId, appInstallationId, opts.AppPrivateKey)
	}
	if err != nil {
		return nil, err
	}

	if itr == nil {
		return nil, errors.New("app-private-key is required for GitHub App authentication")
	}

	return github.NewClient(newGithubRetryableClient(&http.Client{Transport: itr})), nil
}

func newGithubTokenClient(opts githubConnectionOptions) (*github.Client, error) {
	// if we still have no token, error out
	if opts.Token == "" {
		return nil, errors.New("a valid GitHub token is required, pass --token '<yourtoken>' or set GITHUB_TOKEN environment variable")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: opts.Token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(newGithubRetryableClient(tc)), nil
}

func newGithubRetryableClient(httpClient *http.Client) *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 5
	retryClient.Logger = zerologadapter.New(log.Logger)

	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	retryClient.HTTPClient = httpClient

	retryClient.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		// Default Retry Policy would not retry on 403 (adding 429 for good measure)
		if resp != nil && (resp.StatusCode == 403 || resp.StatusCode == 429) {
			if log.Logger.GetLevel() <= zerolog.DebugLevel { // includes Debug and Trace
				e := log.Debug()
				for k, v := range resp.Header {
					e = e.Strs(k, v)
				}
				e.Msg("checking retry")
			}
			// Primary and Secondary rate limit
			if resp.Header.Get("x-ratelimit-remaining") == "0" {
				return true, nil // Should be retried after the rate limit reset (duration handled by Backoff)
			}
		}
		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}
	retryClient.Backoff = func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
		if resp != nil && (resp.StatusCode == 403 || resp.StatusCode == 429) {
			if log.Logger.GetLevel() <= zerolog.DebugLevel { // includes Debug and Trace
				e := log.Debug()
				for k, v := range resp.Header {
					e = e.Strs(k, v)
				}
				e.Msgf("retrying request, attempt %d", attemptNum)
			}

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

		log.Debug().Msgf("falling back to default backoff for attempt %d", attemptNum)
		return retryablehttp.DefaultBackoff(min, max, attemptNum, resp)
	}

	return retryClient.StandardClient()
}
