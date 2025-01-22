// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"fmt"
	"net/url"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

const (
	DiscoveryAll     = "all"
	DiscoveryAuto    = "auto"
	DiscoveryDevices = "devices"
)

var (
	PlatformIdTailscaleTailnet = "//platformid.api.mondoo.app/runtime/tailscale/tailnet/"
	PlatformIdTailscaleDevice  = "//platformid.api.mondoo.app/runtime/tailscale/device/"
)

// Flag Options
const (
	OPTION_TOKEN         = "token"
	OPTION_BASE_URL      = "base-url"
	OPTION_CLIENT_ID     = "client-id"
	OPTION_CLIENT_SECRET = "client-secret"
	OPTION_TAILNET       = "tailnet" // from argument in `ParseCLIReq`
)

// Tailscale environment variables
const (
	TAILSCALE_API_KEY_VAR             = "TAILSCALE_API_KEY"
	TAILSCALE_OAUTH_CLIENT_ID_VAR     = "TAILSCALE_OAUTH_CLIENT_ID"
	TAILSCALE_OAUTH_CLIENT_SECRET_VAR = "TAILSCALE_OAUTH_CLIENT_SECRET"
	TAILSCALE_TAILNET_VAR             = "TAILSCALE_TAILNET"
	TAILSCALE_BASE_URL_VAR            = "TAILSCALE_BASE_URL"
)

type TailscaleConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset

	client *tsclient.Client
}

func NewTailscaleConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*TailscaleConnection, error) {
	conn := &TailscaleConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
		client:     &tsclient.Client{Tailnet: "-"}, // a dash represents the default tailnet
	}

	// Detect authentication method
	switch AuthenticationMethod(conf) {
	case OAuthMethod:
		// OAuth client (id and secret)
		clientID, set := GetClientID(conf)
		if !set {
			return nil, fmt.Errorf("missing client id for OAuth authentication. "+
				"Use the --%s flag or via environment variables %s.",
				OPTION_CLIENT_ID,
				TAILSCALE_OAUTH_CLIENT_ID_VAR,
			)
		}

		clientSecret, set := GetClientSecret(conf)
		if !set {
			return nil, fmt.Errorf("missing client secret for OAuth authentication. "+
				"Use the --%s flag or via environment variables %s.",
				OPTION_CLIENT_SECRET,
				TAILSCALE_OAUTH_CLIENT_SECRET_VAR,
			)
		}
		conn.client.HTTP = tsclient.OAuthConfig{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes: []string{
				// Used in resources/tailscale.go `devices()`
				"devices:core:read",
			},
		}.HTTPClient()
		log.Info().Str("method", "OAuth").Msg("tailscale> authentication configured")

	case TokenAuthMethod:
		// API access token
		token, set := GetToken(conf)
		if set {
			conn.client.APIKey = token
			log.Info().Str("method", "token").Msg("tailscale> authentication configured")
			break
		}
		// this should never happen since AuthenticationMethod() already check the token exists
		// but just in case the code there changes without considering this switch, we check
		fallthrough
	case NoAuthMethod:
		return nil, fmt.Errorf("a valid authentication method is required. "+
			"Use a Tailscale access token using the --token flag or an OAuth client passing --client-id and --client-secret. "+
			"Optionally, pass these credentials via environment variables. (%s %s %s %s)",
			TAILSCALE_OAUTH_CLIENT_ID_VAR,
			TAILSCALE_OAUTH_CLIENT_SECRET_VAR,
			TAILSCALE_TAILNET_VAR,
			TAILSCALE_API_KEY_VAR,
		)
	}

	// Configure the base url if set
	if value, set := GetBaseURL(conf); set {
		baseURL, err := url.Parse(value)
		if err != nil {
			return nil, errors.Wrap(err, "unable to configure base url")
		}
		conn.client.BaseURL = baseURL
		log.Info().Str("url", value).Msg("tailscale> base url configured")
	}

	// Configure a tailnet if set
	if value, set := GetTailnet(conf); set {
		conn.client.Tailnet = value
	}

	return conn, nil
}

func (t *TailscaleConnection) Asset() *inventory.Asset {
	return t.asset
}
func (t *TailscaleConnection) Name() string {
	return "tailscale"
}
func (t *TailscaleConnection) Client() *tsclient.Client {
	return t.client
}

func (t *TailscaleConnection) PlatformInfo() (*inventory.Platform, error) {
	return &inventory.Platform{
		Name:                  "tailscale-org",
		Title:                 "Tailscale",
		Family:                []string{"tailscale"},
		Kind:                  "api",
		Runtime:               "tailscale",
		TechnologyUrlSegments: []string{"network", "tailscale", "org"},
	}, nil
}

func (t *TailscaleConnection) Identifier() string {
	// when a tailnet is not provided, Tailscale uses the default tailnet of the provided creds
	// TODO can we make an API call to get the default tailnet and set it as identifier?
	tailnet := "default"

	conf := t.asset.Connections[0]
	if conf.Options != nil && conf.Options[OPTION_TAILNET] != "" {
		tailnet = conf.Options[OPTION_TAILNET]
	}

	return PlatformIdTailscaleTailnet + tailnet
}

func NewTailscaleDeviceIdentifier(deviceId string) string {
	return PlatformIdTailscaleDevice + deviceId
}

func NewTailscaleDevicePlatform(deviceId string) *inventory.Platform {
	return &inventory.Platform{
		Name:                  "tailscale-device",
		Title:                 "Tailscale Device",
		Family:                []string{"tailscale"},
		TechnologyUrlSegments: []string{"saas", "tailscale", "device", deviceId},
		Kind:                  "api",
		Runtime:               "tailscale",
	}
}
