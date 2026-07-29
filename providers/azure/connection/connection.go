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
	// OptionAuthMethod names the sign-in method(s) to use when no client secret
	// or certificate is supplied, as a comma-separated list of
	// azauth.CredentialMethod values. Unset means try all of them.
	OptionAuthMethod = "auth-method"
)

type AzureConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset
	token azcore.TokenCredential
	// note: in the future, we might make this optional if we have a tenant-level asset.
	subscriptionId string
	clientOptions  policy.ClientOptions
	// Filters holds the parsed discovery filters (from inventory.Discovery.Filter).
	Filters DiscoveryFilters
}

// selectAzureCredential chooses the appropriate Azure token credential based on
// the connection configuration. When a federated token file is provided (via
// option or env var) and no explicit vault credential is present, it returns a
// WorkloadIdentityCredential for keyless auth. Otherwise it falls through to
// the standard cert/secret/default-chain path.
//
// The default chain is the expensive branch: it probes every sign-in method in
// turn, and the managed identity probe alone burns ~15s before giving up. That
// cost is paid per asset, since every discovered asset gets its own connection.
// A caller that knows how it authenticates can say so with OptionAuthMethod and
// skip straight to the method that works.
func selectAzureCredential(conf *inventory.Config) (azcore.TokenCredential, error) {
	tenantId := conf.Options[OptionTenantID]
	clientId := conf.Options[OptionClientID]

	methods, err := azauth.ParseCredentialMethods(conf.Options[OptionAuthMethod])
	if err != nil {
		return nil, err
	}

	var cred *vault.Credential
	if len(conf.Credentials) != 0 {
		cred = conf.Credentials[0]
	}

	federatedTokenFile := conf.Options[OptionFederatedTokenFile]
	if federatedTokenFile == "" {
		federatedTokenFile = os.Getenv("AZURE_FEDERATED_TOKEN_FILE")
	}

	// A token file with no method selection used to shortcut straight to a bare
	// workload identity credential here, on the grounds that there was nothing
	// left to probe. It cost more than it saved: the shortcut logged nothing, so
	// a connection that authenticated this way was invisible and the only
	// sign-ins in the logs were the ones configured some other way -- and the
	// credential it returned had no chain behind it, so a stale token file was
	// the end of the road rather than a fall back to the remaining methods.
	//
	// The chain is cheap to walk now that the managed identity probe sits at the
	// end of it (see azauth.DefaultCredentialMethods), so there is nothing left
	// to shortcut past.
	return azauth.GetTokenFromCredential(cred, tenantId, clientId, &azauth.ChainedTokenOptions{
		FederatedTokenFile: federatedTokenFile,
		Methods:            methods,
		Source:             "azure-connection",
	})
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
		Filters: DiscoveryFiltersFromOpts(conf.GetDiscover().GetFilter()),
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
