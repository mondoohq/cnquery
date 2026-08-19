// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"sync"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/types"
)

const (
	// anonymousTokenAccessorID is the reserved accessor of the token every
	// request without a token of its own resolves to.
	anonymousTokenAccessorID = "00000000-0000-0000-0000-000000000002"

	// globalManagementPolicyID is the reserved identifier of the built-in
	// policy granting read and write on every Consul feature. Matching on it
	// as well as on the name means renaming the policy does not hide it.
	globalManagementPolicyID = "00000000-0000-0000-0000-000000000001"

	// globalManagementPolicyName is the name that same policy ships under.
	globalManagementPolicyName = "global-management"

	// aclPolicyDeny is the default policy value that denies an unmatched
	// request.
	aclPolicyDeny = "deny"
)

// mqlConsulAclSystemInternal caches the policy and role inventories. Tokens and
// roles resolve their grants against these, so a datacenter with many tokens
// costs one list call per kind rather than one lookup per link.
type mqlConsulAclSystemInternal struct {
	policyOnce   sync.Once
	policyList   []any
	policyErr    error
	policyByID   map[string]*mqlConsulAclPolicy
	policyByName map[string]*mqlConsulAclPolicy

	roleOnce   sync.Once
	roleList   []any
	roleErr    error
	roleByID   map[string]*mqlConsulAclRole
	roleByName map[string]*mqlConsulAclRole
}

// mqlConsulAclTokenInternal carries the grant links the token was listed with,
// plus the ACL system they resolve against.
type mqlConsulAclTokenInternal struct {
	cachedSystem      *mqlConsulAclSystem
	cachedPolicyLinks []*consulapi.ACLLink
	cachedRoleLinks   []*consulapi.ACLLink
}

// mqlConsulAclRoleInternal carries the policy links the role was listed with,
// plus the ACL system they resolve against.
type mqlConsulAclRoleInternal struct {
	cachedSystem      *mqlConsulAclSystem
	cachedPolicyLinks []*consulapi.ACLLink
}

// aclDefaultDeny reports whether an unmatched request is denied. A default
// policy of "deny" on an agent with its ACL system switched off denies nothing,
// so both halves have to hold.
func aclDefaultDeny(enabled bool, defaultPolicy string) bool {
	return enabled && strings.EqualFold(strings.TrimSpace(defaultPolicy), aclPolicyDeny)
}

// linksIncludeGlobalManagement reports whether one of the links names the
// built-in unrestricted policy, by reserved identifier or by name.
func linksIncludeGlobalManagement(links []*consulapi.ACLLink) bool {
	for _, link := range links {
		if link == nil {
			continue
		}
		if link.ID == globalManagementPolicyID || link.Name == globalManagementPolicyName {
			return true
		}
	}
	return false
}

// isGlobalManagementPolicy reports whether a policy is the built-in
// unrestricted one.
func isGlobalManagementPolicy(id, name string) bool {
	return id == globalManagementPolicyID || name == globalManagementPolicyName
}

// aclTokenRecord is the shape both the token list and a single token read are
// normalized into, so one mapping serves both and the derived predicates have
// one input type to be tested against.
type aclTokenRecord struct {
	AccessorID        string
	Description       string
	Local             bool
	AuthMethod        string
	ExpirationTime    *time.Time
	CreateTime        time.Time
	Policies          []*consulapi.ACLLink
	Roles             []*consulapi.ACLLink
	ServiceIdentities []*consulapi.ACLServiceIdentity
	NodeIdentities    []*consulapi.ACLNodeIdentity
	TemplatedPolicies []*consulapi.ACLTemplatedPolicy
}

// tokenRecordFromListEntry normalizes an entry of the token list. The secret
// half of the token is deliberately not carried across, even though the list
// endpoint returns it to a caller holding a management token.
func tokenRecordFromListEntry(entry *consulapi.ACLTokenListEntry) aclTokenRecord {
	if entry == nil {
		return aclTokenRecord{}
	}
	return aclTokenRecord{
		AccessorID:        entry.AccessorID,
		Description:       entry.Description,
		Local:             entry.Local,
		AuthMethod:        entry.AuthMethod,
		ExpirationTime:    entry.ExpirationTime,
		CreateTime:        entry.CreateTime,
		Policies:          entry.Policies,
		Roles:             entry.Roles,
		ServiceIdentities: entry.ServiceIdentities,
		NodeIdentities:    entry.NodeIdentities,
		TemplatedPolicies: entry.TemplatedPolicies,
	}
}

