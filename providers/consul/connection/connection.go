// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"os"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	mondoovault "go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

const (
	// requestTimeout bounds a single API call. A Consul agent answers the
	// endpoints this provider reads out of memory or out of the local Raft
	// state, so a slow response means the agent is unhealthy rather than the
	// work being large.
	requestTimeout = 60 * time.Second

	// OptionAddress names the Consul HTTP API endpoint.
	OptionAddress = "address"
	// OptionCACert names the certificate authority to trust, either as the PEM
	// itself or as a path to it. A Consul HTTPS listener is commonly published
	// under the cluster's own authority, and trusting it keeps the certificate
	// checked.
	OptionCACert = "ca-cert"
	// OptionTLSSkipVerify disables certificate verification. It exists for lab
	// agents using a self-signed certificate and is never appropriate against a
	// production cluster.
	OptionTLSSkipVerify = "tls-skip-verify"
)

// ConsulConnection holds an authenticated client for one Consul agent.
type ConsulConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset

	client  *consulapi.Client
	address string
	host    string
}

func NewConsulConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*ConsulConnection, error) {
	conn := &ConsulConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	address := option(conf, OptionAddress)
	if address == "" {
		address = os.Getenv(consulapi.HTTPAddrEnvName)
	}
	if address == "" {
		// A Consul agent listens on the loopback interface by default, and the
		// CLI assumes the same endpoint, so an operator running mql on an agent
		// host expects it to work with no arguments.
		address = DefaultAddress
	}

	normalized, host, err := NormalizeAddress(address)
	if err != nil {
		return nil, err
	}
	conn.address = normalized
	conn.host = host

	cfg := consulapi.DefaultConfig()
	cfg.Address = normalized

	if err := applyTLSConfig(cfg, conf); err != nil {
		return nil, err
	}

	// Build the HTTP client here rather than letting NewClient do it, so the
	// request timeout is attached. Without it a hung agent blocks a scan
	// indefinitely, because the Consul client sets no timeout of its own.
	httpClient, err := consulapi.NewHttpClient(cfg.Transport, cfg.TLSConfig)
	if err != nil {
		return nil, err
	}
	httpClient.Timeout = requestTimeout
	cfg.HttpClient = httpClient

	if token := tokenFromConf(conf); token != "" {
		cfg.Token = token
	}
	// A token may also arrive through CONSUL_HTTP_TOKEN, which DefaultConfig
	// already read into cfg.Token. An agent with ACLs switched off needs no
	// token at all, so an empty token is not an error here: the resources
	// report what the anonymous token may read.

	client, err := consulapi.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	conn.client = client

	return conn, nil
}

// tokenFromConf pulls the ACL token out of the configured credentials. It
// arrives as a password credential, which is the shape the SDK uses for a bare
// secret with no user name.
func tokenFromConf(conf *inventory.Config) string {
	if conf == nil {
		return ""
	}
	for _, cred := range conf.Credentials {
		if cred == nil || len(cred.Secret) == 0 {
			continue
		}
		if cred.Type != mondoovault.CredentialType_password {
			continue
		}
		return string(cred.Secret)
	}
	return ""
}

func (c *ConsulConnection) Name() string { return "consul" }

func (c *ConsulConnection) Asset() *inventory.Asset { return c.asset }

// Client returns the configured Consul API client.
func (c *ConsulConnection) Client() *consulapi.Client { return c.client }

// Address is the normalized Consul HTTP API endpoint.
func (c *ConsulConnection) Address() string { return c.address }

// Host is the host[:port] of the Consul endpoint, used to build platform IDs.
func (c *ConsulConnection) Host() string { return c.host }

func option(conf *inventory.Config, key string) string {
	if conf == nil || conf.Options == nil {
		return ""
	}
	return conf.Options[key]
}
