// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

const (
	// apiBaseURL is the root of the Stripe REST API.
	apiBaseURL = "https://api.stripe.com"

	// apiVersion pins the Stripe API version so response shapes stay stable
	// regardless of the account's default version. It is sent on every request
	// as the Stripe-Version header.
	apiVersion = "2024-06-20"

	// OptionToken is the CLI flag/option name carrying the Stripe secret key.
	OptionToken = "token"
	// OptionAccount is the CLI flag/option name carrying an optional connected
	// account ID (Stripe Connect), sent as the Stripe-Account header.
	OptionAccount = "account"
	// OptionBaseURL overrides the Stripe API root. It is useful for pointing at
	// a Stripe-compatible proxy or a mock server (including in tests).
	OptionBaseURL = "base-url"
)

type StripeConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset

	token   string
	account string
	baseURL string
	client  *http.Client
}

func NewStripeConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*StripeConnection, error) {
	baseURL := apiBaseURL
	if conf.Options != nil {
		if raw := strings.TrimSpace(conf.Options[OptionBaseURL]); raw != "" {
			baseURL = raw
		}
	}

	conn := &StripeConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
		baseURL:    baseURL,
		client:     &http.Client{Timeout: 60 * time.Second},
	}

	// The credential populated by ParseCLI (from --token or the env var) is the
	// authoritative source. The env var is only a fallback for connections
	// built from an inventory config that never went through ParseCLI.
	token := ""
	for _, cred := range conf.Credentials {
		if cred.Type == vault.CredentialType_password && len(cred.Secret) > 0 {
			token = strings.TrimSpace(string(cred.Secret))
		}
	}
	if token == "" {
		token = strings.TrimSpace(os.Getenv("STRIPE_API_KEY"))
	}
	if token == "" {
		token = strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY"))
	}
	if token == "" {
		return nil, errors.New("a valid Stripe secret key is required (set STRIPE_API_KEY or use --token)")
	}
	conn.token = token

	if conf.Options != nil {
		conn.account = strings.TrimSpace(conf.Options[OptionAccount])
	}

	return conn, nil
}

func (c *StripeConnection) Name() string {
	return "stripe"
}

func (c *StripeConnection) Asset() *inventory.Asset {
	return c.asset
}
