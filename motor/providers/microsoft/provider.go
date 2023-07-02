package microsoft

import (
	"sync"

	"errors"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/microsoft/ms365/ms365report"
	"go.mondoo.com/cnquery/motor/vault"
)

type microsoftAssetType int32

const (
	OptionTenantID         = "tenant-id"
	OptionClientID         = "client-id"
	OptionDataReport       = "mondoo-ms365-datareport"
	OptionSubscriptionID   = "subscription-id"
	OptionPlatformOverride = "platform-override"
)

const (
	ms365 microsoftAssetType = 0
	azure microsoftAssetType = 1
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

// New creates a new Microsoft provider that can be used against either Azure, MSGraph or both.
//
// At this point, this provider only supports application permissions
// because we are not able to get the user consent on cli yet. Seems like
// Microsoft is working on some PowerShell features that may make it happen.
//
// For authentication we need a tenant id, client id (appid), and either certificate or a client secret
// cnquery scan ms365 --certificate-path certificate --certificate-secret password --client-id CLIENT_ID --tenant-id TENANT_ID
// cnquery scan ms365 --client-secret password --client-id CLIENT_ID --tenant-id TENANT_ID

// Furthermore, this provider also supports authenticating against Azure. For this, it also requires a subscription
// cnquery scan azure --client-secret password --client-id CLIENT_ID --tenant-id TENANT_ID --subscription SUB_ID

// Depending on what parameters are passed, this provider will give access to different resources.
// > msgraph.* resources are always available if a client id, tenant id and a way to authenticate (password or cert) are provided.
// > azure.rm* resources are available only if a client id, tenant id, a way to authenticate AND a subscription is provided.

// [How to recognize differences between delegated and application permissions](https://docs.microsoft.com/en-us/azure/active-directory/develop/delegated-and-app-perms)
// [Authentication and authorization basics for Microsoft Graph](https://docs.microsoft.com/en-us/graph/auth/auth-concepts)
// [Always check permissions in tokens in an Azure AD protected API](https://joonasw.net/view/always-check-token-permissions-in-aad-protected-api)
func New(pCfg *providers.Config) (*Provider, error) {
	if pCfg.Backend != providers.ProviderType_MS365 && pCfg.Backend != providers.ProviderType_AZURE {
		return nil, providers.ErrProviderTypeDoesNotMatch
	}

	var assetType microsoftAssetType
	if pCfg.Backend == providers.ProviderType_MS365 {
		assetType = ms365
	} else if pCfg.Backend == providers.ProviderType_AZURE {
		assetType = azure
	}
	tenantId := pCfg.Options[OptionTenantID]
	clientId := pCfg.Options[OptionClientID]
	subscriptionId := pCfg.Options[OptionSubscriptionID]

	// we need credentials for ms365. for azure these are optional, we fallback to the AZ cli (if installed)
	if assetType == ms365 && (len(pCfg.Credentials) != 1 || pCfg.Credentials[0] == nil) {
		return nil, errors.New("microsoft provider requires a credentials file, pass path via --certificate-path option")
	}

	var cred *vault.Credential
	if len(pCfg.Credentials) != 0 {
		cred = pCfg.Credentials[0]
	}

	if assetType == ms365 && len(tenantId) == 0 {
		return nil, errors.New("ms365 backend requires a tenant-id")
	}

	p := &Provider{
		assetType:      assetType,
		tenantID:       tenantId,
		subscriptionID: subscriptionId,
		clientID:       clientId,
		// TODO: we want to remove the data report with a proper implementation
		powershellDataReportFile: pCfg.Options[OptionDataReport],
		opts:                     pCfg.Options,
		cred:                     cred,
		rolesMap:                 map[string]struct{}{},
		platformOverride:         pCfg.Options[OptionPlatformOverride],
	}
	// map the roles that we request
	// TODO: check that actual credentials include permissions, this is included in the tokens
	for i := range DefaultRoles {
		r := DefaultRoles[i]
		p.rolesMap[r] = struct{}{}
	}

	return p, nil
}

type Provider struct {
	assetType                   microsoftAssetType
	tenantID                    string
	clientID                    string
	subscriptionID              string
	cred                        *vault.Credential
	opts                        map[string]string
	rolesMap                    map[string]struct{}
	powershellDataReportFile    string
	ms365PowershellReport       *ms365report.Microsoft365Report
	ms365PowershellReportLoader sync.Mutex
	platformOverride            string
}

func (p *Provider) Close() {}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_Microsoft365,
		providers.Capability_Azure,
	}
}

func (p *Provider) Options() map[string]string {
	return p.opts
}

func (p *Provider) Kind() providers.Kind {
	return providers.Kind_KIND_API
}

func (p *Provider) Runtime() string {
	return providers.RUNTIME_AZ
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}
