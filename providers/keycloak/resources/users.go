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

// mqlKeycloakUserInternal holds the realm the account belongs to, which the
// role and group lookups address.
type mqlKeycloakUserInternal struct {
	parentRealm *mqlKeycloakRealm
}

type userRecord struct {
	ID                     string              `json:"id"`
	Username               string              `json:"username"`
	Email                  string              `json:"email"`
	FirstName              string              `json:"firstName"`
	LastName               string              `json:"lastName"`
	Enabled                bool                `json:"enabled"`
	EmailVerified          bool                `json:"emailVerified"`
	RequiredActions        []string            `json:"requiredActions"`
	CreatedTimestamp       int64               `json:"createdTimestamp"`
	FederationLink         string              `json:"federationLink"`
	ServiceAccountClientID string              `json:"serviceAccountClientId"`
	Attributes             map[string][]string `json:"attributes"`
}

// serviceAccountPrefix is the user name Keycloak gives the account a client
// authenticates as. It is the only marker on a server that omits
// serviceAccountClientId from a list response.
const serviceAccountPrefix = "service-account-"

func newKeycloakUser(runtime *plugin.Runtime, realm *mqlKeycloakRealm, rec *userRecord) (*mqlKeycloakUser, error) {
	res, err := CreateResource(runtime, "keycloak.user", map[string]*llx.RawData{
		"__id":                   llx.StringData(realm.realmName() + "/user/" + rec.ID),
		"id":                     llx.StringData(rec.ID),
		"username":               llx.StringData(rec.Username),
		"email":                  llx.StringData(rec.Email),
		"firstName":              llx.StringData(rec.FirstName),
		"lastName":               llx.StringData(rec.LastName),
		"enabled":                llx.BoolData(rec.Enabled),
		"emailVerified":          llx.BoolData(rec.EmailVerified),
		"requiredActions":        llx.ArrayData(strSliceToAny(rec.RequiredActions), types.String),
		"createdTimestamp":       llx.TimeDataPtr(epochMillisToTime(rec.CreatedTimestamp)),
		"federationLink":         llx.StringData(rec.FederationLink),
		"serviceAccountClientId": llx.StringData(ServiceAccountClientID(rec.ServiceAccountClientID, rec.Username)),
		"isServiceAccount":       llx.BoolData(IsServiceAccount(rec.ServiceAccountClientID, rec.Username)),
		"attributes":             llx.DictData(multiMapToDict(rec.Attributes)),
	})
	if err != nil {
		return nil, err
	}

	user := res.(*mqlKeycloakUser)
	user.parentRealm = realm
	return user, nil
}

// ServiceAccountClientID returns the client an account belongs to. Keycloak
// reports it directly on the service account endpoint, and only through the
// user name in a realm's user list, so the name is read when the field is
// absent.
func ServiceAccountClientID(reported, username string) string {
	if reported != "" {
		return reported
	}
	if after, ok := strings.CutPrefix(username, serviceAccountPrefix); ok {
		return after
	}
	return ""
}

// IsServiceAccount reports whether the account belongs to a client rather than
// to a person.
func IsServiceAccount(reported, username string) bool {
	return ServiceAccountClientID(reported, username) != ""
}

func (u *mqlKeycloakUser) id() (string, error) {
	return u.__id, nil
}

func (u *mqlKeycloakUser) realm() (*mqlKeycloakRealm, error) {
	if u.parentRealm == nil {
		setNullResource(&u.Realm)
		return nil, nil
	}
	return u.parentRealm, nil
}

// serviceAccountClient resolves the client the account belongs to through the
// realm's cached client list.
func (u *mqlKeycloakUser) serviceAccountClient() (*mqlKeycloakClient, error) {
	clientID := u.ServiceAccountClientId.Data
	if clientID == "" || u.parentRealm == nil {
		setNullResource(&u.ServiceAccountClient)
		return nil, nil
	}

	clients := u.parentRealm.GetClients()
	if clients.Error != nil {
		return nil, clients.Error
	}

	for _, it := range clients.Data {
		client, ok := it.(*mqlKeycloakClient)
		if ok && client.ClientId.Data == clientID {
			return client, nil
		}
	}

	setNullResource(&u.ServiceAccountClient)
	return nil, nil
}

func (u *mqlKeycloakUser) roles() ([]any, error) {
	if u.parentRealm == nil {
		return nil, nil
	}

	mappings, err := u.roleMappings()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(mappings.RealmMappings))
	for i := range mappings.RealmMappings {
		role, err := newKeycloakRole(u.MqlRuntime, u.parentRealm, &mappings.RealmMappings[i])
		if err != nil {
			return nil, err
		}
		res = append(res, role)
	}
	return res, nil
}

func (u *mqlKeycloakUser) groups() ([]any, error) {
	if u.parentRealm == nil {
		return nil, nil
	}

	ctx := context.Background()
	conn := keycloakConn(u.MqlRuntime)
	path := connection.AdminPath(u.parentRealm.realmName(), "users", u.Id.Data, "groups")

	records, err := connection.GetPaged[groupRecord](ctx, conn, path, connection.FullRepresentation())
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		group, err := newKeycloakGroup(u.MqlRuntime, u.parentRealm, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, group)
	}
	return res, nil
}

