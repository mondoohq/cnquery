// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/keycloak/connection"
	"go.mondoo.com/mql/v13/types"
)

// mqlKeycloakClientInternal holds the realm the client belongs to, so the
// client's own lookups know which realm to address and the realm accessor
// costs no call.
type mqlKeycloakClientInternal struct {
	parentRealm *mqlKeycloakRealm
}

type clientRecord struct {
	ID                                 string            `json:"id"`
	ClientID                           string            `json:"clientId"`
	Name                               string            `json:"name"`
	Description                        string            `json:"description"`
	Enabled                            bool              `json:"enabled"`
	PublicClient                       bool              `json:"publicClient"`
	BearerOnly                         bool              `json:"bearerOnly"`
	StandardFlowEnabled                bool              `json:"standardFlowEnabled"`
	ImplicitFlowEnabled                bool              `json:"implicitFlowEnabled"`
	DirectAccessGrantsEnabled          bool              `json:"directAccessGrantsEnabled"`
	ServiceAccountsEnabled             bool              `json:"serviceAccountsEnabled"`
	AuthorizationServicesEnabled       bool              `json:"authorizationServicesEnabled"`
	RedirectUris                       []string          `json:"redirectUris"`
	WebOrigins                         []string          `json:"webOrigins"`
	RootURL                            string            `json:"rootUrl"`
	BaseURL                            string            `json:"baseUrl"`
	AdminURL                           string            `json:"adminUrl"`
	ConsentRequired                    bool              `json:"consentRequired"`
	FrontchannelLogout                 bool              `json:"frontchannelLogout"`
	Protocol                           string            `json:"protocol"`
	ClientAuthenticatorType            string            `json:"clientAuthenticatorType"`
	FullScopeAllowed                   bool              `json:"fullScopeAllowed"`
	Attributes                         map[string]string `json:"attributes"`
	AuthenticationFlowBindingOverrides map[string]string `json:"authenticationFlowBindingOverrides"`
	DefaultClientScopes                []string          `json:"defaultClientScopes"`
	OptionalClientScopes               []string          `json:"optionalClientScopes"`
}

// pkceAttribute is where Keycloak stores the proof key method a client must
// use for the authorization code exchange.
const pkceAttribute = "pkce.code.challenge.method"

func newKeycloakClient(runtime *plugin.Runtime, realm *mqlKeycloakRealm, rec *clientRecord) (*mqlKeycloakClient, error) {
	wildcards := WildcardRedirectUris(rec.RedirectUris)

	res, err := CreateResource(runtime, "keycloak.client", map[string]*llx.RawData{
		"__id":                               llx.StringData(realm.realmName() + "/client/" + rec.ID),
		"id":                                 llx.StringData(rec.ID),
		"clientId":                           llx.StringData(rec.ClientID),
		"name":                               llx.StringData(rec.Name),
		"description":                        llx.StringData(rec.Description),
		"enabled":                            llx.BoolData(rec.Enabled),
		"publicClient":                       llx.BoolData(rec.PublicClient),
		"bearerOnly":                         llx.BoolData(rec.BearerOnly),
		"standardFlowEnabled":                llx.BoolData(rec.StandardFlowEnabled),
		"implicitFlowEnabled":                llx.BoolData(rec.ImplicitFlowEnabled),
		"directAccessGrantsEnabled":          llx.BoolData(rec.DirectAccessGrantsEnabled),
		"serviceAccountsEnabled":             llx.BoolData(rec.ServiceAccountsEnabled),
		"authorizationServicesEnabled":       llx.BoolData(rec.AuthorizationServicesEnabled),
		"redirectUris":                       llx.ArrayData(strSliceToAny(rec.RedirectUris), types.String),
		"hasWildcardRedirectUri":             llx.BoolData(len(wildcards) > 0),
		"wildcardRedirectUris":               llx.ArrayData(strSliceToAny(wildcards), types.String),
		"webOrigins":                         llx.ArrayData(strSliceToAny(rec.WebOrigins), types.String),
		"hasWildcardWebOrigin":               llx.BoolData(HasWildcardWebOrigin(rec.WebOrigins)),
		"rootUrl":                            llx.StringData(rec.RootURL),
		"baseUrl":                            llx.StringData(rec.BaseURL),
		"adminUrl":                           llx.StringData(rec.AdminURL),
		"consentRequired":                    llx.BoolData(rec.ConsentRequired),
		"frontchannelLogout":                 llx.BoolData(rec.FrontchannelLogout),
		"protocol":                           llx.StringData(rec.Protocol),
		"clientAuthenticatorType":            llx.StringData(rec.ClientAuthenticatorType),
		"fullScopeAllowed":                   llx.BoolData(rec.FullScopeAllowed),
		"authenticationFlowBindingOverrides": llx.MapData(mapStrToAny(rec.AuthenticationFlowBindingOverrides), types.String),
		"pkceCodeChallengeMethod":            llx.StringData(rec.Attributes[pkceAttribute]),
		"attributes":                         llx.MapData(mapStrToAny(rec.Attributes), types.String),
		"defaultClientScopes":                llx.ArrayData(strSliceToAny(rec.DefaultClientScopes), types.String),
		"optionalClientScopes":               llx.ArrayData(strSliceToAny(rec.OptionalClientScopes), types.String),
	})
	if err != nil {
		return nil, err
	}

	client := res.(*mqlKeycloakClient)
	client.parentRealm = realm
	return client, nil
}