// tokenRecordFromToken normalizes a single token read, dropping its secret for
// the same reason.
func tokenRecordFromToken(token *consulapi.ACLToken) aclTokenRecord {
	if token == nil {
		return aclTokenRecord{}
	}
	return aclTokenRecord{
		AccessorID:        token.AccessorID,
		Description:       token.Description,
		Local:             token.Local,
		AuthMethod:        token.AuthMethod,
		ExpirationTime:    token.ExpirationTime,
		CreateTime:        token.CreateTime,
		Policies:          token.Policies,
		Roles:             token.Roles,
		ServiceIdentities: token.ServiceIdentities,
		NodeIdentities:    token.NodeIdentities,
		TemplatedPolicies: token.TemplatedPolicies,
	}
}

// hasGrants reports whether the token authorizes anything. A token with no
// grants still authenticates, but authorizes no request.
func (r aclTokenRecord) hasGrants() bool {
	return len(r.Policies) > 0 ||
		len(r.Roles) > 0 ||
		len(r.ServiceIdentities) > 0 ||
		len(r.NodeIdentities) > 0 ||
		len(r.TemplatedPolicies) > 0
}

// serviceIdentityDicts renders service identities for the schema. Nil entries
// are skipped rather than rendered as an identity with no service name.
func serviceIdentityDicts(identities []*consulapi.ACLServiceIdentity) []any {
	res := make([]any, 0, len(identities))
	for _, identity := range identities {
		if identity == nil {
			continue
		}
		res = append(res, map[string]any{
			"serviceName": identity.ServiceName,
			"datacenters": convert.SliceAnyToInterface(identity.Datacenters),
		})
	}
	return res
}

// nodeIdentityDicts renders node identities for the schema.
func nodeIdentityDicts(identities []*consulapi.ACLNodeIdentity) []any {
	res := make([]any, 0, len(identities))
	for _, identity := range identities {
		if identity == nil {
			continue
		}
		res = append(res, map[string]any{
			"nodeName":   identity.NodeName,
			"datacenter": identity.Datacenter,
		})
	}
	return res
}

// templatedPolicyDicts renders templated policies for the schema. The template
// variables block is optional, so it is rendered as an empty object rather than
// omitted, which keeps the shape the same for every entry.
func templatedPolicyDicts(policies []*consulapi.ACLTemplatedPolicy) []any {
	res := make([]any, 0, len(policies))
	for _, policy := range policies {
		if policy == nil {
			continue
		}
		variables := map[string]any{}
		if policy.TemplateVariables != nil {
			variables["name"] = policy.TemplateVariables.Name
		}
		res = append(res, map[string]any{
			"templateName":      policy.TemplateName,
			"templateVariables": variables,
			"datacenters":       convert.SliceAnyToInterface(policy.Datacenters),
		})
	}
	return res
}

// nullableTime renders a timestamp the API leaves at its zero value as null
// rather than as a date in year one.
func nullableTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

// aclEnabled reports whether the ACL system is switched on, surfacing the error
// if that could not be read. Treating an unread flag as "off" would turn a
// failure to reach the agent into an empty inventory, and every check written
// against that inventory would pass.
func (r *mqlConsulAclSystem) aclEnabled() (bool, error) {
	enabled := r.GetEnabled()
	if enabled.Error != nil {
		return false, enabled.Error
	}
	return enabled.Data, nil
}

func (r *mqlConsulAclSystem) tokens() ([]any, error) {
	enabled, err := r.aclEnabled()
	if err != nil {
		return nil, err
	}
	if !enabled {
		// The token endpoint does not exist on an agent with ACLs switched
		// off, so an empty inventory is a fact about the agent.
		return []any{}, nil
	}

	client, err := consulClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	entries, _, err := client.ACL().TokenList(nil)
	if err != nil {
		if isACLSystemDisabled(err) {
			return []any{}, nil
		}
		return nil, err
	}

	res := make([]any, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		token, err := r.newToken(tokenRecordFromListEntry(entry))
		if err != nil {
			return nil, err
		}
		res = append(res, token)
	}
	return res, nil
}

