// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"os"
	"strings"
	"time"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	mondoovault "go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

const (
	// requestTimeout bounds a single API call. Rancher answers a management API
	// request out of its own database, so a slow response means the server is
	// unhealthy rather than the work being large.
	requestTimeout = 60 * time.Second

	// OptionURL names the Rancher Manager endpoint.
	OptionURL = "url"
	// OptionAccessKey carries the token's public half. The matching secret half
	// arrives as a credential rather than an option.
	OptionAccessKey = "access-key"
	// OptionCACert names the certificate authority to trust, either as the PEM
	// itself or as a path to it. Rancher is commonly published under a private
	// authority, and trusting it keeps the certificate checked.
	OptionCACert = "ca-cert"
	// OptionTLSSkipVerify disables certificate verification. It exists for lab
	// servers using a self-signed certificate and is never appropriate against
	// a production Rancher.
	OptionTLSSkipVerify = "tls-skip-verify"
)

// RancherConnection holds an authenticated client for one Rancher Manager.
type RancherConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset

	client *Client
	url    string
	host   string
}

func NewRancherConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*RancherConnection, error) {
	conn := &RancherConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	rawURL := option(conf, OptionURL)
	if rawURL == "" {
		rawURL = os.Getenv("RANCHER_URL")
	}
	if rawURL == "" {
		return nil, errors.New("a Rancher URL is required (set RANCHER_URL or use --url)")
	}

	normalized, host, err := NormalizeURL(rawURL)
	if err != nil {
		return nil, err
	}
	conn.url = normalized
	conn.host = host

	token, err := resolveToken(conf)
	if err != nil {
		return nil, err
	}

	httpClient, err := newHTTPClient(conf, requestTimeout)
	if err != nil {
		return nil, err
	}

	conn.client = NewClient(normalized, token, httpClient)
	return conn, nil
}

// resolveToken builds the bearer token Rancher expects. A Rancher API key is a
// pair: a public access key that names the token object, and a secret half. The
// wire form joins them with a colon, which is also how the console hands the
// key out, so a whole key pasted into RANCHER_TOKEN is used as-is.
func resolveToken(conf *inventory.Config) (string, error) {
	token, secretKey := credentialsFromConf(conf)

	accessKey := option(conf, OptionAccessKey)
	if accessKey == "" {
		accessKey = os.Getenv("RANCHER_ACCESS_KEY")
	}
	if secretKey == "" {
		secretKey = os.Getenv("RANCHER_SECRET_KEY")
	}

	if accessKey != "" {
		if secretKey == "" {
			return "", errors.New("an access key needs its secret key (set RANCHER_SECRET_KEY or pass --secret-key)")
		}
		return accessKey + ":" + secretKey, nil
	}

	if token == "" {
		token = os.Getenv("RANCHER_TOKEN")
	}
	if token == "" {
		return "", errors.New("a Rancher API token is required (set RANCHER_TOKEN, or use --access-key with --secret-key)")
	}
	return token, nil
}

// credentialsFromConf pulls the bearer token and the secret key out of the
// configured credentials. Both arrive as password credentials, so the user name
// is what tells them apart: a secret key is tagged, a bare password is the
// whole token.
func credentialsFromConf(conf *inventory.Config) (token, secretKey string) {
	if conf == nil {
		return "", ""
	}
	for _, cred := range conf.Credentials {
		if cred == nil || len(cred.Secret) == 0 {
			continue
		}
		if cred.Type != mondoovault.CredentialType_password {
			continue
		}
		if strings.EqualFold(cred.User, "secret-key") {
			secretKey = string(cred.Secret)
			continue
		}
		token = string(cred.Secret)
	}
	return token, secretKey
}

func (c *RancherConnection) Name() string { return "rancher" }

func (c *RancherConnection) Asset() *inventory.Asset { return c.asset }

// Client returns the authenticated Rancher API client.
func (c *RancherConnection) Client() *Client { return c.client }

// URL is the normalized Rancher Manager endpoint.
func (c *RancherConnection) URL() string { return c.url }

// Host is the host[:port] of the Rancher endpoint, used to build platform IDs.
func (c *RancherConnection) Host() string { return c.host }

func option(conf *inventory.Config, key string) string {
	if conf == nil || conf.Options == nil {
		return ""
	}
	return conf.Options[key]
}
