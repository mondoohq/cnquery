// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"strconv"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// -----------------------------------------------------------------------------
// account
// -----------------------------------------------------------------------------

func newAccountResource(runtime *plugin.Runtime, account apiAccount) (*mqlNewrelicAccount, error) {
	res, err := CreateResource(runtime, "newrelic.account", map[string]*llx.RawData{
		"__id": llx.StringData("account/" + strconv.Itoa(account.ID)),
		"id":   llx.IntData(int64(account.ID)),
		"name": llx.StringData(account.Name),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlNewrelicAccount), nil
}

// initNewrelicAccount resolves an account selected by ID against the accounts
// the supplied key can read. An ID the key cannot see is reported as not found
// rather than turned into a blank account, whose every field would read as
// unset.
func initNewrelicAccount(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// The caller already has everything the resource holds, so there is nothing
	// to look up.
	if _, ok := args["name"]; ok {
		return args, nil, nil
	}

	rawID, ok := args["id"]
	if !ok {
		return nil, nil, errors.New("newrelic.account needs an account id, for example newrelic.account(id: 1234567)")
	}
	wanted, ok := rawID.Value.(int64)
	if !ok {
		return nil, nil, fmt.Errorf("newrelic.account needs a numeric id, got %T", rawID.Value)
	}

	accounts, err := cachedAccounts(runtime)
	if err != nil {
		return nil, nil, err
	}
	for _, account := range accounts {
		if int64(account.ID) == wanted {
			mqlAccount, err := newAccountResource(runtime, account)
			if err != nil {
				return nil, nil, err
			}
			return args, mqlAccount, nil
		}
	}

	return nil, nil, fmt.Errorf("newrelic.account with id %d not found", wanted)
}

// resolveAccount hands back the account resource for an ID, or null when the
// supplied key cannot read that account. An account ID of zero means the record
// is not scoped to an account at all.
func resolveAccount(runtime *plugin.Runtime, accountID int) (*mqlNewrelicAccount, bool, error) {
	if accountID <= 0 {
		return nil, false, nil
	}

	accounts, err := cachedAccounts(runtime)
	if err != nil {
		return nil, false, err
	}
	for _, account := range accounts {
		if account.ID == accountID {
			mqlAccount, err := newAccountResource(runtime, account)
			if err != nil {
				return nil, false, err
			}
			return mqlAccount, true, nil
		}
	}
	return nil, false, nil
}

// -----------------------------------------------------------------------------
// authentication domain
// -----------------------------------------------------------------------------

func newAuthDomainResource(runtime *plugin.Runtime, domain apiAuthDomain) (*mqlNewrelicAuthenticationDomain, error) {
	res, err := CreateResource(runtime, "newrelic.authenticationDomain", map[string]*llx.RawData{
		"__id":             llx.StringData("authenticationDomain/" + domain.ID),
		"id":               llx.StringData(domain.ID),
		"name":             llx.StringData(domain.Name),
		"provisioningType": llx.StringData(domain.ProvisioningType),
		"scimEnabled":      llx.BoolData(isScimProvisioning(domain.ProvisioningType)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlNewrelicAuthenticationDomain), nil
}

// authenticationTypeFor reads a domain's login method out of the customer
// administration view.
//
// A domain the view does not return is an error rather than an empty string.
// The two cases that produce it are a key without organization administration
// access and a domain that has just been created, and neither is evidence that
// the domain lacks single sign-on, which is what an empty value would be read
// as.
func authenticationTypeFor(runtime *plugin.Runtime, domainID string) (string, error) {
	byID, err := cachedAuthTypes(runtime)
	if err != nil {
		return "", err
	}
	domain, ok := byID[domainID]
	if !ok {
		return "", fmt.Errorf("the New Relic API did not report a login method for authentication domain %q, so it cannot be checked", domainID)
	}
	return string(domain.AuthenticationType), nil
}

func (r *mqlNewrelicAuthenticationDomain) authenticationType() (string, error) {
	return authenticationTypeFor(r.MqlRuntime, r.Id.Data)
}

func (r *mqlNewrelicAuthenticationDomain) ssoEnabled() (bool, error) {
	authType, err := authenticationTypeFor(r.MqlRuntime, r.Id.Data)
	if err != nil {
		return false, err
	}
	return isSSOAuthentication(authType), nil
}

func (r *mqlNewrelicAuthenticationDomain) passwordLoginEnabled() (bool, error) {
	authType, err := authenticationTypeFor(r.MqlRuntime, r.Id.Data)
	if err != nil {
		return false, err
	}
	return isPasswordAuthentication(authType), nil
}

func (r *mqlNewrelicAuthenticationDomain) users() ([]any, error) {
	domains, err := cachedAuthDomains(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, domain := range domains {
		if domain.ID != r.Id.Data {
			continue
		}
		for _, user := range domain.Users.Users {
			mqlUser, err := newUserResource(r.MqlRuntime, user)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlUser)
		}
	}
	return res, nil
}

func (r *mqlNewrelicAuthenticationDomain) groups() ([]any, error) {
	groups, err := cachedGroups(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, group := range groups {
		if group.domainID != r.Id.Data {
			continue
		}
		mqlGroup, err := newGroupResource(r.MqlRuntime, group)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlGroup)
	}
	return res, nil
}

// resolveAuthDomain hands back the domain resource for an ID, or null when the
// domain is not in the organization's domain list.
func resolveAuthDomain(runtime *plugin.Runtime, domainID string) (*mqlNewrelicAuthenticationDomain, bool, error) {
	if domainID == "" {
		return nil, false, nil
	}

	domains, err := cachedAuthDomains(runtime)
	if err != nil {
		return nil, false, err
	}
	for _, domain := range domains {
		if domain.ID == domainID {
			mqlDomain, err := newAuthDomainResource(runtime, domain)
			if err != nil {
				return nil, false, err
			}
			return mqlDomain, true, nil
		}
	}
	return nil, false, nil
}

// -----------------------------------------------------------------------------
// user
// -----------------------------------------------------------------------------

// mqlNewrelicUserInternal keeps the identifiers a user's references need. The
// authentication domain is not repeated on the user record itself, so it has to
// be carried down from the domain the user was listed under.
type mqlNewrelicUserInternal struct {
	cachedDomainID string
	cachedGroupIDs []string
}

func newUserResource(runtime *plugin.Runtime, user apiUser) (*mqlNewrelicUser, error) {
	res, err := CreateResource(runtime, "newrelic.user", map[string]*llx.RawData{
		"__id":                   llx.StringData("user/" + user.ID),
		"id":                     llx.StringData(user.ID),
		"name":                   llx.StringData(user.Name),
		"email":                  llx.StringData(user.Email),
		"type":                   llx.StringData(user.Type.DisplayName),
		"emailVerificationState": llx.StringData(user.EmailVerificationState),
		"timeZone":               llx.StringData(user.TimeZone),
		"lastActive":             llx.TimeDataPtr(user.LastActive.Time()),
		"pendingUpgrade":         llx.BoolData(hasPendingUpgrade(user)),
		"requestedType":          llx.StringData(requestedUserType(user)),
	})
	if err != nil {
		return nil, err
	}

	mqlUser := res.(*mqlNewrelicUser)
	mqlUser.cachedDomainID = user.domainID
	groupIDs := make([]string, 0, len(user.Groups.Groups))
	for _, group := range user.Groups.Groups {
		groupIDs = append(groupIDs, group.ID)
	}
	mqlUser.cachedGroupIDs = groupIDs

	return mqlUser, nil
}

func (r *mqlNewrelicUser) authenticationDomain() (*mqlNewrelicAuthenticationDomain, error) {
	domain, found, err := resolveAuthDomain(r.MqlRuntime, r.cachedDomainID)
	if err != nil {
		return nil, err
	}
	if !found {
		r.AuthenticationDomain.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return domain, nil
}

func (r *mqlNewrelicUser) groups() ([]any, error) {
	groups, err := cachedGroups(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	wanted := make(map[string]struct{}, len(r.cachedGroupIDs))
	for _, id := range r.cachedGroupIDs {
		wanted[id] = struct{}{}
	}

	res := []any{}
	for _, group := range groups {
		if _, ok := wanted[group.ID]; !ok {
			continue
		}
		mqlGroup, err := newGroupResource(r.MqlRuntime, group)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlGroup)
	}
	return res, nil
}

// resolveUser hands back the user resource for an ID, or null when the user is
// not in the organization's user list.
func resolveUser(runtime *plugin.Runtime, userID string) (*mqlNewrelicUser, bool, error) {
	if userID == "" || userID == "0" {
		return nil, false, nil
	}

	users, err := cachedUsers(runtime)
	if err != nil {
		return nil, false, err
	}
	for _, user := range users {
		if user.ID == userID {
			mqlUser, err := newUserResource(runtime, user)
			if err != nil {
				return nil, false, err
			}
			return mqlUser, true, nil
		}
	}
	return nil, false, nil
}

// -----------------------------------------------------------------------------
// group
// -----------------------------------------------------------------------------

// mqlNewrelicGroupInternal keeps the authentication domain the group belongs
// to, which the group record does not repeat.
type mqlNewrelicGroupInternal struct {
	cachedDomainID string
}

func newGroupResource(runtime *plugin.Runtime, group apiGroup) (*mqlNewrelicGroup, error) {
	res, err := CreateResource(runtime, "newrelic.group", map[string]*llx.RawData{
		"__id":        llx.StringData("group/" + group.ID),
		"id":          llx.StringData(group.ID),
		"displayName": llx.StringData(group.DisplayName),
	})
	if err != nil {
		return nil, err
	}

	mqlGroup := res.(*mqlNewrelicGroup)
	mqlGroup.cachedDomainID = group.domainID
	return mqlGroup, nil
}

func (r *mqlNewrelicGroup) authenticationDomain() (*mqlNewrelicAuthenticationDomain, error) {
	domain, found, err := resolveAuthDomain(r.MqlRuntime, r.cachedDomainID)
	if err != nil {
		return nil, err
	}
	if !found {
		r.AuthenticationDomain.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return domain, nil
}

// users lists the members of the group by reading the membership off the users
// themselves. New Relic reports the same edge from both ends, and reading it
// from the user list costs nothing extra because that list is already fetched,
// where asking for each group's members would be one call per group.
func (r *mqlNewrelicGroup) users() ([]any, error) {
	users, err := cachedUsers(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, user := range users {
		for _, group := range user.Groups.Groups {
			if group.ID != r.Id.Data {
				continue
			}
			mqlUser, err := newUserResource(r.MqlRuntime, user)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlUser)
			break
		}
	}
	return res, nil
}

func (r *mqlNewrelicGroup) accessGrants() ([]any, error) {
	groups, err := cachedGroups(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, group := range groups {
		if group.ID != r.Id.Data {
			continue
		}
		for _, grant := range group.Roles.Roles {
			mqlGrant, err := newAccessGrantResource(r.MqlRuntime, group, grant)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlGrant)
		}
	}
	return res, nil
}

// resolveGroup hands back the group resource for an ID, or null when the group
// is not in the organization's group list.
func resolveGroup(runtime *plugin.Runtime, groupID string) (*mqlNewrelicGroup, bool, error) {
	if groupID == "" {
		return nil, false, nil
	}

	groups, err := cachedGroups(runtime)
	if err != nil {
		return nil, false, err
	}
	for _, group := range groups {
		if group.ID == groupID {
			mqlGroup, err := newGroupResource(runtime, group)
			if err != nil {
				return nil, false, err
			}
			return mqlGroup, true, nil
		}
	}
	return nil, false, nil
}

// -----------------------------------------------------------------------------
// role
// -----------------------------------------------------------------------------

func newRoleResource(runtime *plugin.Runtime, role apiRole) (*mqlNewrelicRole, error) {
	res, err := CreateResource(runtime, "newrelic.role", map[string]*llx.RawData{
		"__id":        llx.StringData("role/" + role.ID),
		"id":          llx.StringData(role.ID),
		"name":        llx.StringData(role.Name),
		"displayName": llx.StringData(role.DisplayName),
		"type":        llx.StringData(role.Type),
		"scope":       llx.StringData(role.Scope),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlNewrelicRole), nil
}

// resolveRole hands back the role resource for an ID, or null when the role is
// not in the organization's role list.
func resolveRole(runtime *plugin.Runtime, roleID string) (*mqlNewrelicRole, bool, error) {
	if roleID == "" || roleID == "0" {
		return nil, false, nil
	}

	roles, err := cachedRoles(runtime)
	if err != nil {
		return nil, false, err
	}
	for _, role := range roles {
		if role.ID == roleID {
			mqlRole, err := newRoleResource(runtime, role)
			if err != nil {
				return nil, false, err
			}
			return mqlRole, true, nil
		}
	}
	return nil, false, nil
}

// -----------------------------------------------------------------------------
// access grant
// -----------------------------------------------------------------------------

// mqlNewrelicAccessGrantInternal keeps the identifiers the grant's references
// resolve against.
type mqlNewrelicAccessGrantInternal struct {
	cachedRoleID    string
	cachedGroupID   string
	cachedAccountID int
}

// accessGrantID builds the cache key of an access grant. A grant repeats along
// three dimensions, the group holding it, the role it confers and the account
// or organization it applies to, so all three go into the key. Keying on the
// API's own grant ID alone would risk one grant's data overwriting another's,
// which reports fewer grants than exist and attributes the survivor's scope to
// all of them.
func accessGrantID(group apiGroup, grant apiGrantedRole) string {
	scope := "account/" + strconv.Itoa(grant.AccountID)
	if isOrganizationWideGrant(grant) {
		scope = "organization/" + grant.OrganizationId
	}
	return "accessGrant/" + group.ID + "/" + scope + "/" + strconv.Itoa(grant.RoleId) + "/" + grant.ID
}

func newAccessGrantResource(runtime *plugin.Runtime, group apiGroup, grant apiGrantedRole) (*mqlNewrelicAccessGrant, error) {
	// The grant carries its own group ID, but New Relic leaves it null on an
	// organization-wide grant. The group it was listed under is authoritative
	// either way.
	groupID := group.ID
	if groupID == "" {
		groupID = grant.GroupId
	}

	res, err := CreateResource(runtime, "newrelic.accessGrant", map[string]*llx.RawData{
		"__id":             llx.StringData(accessGrantID(group, grant)),
		"id":               llx.StringData(grant.ID),
		"roleName":         llx.StringData(grant.Name),
		"roleDisplayName":  llx.StringData(grant.DisplayName),
		"roleType":         llx.StringData(grant.Type),
		"groupId":          llx.StringData(groupID),
		"accountId":        llx.IntData(int64(grant.AccountID)),
		"organizationWide": llx.BoolData(isOrganizationWideGrant(grant)),
	})
	if err != nil {
		return nil, err
	}

	mqlGrant := res.(*mqlNewrelicAccessGrant)
	mqlGrant.cachedRoleID = strconv.Itoa(grant.RoleId)
	mqlGrant.cachedGroupID = groupID
	mqlGrant.cachedAccountID = grant.AccountID
	return mqlGrant, nil
}

func (r *mqlNewrelicAccessGrant) roleRef() (*mqlNewrelicRole, error) {
	role, found, err := resolveRole(r.MqlRuntime, r.cachedRoleID)
	if err != nil {
		return nil, err
	}
	if !found {
		r.RoleRef.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return role, nil
}

func (r *mqlNewrelicAccessGrant) groupRef() (*mqlNewrelicGroup, error) {
	group, found, err := resolveGroup(r.MqlRuntime, r.cachedGroupID)
	if err != nil {
		return nil, err
	}
	if !found {
		r.GroupRef.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return group, nil
}

func (r *mqlNewrelicAccessGrant) account() (*mqlNewrelicAccount, error) {
	account, found, err := resolveAccount(r.MqlRuntime, r.cachedAccountID)
	if err != nil {
		return nil, err
	}
	if !found {
		r.Account.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return account, nil
}
