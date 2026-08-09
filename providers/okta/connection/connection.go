// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/okta/okta-sdk-golang/v5/okta"
	goCache "github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
	"go.mondoo.com/mql/v13/providers/okta/resources/sdk"
)

// Okta accepts two kinds of API credential. An SSWS token belongs to the admin
// who created it and carries that admin's full privileges, which is why many
// orgs prohibit them outright. A service application authenticates with a
// private key JWT instead and holds only the scopes it was granted, so it can
// be given read-only access.
const (
	authModeSSWS       = "SSWS"
	authModePrivateKey = "PrivateKey"
)

// accessTokenCacheTTL bounds how long a minted access token is reused for the
// requests issued outside the generated SDK. Each entry is also stored under
// the token's own expiry, so this is only an upper bound.
const accessTokenCacheTTL = 55 * time.Minute

type OktaConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset
	// custom connection fields
	organization string
	client       *okta.APIClient
	token        string
	// authorize stamps the Authorization header onto requests issued outside
	// the generated SDK, so the raw endpoints in resources/sdk authenticate the
	// same way the SDK-served ones do.
	authorize func(req *http.Request) error
}

func NewOktaConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*OktaConnection, error) {
	conn := &OktaConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	// initialize connection
	if conf.Type != "okta" {
		return nil, errors.New("provider type does not match") // TODO: switch to plugin.ErrProviderTypeDoesNotMatch
	}

	if conf.Options == nil || conf.Options["organization"] == "" {
		return nil, errors.New("okta provider requires an organization id. please set option `organization`")
	}

	org := conf.Options["organization"]

	var token string
	var privateKey string
	if len(conf.Credentials) > 0 {
		log.Debug().Int("credentials", len(conf.Credentials)).Msg("credentials")
		for i := range conf.Credentials {
			cred := conf.Credentials[i]
			switch cred.Type {
			case vault.CredentialType_password:
				token = string(cred.Secret)
			case vault.CredentialType_private_key:
				privateKey = string(cred.Secret)
			default:
				log.Warn().Str("credential-type", cred.Type.String()).Msg("unsupported credential type for Okta provider")
			}
		}
	}

	orgURL := "https://" + org
	options := []okta.ConfigSetter{okta.WithOrgUrl(orgURL)}
	authMode := authModeSSWS

	switch {
	case privateKey != "":
		clientID := conf.Options["client-id"]
		privateKeyID := conf.Options["private-key-id"]
		scopes := splitScopes(conf.Options["scopes"])
		if clientID == "" {
			return nil, errors.New("okta service app authentication requires a client id, pass --client-id or set OKTA_API_CLIENT_ID")
		}
		if privateKeyID == "" {
			return nil, errors.New("okta service app authentication requires the id of the registered public key, pass --private-key-id or set OKTA_API_PRIVATE_KEY_ID")
		}
		if len(scopes) == 0 {
			return nil, errors.New("okta service app authentication requires the scopes to request, pass --scopes or set OKTA_API_SCOPES")
		}

		authMode = authModePrivateKey
		options = append(options,
			okta.WithAuthorizationMode(authModePrivateKey),
			okta.WithClientId(clientID),
			okta.WithScopes(scopes),
			okta.WithPrivateKey(privateKey),
			okta.WithPrivateKeyId(privateKeyID),
		)

	case token != "":
		options = append(options, okta.WithToken(token))

	default:
		return nil, errors.New("okta requires either an API token (--token or OKTA_API_TOKEN) or service app credentials (--client-id, --private-key, --private-key-id and --scopes)")
	}

	config, err := okta.NewConfiguration(options...)
	if err != nil {
		return nil, err
	}
	client := okta.NewAPIClient(config)

	// Build the authorizer from the finished configuration so requests issued
	// outside the generated SDK carry the same user agent and rate-limit
	// settings the SDK-served ones do.
	if authMode == authModePrivateKey {
		conn.authorize = privateKeyAuthorizer(config)
	} else {
		conn.authorize = sswsAuthorizer(token)
	}

	conn.organization = org
	conn.client = client
	conn.token = token

	return conn, nil
}

// splitScopes accepts the scopes as a single option value, separated by commas
// or whitespace, since Okta writes them space-separated but a CLI flag reads
// more naturally comma-separated.
func splitScopes(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
	scopes := make([]string, 0, len(fields))
	for _, f := range fields {
		if f != "" {
			scopes = append(scopes, f)
		}
	}
	return scopes
}

func sswsAuthorizer(token string) func(*http.Request) error {
	return func(req *http.Request) error {
		req.Header.Set("Authorization", "SSWS "+token)
		return nil
	}
}

// privateKeyAuthorizer reuses the SDK's private key JWT exchange for requests
// the generated client does not serve. The SDK keeps its minted access token in
// a cache private to its APIClient, so this holds its own; the cost is one
// extra token exchange for the raw path, after which both are cached.
func privateKeyAuthorizer(config *okta.Configuration) func(*http.Request) error {
	tokenCache := goCache.New(accessTokenCacheTTL, accessTokenCacheTTL)
	client := config.Okta.Client
	userAgent := okta.NewUserAgent(config).String()

	return func(req *http.Request) error {
		auth := okta.NewPrivateKeyAuth(okta.PrivateKeyAuthConfig{
			TokenCache:       tokenCache,
			HttpClient:       config.HTTPClient,
			PrivateKeySigner: config.PrivateKeySigner,
			PrivateKey:       client.PrivateKey,
			PrivateKeyId:     client.PrivateKeyId,
			ClientId:         client.ClientId,
			OrgURL:           client.OrgUrl,
			UserAgent:        userAgent,
			Scopes:           client.Scopes,
			MaxRetries:       client.RateLimit.MaxRetries,
			MaxBackoff:       client.RateLimit.MaxBackoff,
			Req:              req,
		})
		// The SDK signs the DPoP proof against the request URL without its
		// query string, so strip it here the same way the generated client does.
		urlWithoutQuery := *req.URL
		urlWithoutQuery.RawQuery = ""
		return auth.Authorize(req.Method, urlWithoutQuery.String())
	}
}

func (c *OktaConnection) Name() string {
	return "okta"
}

func (c *OktaConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *OktaConnection) OrganizationID() string {
	return c.organization
}

func (c *OktaConnection) Client() *okta.APIClient {
	return c.client
}

func (c *OktaConnection) Token() string {
	return c.token
}

// ApiExtension returns a client for the Okta endpoints the generated SDK does
// not serve correctly. It carries this connection's authorization, so the raw
// path works under service app credentials as well as an API token.
func (c *OktaConnection) ApiExtension() *sdk.ApiExtension {
	return &sdk.ApiExtension{
		Host:      c.organization,
		Authorize: c.authorize,
	}
}

func (c *OktaConnection) Identifier() (string, error) {
	settings, _, err := c.client.OrgSettingAPI.GetOrgSettings(context.Background()).Execute()
	if err != nil {
		return "", errors.Join(errors.New("failed to get Okta org ID"), err)
	}
	if settings == nil || settings.Id == nil {
		return "", errors.New("failed to get Okta org ID: empty response")
	}

	return *settings.Id, nil
}