func (r *mqlConsulAclSystem) anonymousToken() (*mqlConsulAclToken, error) {
	enabled, err := r.aclEnabled()
	if err != nil {
		return nil, err
	}
	if !enabled {
		r.AnonymousToken.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	client, err := consulClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	token, _, err := client.ACL().TokenRead(anonymousTokenAccessorID, nil)
	if err != nil {
		if isACLSystemDisabled(err) || isNotFound(err) {
			// The reserved token is absent. A permission failure is not an
			// absence and falls through to the error below.
			r.AnonymousToken.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}
	if token == nil {
		r.AnonymousToken.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	return r.newToken(tokenRecordFromToken(token))
}

// newToken maps one normalized token record onto the schema.
func (r *mqlConsulAclSystem) newToken(record aclTokenRecord) (*mqlConsulAclToken, error) {
	res, err := CreateResource(r.MqlRuntime, "consul.acl.token", map[string]*llx.RawData{
		"__id":               llx.StringData(record.AccessorID),
		"accessorId":         llx.StringData(record.AccessorID),
		"description":        llx.StringData(record.Description),
		"local":              llx.BoolData(record.Local),
		"authMethod":         llx.StringData(record.AuthMethod),
		"expirationTime":     llx.TimeDataPtr(record.ExpirationTime),
		"createTime":         llx.TimeDataPtr(nullableTime(record.CreateTime)),
		"hasGrants":          llx.BoolData(record.hasGrants()),
		"isGlobalManagement": llx.BoolData(linksIncludeGlobalManagement(record.Policies)),
		"serviceIdentities":  llx.ArrayData(serviceIdentityDicts(record.ServiceIdentities), types.Dict),
		"nodeIdentities":     llx.ArrayData(nodeIdentityDicts(record.NodeIdentities), types.Dict),
		"templatedPolicies":  llx.ArrayData(templatedPolicyDicts(record.TemplatedPolicies), types.Dict),
	})
	if err != nil {
		return nil, err
	}

	token := res.(*mqlConsulAclToken)
	token.cachedSystem = r
	token.cachedPolicyLinks = record.Policies
	token.cachedRoleLinks = record.Roles
	return token, nil
}

func (r *mqlConsulAclSystem) policies() ([]any, error) {
	list, _, _, err := r.fetchPolicies()
	return list, err
}

// fetchPolicies reads the policy inventory once and indexes it, so tokens and
// roles resolve their links without a call per link.
func (r *mqlConsulAclSystem) fetchPolicies() ([]any, map[string]*mqlConsulAclPolicy, map[string]*mqlConsulAclPolicy, error) {
	r.policyOnce.Do(func() {
		r.policyList = []any{}
		r.policyByID = map[string]*mqlConsulAclPolicy{}
		r.policyByName = map[string]*mqlConsulAclPolicy{}

		enabled, err := r.aclEnabled()
		if err != nil {
			r.policyErr = err
			return
		}
		if !enabled {
			return
		}

		client, err := consulClient(r.MqlRuntime)
		if err != nil {
			r.policyErr = err
			return
		}

		entries, _, err := client.ACL().PolicyList(nil)
		if err != nil {
			if isACLSystemDisabled(err) {
				return
			}
			r.policyErr = err
			return
		}

		for _, entry := range entries {
			if entry == nil {
				continue
			}
			policy, err := r.newPolicy(entry.ID, entry.Name, entry.Description, entry.Datacenters)
			if err != nil {
				r.policyErr = err
				return
			}
			r.policyList = append(r.policyList, policy)
			r.policyByID[entry.ID] = policy
			r.policyByName[entry.Name] = policy
		}
	})
	return r.policyList, r.policyByID, r.policyByName, r.policyErr
}

// newPolicy maps one policy onto the schema. Rules are not read here: the list
// endpoint does not carry them, and fetching them per policy would cost a call
// per policy on every query that only wants names.
func (r *mqlConsulAclSystem) newPolicy(id, name, description string, datacenters []string) (*mqlConsulAclPolicy, error) {
	res, err := CreateResource(r.MqlRuntime, "consul.acl.policy", map[string]*llx.RawData{
		"__id":               llx.StringData(id),
		"id":                 llx.StringData(id),
		"name":               llx.StringData(name),
		"description":        llx.StringData(description),
		"datacenters":        llx.ArrayData(convert.SliceAnyToInterface(datacenters), types.String),
		"isGlobalManagement": llx.BoolData(isGlobalManagementPolicy(id, name)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlConsulAclPolicy), nil
}

func (r *mqlConsulAclSystem) roles() ([]any, error) {
	list, _, _, err := r.fetchRoles()
	return list, err
}

// fetchRoles reads the role inventory once and indexes it, for the same reason
// fetchPolicies does.
func (r *mqlConsulAclSystem) fetchRoles() ([]any, map[string]*mqlConsulAclRole, map[string]*mqlConsulAclRole, error) {
	r.roleOnce.Do(func() {
		r.roleList = []any{}
		r.roleByID = map[string]*mqlConsulAclRole{}
		r.roleByName = map[string]*mqlConsulAclRole{}

		enabled, err := r.aclEnabled()
		if err != nil {
			r.roleErr = err
			return
		}
		if !enabled {
			return
		}

		client, err := consulClient(r.MqlRuntime)
		if err != nil {
			r.roleErr = err
			return
		}

		entries, _, err := client.ACL().RoleList(nil)
		if err != nil {
			if isACLSystemDisabled(err) {
				return
			}
			r.roleErr = err
			return
		}

		for _, entry := range entries {
			if entry == nil {
				continue
			}
			role, err := r.newRole(entry)
			if err != nil {
				r.roleErr = err
				return
			}
			r.roleList = append(r.roleList, role)
			r.roleByID[entry.ID] = role
			r.roleByName[entry.Name] = role
		}
	})
	return r.roleList, r.roleByID, r.roleByName, r.roleErr
}

// newRole maps one role onto the schema.
func (r *mqlConsulAclSystem) newRole(entry *consulapi.ACLRole) (*mqlConsulAclRole, error) {
	res, err := CreateResource(r.MqlRuntime, "consul.acl.role", map[string]*llx.RawData{
		"__id":              llx.StringData(entry.ID),
		"id":                llx.StringData(entry.ID),
		"name":              llx.StringData(entry.Name),
		"description":       llx.StringData(entry.Description),
		"serviceIdentities": llx.ArrayData(serviceIdentityDicts(entry.ServiceIdentities), types.Dict),
		"nodeIdentities":    llx.ArrayData(nodeIdentityDicts(entry.NodeIdentities), types.Dict),
		"templatedPolicies": llx.ArrayData(templatedPolicyDicts(entry.TemplatedPolicies), types.Dict),
	})
	if err != nil {
		return nil, err
	}

	role := res.(*mqlConsulAclRole)
	role.cachedSystem = r
	role.cachedPolicyLinks = entry.Policies
	return role, nil
}

// resolvePolicyLinks turns grant links into policy resources. Links are matched
// against the indexed inventory first, by identifier and then by name, because
// a link created by name carries no identifier until the server fills it in. A
// link matching nothing in the inventory is still returned, built from what the
// link itself carries, so a policy created between the two reads shortens
// nobody's list.
func resolvePolicyLinks(system *mqlConsulAclSystem, links []*consulapi.ACLLink) ([]any, error) {
	if len(links) == 0 {
		return []any{}, nil
	}
	if system == nil {
		return nil, errNoAclSystem
	}

	_, byID, byName, err := system.fetchPolicies()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(links))
	for _, link := range links {
		if link == nil {
			continue
		}
		if policy, ok := byID[link.ID]; ok && link.ID != "" {
			res = append(res, policy)
			continue
		}
		if policy, ok := byName[link.Name]; ok && link.Name != "" {
			res = append(res, policy)
			continue
		}
		policy, err := system.newPolicy(link.ID, link.Name, "", nil)
		if err != nil {
			return nil, err
		}
		res = append(res, policy)
	}
	return res, nil
}

// resolveRoleLinks turns role links into role resources, with the same matching
// and fallback rules as resolvePolicyLinks.
func resolveRoleLinks(system *mqlConsulAclSystem, links []*consulapi.ACLLink) ([]any, error) {
	if len(links) == 0 {
		return []any{}, nil
	}
	if system == nil {
		return nil, errNoAclSystem
	}

	_, byID, byName, err := system.fetchRoles()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(links))
	for _, link := range links {
		if link == nil {
			continue
		}
		if role, ok := byID[link.ID]; ok && link.ID != "" {
			res = append(res, role)
			continue
		}
		if role, ok := byName[link.Name]; ok && link.Name != "" {
			res = append(res, role)
			continue
		}
		role, err := system.newRole(&consulapi.ACLRole{ID: link.ID, Name: link.Name})
		if err != nil {
			return nil, err
		}
		res = append(res, role)
	}
	return res, nil
}

func (r *mqlConsulAclToken) policies() ([]any, error) {
	return resolvePolicyLinks(r.cachedSystem, r.cachedPolicyLinks)
}

func (r *mqlConsulAclToken) roles() ([]any, error) {
	return resolveRoleLinks(r.cachedSystem, r.cachedRoleLinks)
}

func (r *mqlConsulAclRole) policies() ([]any, error) {
	return resolvePolicyLinks(r.cachedSystem, r.cachedPolicyLinks)
}

func (r *mqlConsulAclPolicy) rules() (string, error) {
	id := r.GetId().Data
	if id == "" {
		// A policy known only by name carries no handle to read its document
		// with, so the field is null rather than an empty document, which
		// would read as a policy granting nothing.
		r.Rules.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}

	client, err := consulClient(r.MqlRuntime)
	if err != nil {
		return "", err
	}

	policy, _, err := client.ACL().PolicyRead(id, nil)
	if err != nil {
		if isNotFound(err) {
			r.Rules.State = plugin.StateIsSet | plugin.StateIsNull
			return "", nil
		}
		return "", err
	}
	if policy == nil {
		r.Rules.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}
	return policy.Rules, nil
}
