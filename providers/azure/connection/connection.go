// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"net/http"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/azauth"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
	"go.mondoo.com/mql/v13/providers/azure/connection/shared"
)

const (
	OptionTenantID           = "tenant-id"
	OptionClientID           = "client-id"
	OptionDataReport         = "mondoo-ms365-datareport"
	OptionSubscriptionID     = "subscription-id"
	OptionPlatformOverride   = "platform-override"
	OptionFederatedTokenFile = "azure-federated-token-file"
)

type AzureConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset
	token azcore.TokenCredential
	// note: in the future, we might make this optional if we have a tenant-level asset.
	subscriptionId string
	clientOptions  policy.ClientOptions
}

// selectAzureCredential chooses the appropriate Azure token credential based on
// the connection configuration. When a federated token file is provided (via
// option or env var) and no explicit vault credential is present, it returns a
// WorkloadIdentityCredential for keyless auth. Otherwise it falls through to
// the standard cert/secret/default-chain path.
func selectAzureCredential(conf *inventory.Config) (azcore.TokenCredential, error) {
	tenantId := conf.Options[OptionTenantID]
	clientId := conf.Options[OptionClientID]

	var cred *vault.Credential
	if len(conf.Credentials) != 0 {
		cred = conf.Credentials[0]
	}

	federatedTokenFile := conf.Options[OptionFederatedTokenFile]
	if federatedTokenFile == "" {
		federatedTokenFile = os.Getenv("AZURE_FEDERATED_TOKEN_FILE")
	}

	if cred == nil && federatedTokenFile != "" {
		return azauth.GetWorkloadIdentityToken(tenantId, clientId, federatedTokenFile)
	}
	return azauth.GetTokenFromCredential(cred, tenantId, clientId)
}

func NewAzureConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*AzureConnection, error) {
	subId := conf.Options[OptionSubscriptionID]

	token, err := selectAzureCredential(conf)
	if err != nil {
		return nil, errors.Wrap(err, "cannot fetch credentials for microsoft provider")
	}
	return &AzureConnection{
		Connection:     plugin.NewConnection(id, asset),
		Conf:           conf,
		asset:          asset,
		token:          token,
		subscriptionId: subId,
		clientOptions: policy.ClientOptions{
			PerCallPolicies: []policy.Policy{&apiTracePolicy{}},
		},
	}, nil
}

func (h *AzureConnection) Name() string {
	return "azure"
}

func (p *AzureConnection) Asset() *inventory.Asset {
	return p.asset
}

func (p *AzureConnection) SubId() string {
	return p.subscriptionId
}

func (p *AzureConnection) Token() azcore.TokenCredential {
	return p.token
}

func (p *AzureConnection) PlatformId() string {
	return "//platformid.api.mondoo.app/runtime/azure/subscriptions/" + p.subscriptionId
}

func (p *AzureConnection) ClientOptions() policy.ClientOptions {
	return p.clientOptions
}

func (p *AzureConnection) Config() *inventory.Config {
	return p.Conf
}

func (p *AzureConnection) Type() shared.ConnectionType {
	return "azure"
}

// apiTracePolicy is an Azure SDK pipeline policy that logs every HTTP request
// with its method, URL, status code, and duration at Debug level.
type apiTracePolicy struct{}

func (p *apiTracePolicy) Do(req *policy.Request) (*http.Response, error) {
	start := time.Now()
	rawReq := req.Raw()

	resp, err := req.Next()

	elapsed := time.Since(start)
	status := 0
	if resp != nil {
		status = resp.StatusCode
	}
	log.Debug().
		Str("method", rawReq.Method).
		// host+path only: the query string can carry SAS tokens or other
		// signed-URL credentials that must not leak into debug logs.
		Str("url", rawReq.URL.Host+rawReq.URL.Path).
		Int("status", status).
		Dur("duration", elapsed).
		Err(err).
		Msg("azure api call")

	return resp, err
}
