package ms365

import (
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/ms365/ms365report"
	"go.mondoo.com/cnquery/motor/vault"
)

const (
	OptionTenantID   = "tenant-id"
	OptionClientID   = "client-id"
	OptionDataReport = "mondoo-ms365-datareport"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

// New create a new Microsoft 365 provider
//
// At this point, this provider only supports application permissions
// because we are not able to get the user consent on cli yet. Seems like
// Microsoft is working on some Powershell features that may make it happen.
//
// For authentication we need a tenant id, client id (appid), and a certificate and an optional password
// mondoo scan -t ms365:// --certificate-path certificate --certificate-secret password --client-id CLIENT_ID --tenant-id TENANT_ID
//
// [How to recognize differences between delegated and application permissions](https://docs.microsoft.com/en-us/azure/active-directory/develop/delegated-and-app-perms)
// [Authentication and authorization basics for Microsoft Graph](https://docs.microsoft.com/en-us/graph/auth/auth-concepts)
// [Always check permissions in tokens in an Azure AD protected API](https://joonasw.net/view/always-check-token-permissions-in-aad-protected-api)
func New(pCfg *providers.Config) (*Provider, error) {
	if pCfg.Backend != providers.ProviderType_MS365 {
		return nil, providers.ErrProviderTypeDoesNotMatch
	}

	if len(pCfg.Credentials) != 1 || pCfg.Credentials[0] == nil {
		return nil, errors.New("ms365 provider requires a credentials file, pass path via --certificate-path option")
	}

	tenantId := pCfg.Options[OptionTenantID]
	clientId := pCfg.Options[OptionClientID]

	// deprecated options for backward compatibility with older inventory files
	if tenantId == "" {
		tid, ok := pCfg.Options["tenantId"]
		if ok {
			log.Warn().Str("tenantId", tid).Msg("tenantId is deprecated, use tenant-id instead")
		}
		tenantId = tid
	}

	if clientId == "" {
		cid, ok := pCfg.Options["clientId"]
		if ok {
			log.Warn().Str("clientId", cid).Msg("clientId is deprecated, use client-id instead")
		}
		clientId = cid
	}

	p := &Provider{
		tenantID: tenantId,
		clientID: clientId,
		// TODO: we want to remove the data report with a proper implementation
		powershellDataReportFile: pCfg.Options[OptionDataReport],
		opts:                     pCfg.Options,
		cred:                     pCfg.Credentials[0],
	}

	// we only support private key authentication and client secret for ms 365
	switch p.cred.Type {
	case vault.CredentialType_pkcs12:
	case vault.CredentialType_password:
	default:
		return nil, errors.New("invalid secret configuration for ms365 provider: " + p.cred.Type.String())
	}

	if len(p.tenantID) == 0 {
		return nil, errors.New("ms365 backend requires a tenant-id")
	}

	// map the roles that we request
	// TODO: check that actual credentials include permissions, this is included in the tokens
	p.rolesMap = map[string]struct{}{}
	for i := range DefaultRoles {
		r := DefaultRoles[i]
		p.rolesMap[r] = struct{}{}
	}

	return p, nil
}

type Provider struct {
	tenantID                    string
	clientID                    string
	cred                        *vault.Credential
	opts                        map[string]string
	rolesMap                    map[string]struct{}
	powershellDataReportFile    string
	ms365PowershellReport       *ms365report.Microsoft365Report
	ms365PowershellReportLoader sync.Mutex
}

func (p *Provider) MissingRoles(checkRoles ...string) []string {
	missing := []string{}
	for i := range checkRoles {
		_, ok := p.rolesMap[checkRoles[i]]
		if !ok {
			missing = append(missing, checkRoles[i])
		}
	}
	return missing
}

func (p *Provider) Close() {}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_Microsoft365,
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
