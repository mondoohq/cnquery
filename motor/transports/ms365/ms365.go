package ms365

import (
	"context"
	"encoding/json"
	"os"

	"github.com/cockroachdb/errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/rs/zerolog/log"
	"github.com/yaegashi/msgraph.go/msauth"

	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
)

// New create a new Microsoft 365 transport
//
// At this point, this transports only supports application permissions
// because we are not able to get the user consent on cli yet. Seems like
// Microsoft is workin on some Powershell features that may make it happen.
//
// [How to recognize differences between delegated and application permissions](https://docs.microsoft.com/en-us/azure/active-directory/develop/delegated-and-app-perms)
// [Authentication and authorization basics for Microsoft Graph](https://docs.microsoft.com/en-us/graph/auth/auth-concepts)
// [Always check permissions in tokens in an Azure AD protected API](https://joonasw.net/view/always-check-token-permissions-in-aad-protected-api)
func New(tc *transports.TransportConfig) (*Transport, error) {
	if tc.Backend != transports.TransportBackend_CONNECTION_MS365 {
		return nil, errors.New("backend is not supported for ms365 transport")
	}

	if len(tc.IdentityFiles) != 1 {
		return nil, errors.New("ms365 backend requires a credentials file, pass json via -i option")
	}

	var msauth *MicrosoftAuth
	if len(tc.IdentityFiles) == 1 {

		filename := tc.IdentityFiles[0]

		f, err := os.Open(filename)
		if err != nil {
			return nil, errors.Wrap(err, "could not open credentials file")
		}

		msauth, err = ParseMicrosoftAuth(f)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse credentials file")
		}
	}

	if msauth == nil {
		return nil, errors.New("could not parse credentials file")
	}

	if len(msauth.TenantId) == 0 {
		return nil, errors.New("ms365 backend requires a tenantID")
	}

	t := &Transport{
		tenantID:     msauth.TenantId,
		opts:         tc.Options,
		clientID:     msauth.ClientId,
		clientSecret: msauth.ClientSecret,
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
	tenantID     string
	clientID     string
	clientSecret string
	opts         map[string]string
	rolesMap     map[string]struct{}
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
	return transports.Capabilities{}
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
