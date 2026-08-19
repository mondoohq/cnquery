// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"os"
	"strings"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	mondoovault "go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

const (
	// requestTimeout bounds a single API call. Vault answers from memory or its
	// storage backend, so a slow response means the server is unhealthy rather
	// than the work being large.
	requestTimeout = 60 * time.Second

	// OptionAddress names the Vault API endpoint.
	OptionAddress = "address"
	// OptionNamespace selects an Enterprise namespace to scan. An empty value
	// scans the root namespace.
	OptionNamespace = "namespace"
	// OptionRoleID carries the AppRole role ID. The matching secret ID arrives
	// as a credential rather than an option, since it is the secret half.
	OptionRoleID = "role-id"
	// OptionCACert names the certificate authority to trust, either as the PEM
	// itself or as a path to it. Vault is commonly published under a private
	// authority, and trusting it keeps the certificate checked.
	OptionCACert = "ca-cert"
	// OptionTLSSkipVerify disables certificate verification. It exists for lab
	// servers using a self-signed certificate and is never appropriate against
	// a production Vault.
	OptionTLSSkipVerify = "tls-skip-verify"
)

// VaultConnection holds an authenticated client for one Vault server.
type VaultConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset

	client    *vaultapi.Client
	address   string
	host      string
	namespace string
}

func NewVaultConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*VaultConnection, error) {
	conn := &VaultConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	address := option(conf, OptionAddress)
	if address == "" {
		address = os.Getenv("VAULT_ADDR")
	}
	if address == "" {
		return nil, errors.New("a Vault address is required (set VAULT_ADDR or use --address)")
	}

	normalized, host, err := NormalizeAddress(address)
	if err != nil {
		return nil, err
	}
	conn.address = normalized
	conn.host = host

	cfg := vaultapi.DefaultConfig()
	cfg.Address = normalized
	cfg.Timeout = requestTimeout
	// DefaultConfig reads VAULT_* environment variables into cfg.Error. Surface
	// that now, because a malformed VAULT_CLIENT_TIMEOUT would otherwise only
	// show up as an unrelated failure on the first request.
	if cfg.Error != nil {
		return nil, cfg.Error
	}

	if err := applyTLSConfig(cfg, conf); err != nil {
		return nil, err
	}

	client, err := vaultapi.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	conn.namespace = option(conf, OptionNamespace)
	if conn.namespace == "" {
		conn.namespace = os.Getenv("VAULT_NAMESPACE")
	}
	if conn.namespace != "" {
		client.SetNamespace(conn.namespace)
	}

	if err := authenticate(client, conf, option(conf, OptionRoleID)); err != nil {
		return nil, err
	}
	conn.client = client

	return conn, nil
}

// authenticate resolves a usable token onto the client. A token is used
// directly; a role ID means AppRole, which trades the role and secret ID for a
// short-lived token.
func authenticate(client *vaultapi.Client, conf *inventory.Config, roleID string) error {
	token, secretID := credentialsFromConf(conf)

	if roleID != "" {
		if secretID == "" {
			secretID = os.Getenv("VAULT_SECRET_ID")
		}
		if secretID == "" {
			return errors.New("AppRole login needs a secret ID (set VAULT_SECRET_ID or pass --secret-id)")
		}
		return appRoleLogin(client, roleID, secretID)
	}

	if token == "" {
		token = os.Getenv("VAULT_TOKEN")
	}
	if token == "" {
		return errors.New("a Vault token is required (set VAULT_TOKEN, or use --role-id with --secret-id for AppRole)")
	}
	client.SetToken(token)
	return nil
}

// appRoleLogin exchanges a role and secret ID for a token.
func appRoleLogin(client *vaultapi.Client, roleID, secretID string) error {
	resp, err := client.Logical().Write("auth/approle/login", map[string]any{
		"role_id":   roleID,
		"secret_id": secretID,
	})
	if err != nil {
		return err
	}
	if resp == nil || resp.Auth == nil || resp.Auth.ClientToken == "" {
		return errors.New("AppRole login returned no token")
	}
	client.SetToken(resp.Auth.ClientToken)
	return nil
}

// credentialsFromConf pulls the token and the AppRole secret ID out of the
// configured credentials. Both arrive as password credentials, so the user name
// is what tells them apart: a secret ID is tagged, a bare password is the token.
func credentialsFromConf(conf *inventory.Config) (token, secretID string) {
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
		if strings.EqualFold(cred.User, "secret-id") {
			secretID = string(cred.Secret)
			continue
		}
		token = string(cred.Secret)
	}
	return token, secretID
}

func (c *VaultConnection) Name() string { return "vault" }

func (c *VaultConnection) Asset() *inventory.Asset { return c.asset }

// Client returns the authenticated Vault API client.
func (c *VaultConnection) Client() *vaultapi.Client { return c.client }

// Address is the normalized Vault API endpoint.
func (c *VaultConnection) Address() string { return c.address }

// Host is the host[:port] of the Vault endpoint, used to build platform IDs.
func (c *VaultConnection) Host() string { return c.host }

// Namespace is the Enterprise namespace being scanned, empty for root.
func (c *VaultConnection) Namespace() string { return c.namespace }

func option(conf *inventory.Config, key string) string {
	if conf == nil || conf.Options == nil {
		return ""
	}
	return conf.Options[key]
}
