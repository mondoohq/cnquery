package ms365

import (
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/spf13/afero"
	ms356_resources "go.mondoo.io/mondoo/lumi/resources/ms365"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
	"go.mondoo.io/mondoo/motor/vault"
)

const (
	OptionTenantID   = "tenantId"
	OptionClientID   = "clientId"
	OptionDataReport = "mondoo-ms365-datareport"
)

var (
	_ transports.Transport                   = (*Transport)(nil)
	_ transports.TransportPlatformIdentifier = (*Transport)(nil)
)

// New create a new Microsoft 365 transport
//
// At this point, this transports only supports application permissions
// because we are not able to get the user consent on cli yet. Seems like
// Microsoft is working on some Powershell features that may make it happen.
//
// For authentication we need a tenant id, client id (appid), and a certificate and an optional password
// mondoo scan -t ms365:// -i certificate --password password --option clientId --option tenantId
// mondoo scan -t ms365:// --password clientSecret --option clientId --option tenantId
//
// [How to recognize differences between delegated and application permissions](https://docs.microsoft.com/en-us/azure/active-directory/develop/delegated-and-app-perms)
// [Authentication and authorization basics for Microsoft Graph](https://docs.microsoft.com/en-us/graph/auth/auth-concepts)
// [Always check permissions in tokens in an Azure AD protected API](https://joonasw.net/view/always-check-token-permissions-in-aad-protected-api)
func New(tc *transports.TransportConfig) (*Transport, error) {
	if tc.Backend != transports.TransportBackend_CONNECTION_MS365 {
		return nil, errors.New("backend is not supported for ms365 transport")
	}

	if len(tc.Credentials) != 1 || tc.Credentials[0] == nil {
		return nil, errors.New("ms365 backend requires a credentials file, pass json via -i option")
	}

	t := &Transport{
		tenantID: tc.Options[OptionTenantID],
		clientID: tc.Options[OptionClientID],
		// TODO: we want to remove the data report with a proper implementation
		powershellDataReportFile: tc.Options[OptionDataReport],
		opts:                     tc.Options,
		cred:                     tc.Credentials[0],
	}

	// we only support private key authentication and client secret for ms 365
	switch t.cred.Type {
	case vault.CredentialType_pkcs12:
	case vault.CredentialType_password:
	default:
		return nil, errors.New("invalid secret configuration for ms365 transport: " + t.cred.Type.String())
	}

	if len(t.tenantID) == 0 {
		return nil, errors.New("ms365 backend requires a tenantID")
	}

	// map the roles that we request
	// TODO: check that actual credentials include permissions, this is included in the tokens
	t.rolesMap = map[string]struct{}{}
	for i := range DefaultRoles {
		r := DefaultRoles[i]
		t.rolesMap[r] = struct{}{}
	}

	return t, nil
}

type Transport struct {
	tenantID                    string
	clientID                    string
	cred                        *vault.Credential
	opts                        map[string]string
	rolesMap                    map[string]struct{}
	powershellDataReportFile    string
	ms365PowershellReport       *ms356_resources.Microsoft365Report
	ms365PowershellReportLoader sync.Mutex
}

func (t *Transport) MissingRoles(checkRoles ...string) []string {
	missing := []string{}
	for i := range checkRoles {
		_, ok := t.rolesMap[checkRoles[i]]
		if !ok {
			missing = append(missing, checkRoles[i])
		}
	}
	return missing
}

func (t *Transport) RunCommand(command string) (*transports.Command, error) {
	return nil, errors.New("ms365 does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("ms365 does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() transports.Capabilities {
	return transports.Capabilities{
		transports.Capability_Microsoft365,
	}
}

func (t *Transport) Options() map[string]string {
	return t.opts
}

func (t *Transport) Kind() transports.Kind {
	return transports.Kind_KIND_API
}

func (t *Transport) Runtime() string {
	return transports.RUNTIME_AZ
}

func (t *Transport) PlatformIdDetectors() []transports.PlatformIdDetector {
	return []transports.PlatformIdDetector{
		transports.TransportPlatformIdentifierDetector,
	}
}
