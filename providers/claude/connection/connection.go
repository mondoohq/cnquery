// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"errors"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

const (
	TokenOption             = "token"
	AdminTokenOption        = "admin-token"
	IdentityTokenFileOption = "identity-token-file"
	FederationRuleIDOption  = "federation-rule-id"
	OrganizationIDOption    = "organization-id"
	ServiceAccountIDOption  = "service-account-id"
	WorkspaceIDOption       = "workspace-id"
)

type ClaudeConnection struct {
	plugin.Connection
	Conf       *inventory.Config
	asset      *inventory.Asset
	client     *anthropic.Client
	adminToken string
	host       string
	orgID      string
	orgName    string
}

func NewClaudeConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*ClaudeConnection, error) {
	token := conf.Options[TokenOption]
	if token == "" {
		token = os.Getenv("ANTHROPIC_API_KEY")
	}

	adminToken := conf.Options[AdminTokenOption]
	if adminToken == "" {
		adminToken = os.Getenv("ANTHROPIC_ADMIN_API_KEY")
	}

	host := "https://api.anthropic.com"

	var opts []option.RequestOption
	if token != "" {
		opts = append(opts, option.WithAPIKey(token))
	}

	if identityTokenFile := conf.Options[IdentityTokenFileOption]; identityTokenFile != "" {
		fedOpts := option.FederationOptions{
			FederationRuleID: conf.Options[FederationRuleIDOption],
			OrganizationID:   conf.Options[OrganizationIDOption],
			ServiceAccountID: conf.Options[ServiceAccountIDOption],
			WorkspaceID:      conf.Options[WorkspaceIDOption],
		}
		opts = append(opts, option.WithFederationTokenProvider(
			option.IdentityTokenFile(identityTokenFile), fedOpts,
		))
	}

	hasSDKCreds := token != "" ||
		os.Getenv("ANTHROPIC_AUTH_TOKEN") != "" ||
		os.Getenv("ANTHROPIC_PROFILE") != "" ||
		conf.Options[IdentityTokenFileOption] != "" ||
		os.Getenv("ANTHROPIC_FEDERATION_RULE_ID") != ""
	if !hasSDKCreds && adminToken == "" {
		return nil, errors.New("no credentials provided: set --token, --admin-token, ANTHROPIC_API_KEY, or configure WIF")
	}

	client := anthropic.NewClient(opts...)

	conn := &ClaudeConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
		client:     &client,
		adminToken: adminToken,
		host:       host,
	}

	if adminToken != "" {
		adminClient := NewAdminClient(adminToken, host)
		org, err := adminClient.GetOrganization(context.Background())
		if err == nil {
			conn.orgID = org.ID
			conn.orgName = org.Name
		}
	}

	return conn, nil
}

func (c *ClaudeConnection) Name() string {
	return "claude"
}

func (c *ClaudeConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *ClaudeConnection) Client() *anthropic.Client {
	return c.client
}

func (c *ClaudeConnection) AdminToken() string {
	return c.adminToken
}

func (c *ClaudeConnection) Host() string {
	return c.host
}

func (c *ClaudeConnection) OrgID() string {
	return c.orgID
}

func (c *ClaudeConnection) OrgName() string {
	return c.orgName
}
