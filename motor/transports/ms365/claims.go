package ms365

import (
	"github.com/cockroachdb/errors"
	"github.com/dgrijalva/jwt-go"
)

// https://docs.microsoft.com/en-us/azure/active-directory/develop/id-tokens
// https://docs.microsoft.com/en-us/azure/active-directory/develop/access-tokens
// check the tokens via https://jwt.ms/
type MicrosoftIdTokenClaims struct {
	jwt.StandardClaims
	Audience  string `json:"aud,omitempty"`
	ExpiresAt int64  `json:"exp,omitempty"`
	Id        string `json:"jti,omitempty"`
	IssuedAt  int64  `json:"iat,omitempty"`
	Issuer    string `json:"iss,omitempty"`
	NotBefore int64  `json:"nbf,omitempty"`
	Subject   string `json:"sub,omitempty"`

	// Azure AD (internal id)
	Aio string `json:"aio,omitempty"`
	// An internal claim used by Azure to revalidate tokens
	Rh string `json:"rh,omitempty"`

	AppDisplayName string `json:"app_displayname,omitempty"`
	// Only present in v1.0 tokens. The application ID of the client using the token.
	AppId string `json:"appid,omitempty"`
	// Only present in v1.0 tokens. Indicates how the client was authenticated.
	AppIdAcr string `json:"appidacr,omitempty"`
	// Records the identity provider that authenticated the subject of the token.
	Idp string `json:"idp,omitempty"`
	// distinguish between app-only access tokens and access tokens for users
	IdpTyp string `json:"idtyp,omitempty"`
	// The immutable identifier for an object in the Microsoft identity system, in this case, a user account.
	Oid string `json:"oid,omitempty"`
	// The set of roles that were assigned to the user who is logging in.
	Roles             []string `json:"roles,omitempty"`
	TenantRegionScope string   `json:"tenant_region_scope,omitempty"`
	// A GUID that represents the Azure AD tenant
	Tid string `json:"tid,omitempty"`
	// An internal claim used by Azure to revalidate tokens
	Uti string `json:"uti,omitempty"`
	// Indicates the version of the id_token
	Ver     string `json:"ver,omitempty"`
	XmsTcdt int64  `json:"xms_tcdt,omitempty"`
}

func (m MicrosoftIdTokenClaims) Valid() error {
	return errors.New("not implemented")
}
