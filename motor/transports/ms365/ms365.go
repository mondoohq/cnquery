package ms365

import (
	"context"
	"encoding/json"
	"sync"

	"go.mondoo.io/mondoo/motor/vault"

	ms356_resources "go.mondoo.io/mondoo/lumi/resources/ms365"

	"github.com/cockroachdb/errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/rs/zerolog/log"
	"github.com/yaegashi/msgraph.go/msauth"

	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
)

const (
	OptionTenantID     = "tenantId"
	OptionClientID     = "clientId"
	OptionClientSecret = "clientSecret"
	OptionDataReport   = "mondoo-ms365-datareport"
)

var _ transports.Transport = (*Transport)(nil)
var _ transports.TransportPlatformIdentifier = (*Transport)(nil)

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

	if len(tc.Credentials) != 1 {
		return nil, errors.New("ms365 backend requires a credentials file, pass json via -i option")
	}

	cred := tc.Credentials[0]

	// we only support private key authentication for ms 365
	clientSecret := ""
	switch cred.Type {
	case vault.CredentialType_private_key:
		return nil, errors.New("certificate authentication is not implemented yet")
	case vault.CredentialType_password:
		clientSecret = string(cred.Secret)
	default:
		return nil, errors.New("invalid secret configuration for ms365 transport: " + cred.Type.String())
	}

	t := &Transport{
		tenantID: tc.Options[OptionTenantID],
		clientID: tc.Options[OptionClientID],
		// TODO: we want to support secret and certificate authentication
		clientSecret: clientSecret,
		// TODO: we want to remove the data report with a proper implementation
		powershellDataReportFile: tc.Options[OptionDataReport],
		opts:                     tc.Options,
	}

	if len(t.tenantID) == 0 {
		return nil, errors.New("ms365 backend requires a tenantID")
	}

	claims, err := t.TokenClaims()
	if err != nil {
		return nil, err
	}

	// cache roles from token
	rolesMap := map[string]struct{}{}
	for i := range claims.Roles {
		rolesMap[claims.Roles[i]] = struct{}{}
	}
	t.rolesMap = rolesMap

	if len(rolesMap) == 0 {
		log.Warn().Msg("your credentials do not include any permissions. please ensure you are using application permissions.")
	}

	data, err := json.Marshal(claims)
	if err == nil {
		log.Debug().Str("claims", string(data)).Msg("connect to microsoft 365")
	}

	return t, nil
}

type Transport struct {
	tenantID                    string
	clientID                    string
	clientSecret                string
	opts                        map[string]string
	rolesMap                    map[string]struct{}
	powershellDataReportFile    string
	ms365PowershellReport       *ms356_resources.Microsoft365Report
	ms365PowershellReportLoader sync.Mutex
}

func (t *Transport) TokenClaims() (*MicrosoftIdTokenClaims, error) {
	ctx := context.Background()
	m := msauth.NewManager()
	ts, err := m.ClientCredentialsGrant(ctx, t.tenantID, t.clientID, t.clientSecret, DefaultMSGraphScopes)
	if err != nil {
		return nil, err
	}

	token, err := ts.Token()
	if err != nil {
		return nil, err
	}

	claims := &MicrosoftIdTokenClaims{}
	p := jwt.Parser{}
	_, _, err = p.ParseUnverified(token.AccessToken, claims)
	if err != nil {
		return nil, err
	}
	return claims, nil
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