func (u *mqlKeycloakUser) roleMappings() (*roleMappingsRecord, error) {
	ctx := context.Background()
	conn := keycloakConn(u.MqlRuntime)
	path := connection.AdminPath(u.parentRealm.realmName(), "users", u.Id.Data, "role-mappings")

	var mappings roleMappingsRecord
	if err := conn.Get(ctx, path, nil, &mappings); err != nil {
		return nil, err
	}
	return &mappings, nil
}

// hasAdminRole reports whether the account holds a role that administers the
// realm. The direct mappings are expanded through their composites, because a
// role named after a team commonly carries realm-admin without saying so.
func (u *mqlKeycloakUser) hasAdminRole() (bool, error) {
	if u.parentRealm == nil {
		return false, nil
	}

	mappings, err := u.roleMappings()
	if err != nil {
		return false, err
	}

	refs, err := expandRoleRefs(u.MqlRuntime, u.parentRealm, mappings)
	if err != nil {
		return false, err
	}

	return HoldsAdminRole(refs), nil
}

// RoleRef names a role together with the client that defines it, which is what
// the admin check needs and the role records alone do not carry.
type RoleRef struct {
	Name       string
	ClientRole bool
	// ClientID is the client's own identifier, such as realm-management. It is
	// empty for a realm role, and also when the client could not be resolved.
	ClientID string
}

// adminClientRoles are the realm-management roles that administer a realm.
// Holding any one of them is enough to change how the realm authenticates.
var adminClientRoles = map[string]struct{}{
	"realm-admin":               {},
	"manage-realm":              {},
	"manage-users":              {},
	"manage-clients":            {},
	"manage-authorization":      {},
	"manage-identity-providers": {},
	"create-client":             {},
	"impersonation":             {},
}

// HoldsAdminRole reports whether any of the roles administers a realm. A realm
// role named admin is the master realm's superuser role, and the client roles
// are the realm-management ones that administer a single realm.
func HoldsAdminRole(roles []RoleRef) bool {
	for _, role := range roles {
		if !role.ClientRole {
			if role.Name == "admin" {
				return true
			}
			continue
		}
		// A client role only administers a realm when it belongs to a
		// realm-management client. An unresolved client is still checked,
		// since these role names exist nowhere else.
		if role.ClientID != "" && !isRealmManagementClient(role.ClientID) {
			continue
		}
		if _, ok := adminClientRoles[role.Name]; ok {
			return true
		}
	}
	return false
}

// isRealmManagementClient reports whether a client is the one that holds a
// realm's administration roles. A realm holds them on its own
// realm-management client, and the master realm holds another realm's on a
// client named after that realm.
func isRealmManagementClient(clientID string) bool {
	return clientID == "realm-management" || strings.HasSuffix(clientID, "-realm")
}

// maxCompositeLookups bounds how many composite expansions one admin check
// performs. A realm can nest composites deeply, and the cap keeps a single
// field from walking the whole role graph.
const maxCompositeLookups = 100

// expandRoleRefs turns the direct role mappings into every role they grant,
// following composites. A composite can nest further composites, so the walk
// continues until nothing new is found.
func expandRoleRefs(runtime *plugin.Runtime, realm *mqlKeycloakRealm, mappings *roleMappingsRecord) ([]RoleRef, error) {
	clientIDs := clientIDsByUUID(realm)

	type pending struct {
		rec      roleRecord
		clientID string
	}

	queue := make([]pending, 0, len(mappings.RealmMappings))
	for _, rec := range mappings.RealmMappings {
		queue = append(queue, pending{rec: rec})
	}
	for clientID, mapping := range mappings.ClientMappings {
		name := mapping.Client
		if name == "" {
			name = clientID
		}
		for _, rec := range mapping.Mappings {
			queue = append(queue, pending{rec: rec, clientID: name})
		}
	}

	refs := make([]RoleRef, 0, len(queue))
	visited := map[string]struct{}{}
	lookups := 0

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if item.rec.ID != "" {
			if _, seen := visited[item.rec.ID]; seen {
				continue
			}
			visited[item.rec.ID] = struct{}{}
		}

		refs = append(refs, RoleRef{
			Name:       item.rec.Name,
			ClientRole: item.rec.ClientRole,
			ClientID:   item.clientID,
		})

		if !item.rec.Composite || lookups >= maxCompositeLookups {
			continue
		}
		lookups++

		composites, err := fetchRoleComposites(runtime, realm, item.rec.ID)
		if err != nil {
			// A role whose composites cannot be read leaves the roles it
			// grants unknown. Reporting the field as a failure is honest,
			// since answering false would claim the account holds no
			// administration role when that was never established.
			return nil, err
		}

		for _, rec := range composites {
			clientID := ""
			if rec.ClientRole {
				clientID = clientIDs[rec.ContainerID]
			}
			queue = append(queue, pending{rec: rec, clientID: clientID})
		}
	}

	return refs, nil
}

// clientIDsByUUID maps each client's internal identifier to the identifier it
// is addressed by, which is what a composite role's container names. An
// unreadable client list yields an empty map, and the admin check then falls
// back to matching the role name alone.
func clientIDsByUUID(realm *mqlKeycloakRealm) map[string]string {
	ids := map[string]string{}

	clients := realm.GetClients()
	if clients.Error != nil {
		return ids
	}

	for _, it := range clients.Data {
		client, ok := it.(*mqlKeycloakClient)
		if ok {
			ids[client.Id.Data] = client.ClientId.Data
		}
	}
	return ids
}
