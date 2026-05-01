// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"errors"
	"os"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

type DatadogConnection struct {
	plugin.Connection
	Conf        *inventory.Config
	asset       *inventory.Asset
	apiClient   *datadog.APIClient
	authCtx     context.Context
	orgPublicId string
}

func NewDatadogConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*DatadogConnection, error) {
	conn := &DatadogConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	apiKey := os.Getenv("DD_API_KEY")
	appKey := os.Getenv("DD_APP_KEY")
	site := os.Getenv("DD_SITE")

	// Credentials can provide api-key and app-key via the user field
	for _, cred := range conf.Credentials {
		if cred.Type == vault.CredentialType_password {
			switch cred.User {
			case "app-key":
				appKey = string(cred.Secret)
			default:
				// Default password credential is the API key (backward-compatible)
				apiKey = string(cred.Secret)
			}
		}
	}

	if conf.Options != nil {
		if v, ok := conf.Options["app-key"]; ok && v != "" {
			appKey = v
		}
		if v, ok := conf.Options["site"]; ok && v != "" {
			site = v
		}
	}

	if apiKey == "" {
		return nil, errors.New("a valid Datadog API key is required (set DD_API_KEY or use --api-key)")
	}
	if appKey == "" {
		return nil, errors.New("a valid Datadog application key is required (set DD_APP_KEY or use --app-key)")
	}

	ctx := context.WithValue(
		context.Background(),
		datadog.ContextAPIKeys,
		map[string]datadog.APIKey{
			"apiKeyAuth": {Key: apiKey},
			"appKeyAuth": {Key: appKey},
		},
	)

	if site != "" {
		ctx = context.WithValue(ctx, datadog.ContextServerVariables, map[string]string{
			"site": site,
		})
	}

	configuration := datadog.NewConfiguration()
	conn.apiClient = datadog.NewAPIClient(configuration)
	conn.authCtx = ctx

	// Fetch org public ID for unique platform identification
	orgApi := datadogV1.NewOrganizationsApi(conn.apiClient)
	orgResp, _, err := orgApi.ListOrgs(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("datadog> could not fetch organization info for platform ID")
	} else if orgs := orgResp.GetOrgs(); len(orgs) > 0 {
		conn.orgPublicId = orgs[0].GetPublicId()
	}

	return conn, nil
}

func (c *DatadogConnection) Name() string {
	return "datadog"
}

func (c *DatadogConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *DatadogConnection) ApiClient() *datadog.APIClient {
	return c.apiClient
}

func (c *DatadogConnection) AuthCtx() context.Context {
	return c.authCtx
}

func (c *DatadogConnection) OrgPublicId() string {
	return c.orgPublicId
}