// WildcardRedirectUris returns the redirect URIs that carry a wildcard.
// Keycloak matches a `*` against any remainder of the URI, so an entry such as
// `https://app.example.com/*` accepts every path under the host and an entry of
// `*` accepts any URL at all. Whoever controls a matching URL collects the
// authorization codes the realm issues.
func WildcardRedirectUris(uris []string) []string {
	var wildcards []string
	for _, uri := range uris {
		if strings.Contains(uri, "*") {
			wildcards = append(wildcards, uri)
		}
	}
	return wildcards
}

// HasWildcardWebOrigin reports whether any web origin is the wildcard `*`,
// which lets a page on any origin call the realm from a browser. Keycloak also
// accepts `+`, which stands for the origins the redirect URIs imply and is
// therefore bounded by them.
func HasWildcardWebOrigin(origins []string) bool {
	for _, origin := range origins {
		if strings.TrimSpace(origin) == "*" {
			return true
		}
	}
	return false
}

func (c *mqlKeycloakClient) id() (string, error) {
	return c.__id, nil
}

func (c *mqlKeycloakClient) realm() (*mqlKeycloakRealm, error) {
	if c.parentRealm == nil {
		setNullResource(&c.Realm)
		return nil, nil
	}
	return c.parentRealm, nil
}

// roles lists the roles the client defines. They are the roles that appear in a
// token's resource_access claim, and the realm-management client's roles are
// what grant administration of the realm.
func (c *mqlKeycloakClient) roles() ([]any, error) {
	if c.parentRealm == nil {
		return nil, nil
	}

	ctx := context.Background()
	conn := keycloakConn(c.MqlRuntime)
	path := connection.AdminPath(c.parentRealm.realmName(), "clients", c.Id.Data, "roles")

	records, err := connection.GetPaged[roleRecord](ctx, conn, path, nil)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		role, err := newKeycloakRole(c.MqlRuntime, c.parentRealm, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, role)
	}
	return res, nil
}

// serviceAccountUser returns the user the client authenticates as. The account
// carries the client's role mappings, so it is where a service account that
// holds realm administration becomes visible.
func (c *mqlKeycloakClient) serviceAccountUser() (*mqlKeycloakUser, error) {
	if c.parentRealm == nil || !c.ServiceAccountsEnabled.Data {
		setNullResource(&c.ServiceAccountUser)
		return nil, nil
	}

	ctx := context.Background()
	conn := keycloakConn(c.MqlRuntime)
	path := connection.AdminPath(c.parentRealm.realmName(), "clients", c.Id.Data, "service-account-user")

	var rec userRecord
	if err := conn.Get(ctx, path, nil, &rec); err != nil {
		// A client can report service accounts as enabled while the account
		// itself is gone, and a token scoped below view-users cannot read it.
		// Both are reported as null rather than failing the client.
		if connection.IsNotFound(err) || connection.IsForbidden(err) {
			setNullResource(&c.ServiceAccountUser)
			return nil, nil
		}
		return nil, err
	}

	return newKeycloakUser(c.MqlRuntime, c.parentRealm, &rec)
}
