// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	madmin "github.com/minio/madmin-go/v4"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/minio/connection"
)

// rootResource returns the deployment singleton, which owns the cached identity
// listings. References between users, groups, policies and service accounts are
// resolved by walking those listings rather than by looking each target up on
// its own, so a deployment with many users costs one listing instead of one
// request per reference.
func rootResource(runtime *plugin.Runtime) (*mqlMinio, error) {
	resource, err := CreateResource(runtime, "minio", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMinio), nil
}

// splitPolicyNames splits the comma-separated policy attachment MinIO reports
// on a user or group. An unattached identity reports an empty string, which
// must not become a single empty name.
func splitPolicyNames(value string) []string {
	out := []string{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

// ---------------------------------------------------------------- users

// mqlMinioUserInternal carries the raw attachment names the listing reported,
// so the typed references can be resolved without asking the deployment again.
type mqlMinioUserInternal struct {
	cachedPolicyNames []string
	cachedGroupNames  []string
}

func (a *mqlMinioUser) id() (string, error) {
	if a.Name.Data == "" {
		return "", errors.New("minio.user requires a name")
	}
	return "user/" + a.Name.Data, nil
}

func (a *mqlMinio) users() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.MinioConnection)
	users, err := conn.Admin().ListUsers(context.Background())
	if err != nil {
		return nil, err
	}

	// The listing is a map, so it arrives in an arbitrary order. Sorting keeps
	// the result stable across runs, which matters for anything diffing scans.
	names := make([]string, 0, len(users))
	for name := range users {
		names = append(names, name)
	}
	sort.Strings(names)

	res := make([]any, 0, len(names))
	for _, name := range names {
		resource, err := newUserResource(a.MqlRuntime, name, users[name])
		if err != nil {
			return nil, err
		}
		res = append(res, resource)
	}
	return res, nil
}

// userSchemaArgs maps a user listing entry onto the schema.
//
// UserInfo carries a SecretKey field, which the admin API populates on some
// endpoints. It is the credential itself and is deliberately never read here:
// nothing in this schema needs it, and a provider that copies it into a result
// hands it to every consumer of the scan.
func userSchemaArgs(name string, info madmin.UserInfo) map[string]*llx.RawData {
	status := string(info.Status)
	return map[string]*llx.RawData{
		"__id":      llx.StringData("user/" + name),
		"name":      llx.StringData(name),
		"status":    llx.StringData(status),
		"enabled":   llx.BoolData(strings.EqualFold(status, string(madmin.AccountEnabled))),
		"updatedAt": timeData(info.UpdatedAt),
	}
}

func newUserResource(runtime *plugin.Runtime, name string, info madmin.UserInfo) (plugin.Resource, error) {
	resource, err := CreateResource(runtime, "minio.user", userSchemaArgs(name, info))
	if err != nil {
		return nil, err
	}
	mqlUser := resource.(*mqlMinioUser)
	mqlUser.cachedPolicyNames = splitPolicyNames(info.PolicyName)
	mqlUser.cachedGroupNames = info.MemberOf
	return resource, nil
}

func initMinioUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	nameArg, ok := args["name"]
	if !ok {
		return args, nil, nil
	}
	name, ok := nameArg.Value.(string)
	if !ok || name == "" {
		return nil, nil, errors.New("minio.user requires a name")
	}

	conn := runtime.Connection.(*connection.MinioConnection)
	users, err := conn.Admin().ListUsers(context.Background())
	if err != nil {
		return nil, nil, err
	}
	info, ok := users[name]
	if !ok {
		return nil, nil, fmt.Errorf("minio.user with name %q not found", name)
	}
	resource, err := newUserResource(runtime, name, info)
	if err != nil {
		return nil, nil, err
	}
	return nil, resource, nil
}

func (a *mqlMinioUser) policies() ([]any, error) {
	root, err := rootResource(a.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return policiesByName(root, a.cachedPolicyNames)
}

func (a *mqlMinioUser) groups() ([]any, error) {
	root, err := rootResource(a.MqlRuntime)
	if err != nil {
		return nil, err
	}
	groups := root.GetGroups()
	if groups.Error != nil {
		return nil, groups.Error
	}

	wanted := map[string]struct{}{}
	for _, name := range a.cachedGroupNames {
		wanted[name] = struct{}{}
	}

	res := []any{}
	for _, entry := range groups.Data {
		group, ok := entry.(*mqlMinioGroup)
		if !ok {
			continue
		}
		if _, ok := wanted[group.Name.Data]; ok {
			res = append(res, group)
		}
	}
	return res, nil
}

func (a *mqlMinioUser) serviceAccounts() ([]any, error) {
	root, err := rootResource(a.MqlRuntime)
	if err != nil {
		return nil, err
	}
	accounts := root.GetServiceAccounts()
	if accounts.Error != nil {
		return nil, accounts.Error
	}

	res := []any{}
	for _, entry := range accounts.Data {
		account, ok := entry.(*mqlMinioServiceAccount)
		if !ok {
			continue
		}
		if account.cachedParentUser == a.Name.Data {
			res = append(res, account)
		}
	}
	return res, nil
}

// ---------------------------------------------------------------- groups

type mqlMinioGroupInternal struct {
	cachedPolicyNames []string
	cachedMembers     []string
}

func (a *mqlMinioGroup) id() (string, error) {
	if a.Name.Data == "" {
		return "", errors.New("minio.group requires a name")
	}
	return "group/" + a.Name.Data, nil
}

func (a *mqlMinio) groups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.MinioConnection)
	ctx := context.Background()
	names, err := conn.Admin().ListGroups(ctx)
	if err != nil {
		return nil, err
	}
	sort.Strings(names)

	res := make([]any, 0, len(names))
	for _, name := range names {
		desc, err := conn.Admin().GetGroupDescription(ctx, name)
		if err != nil {
			// A group that vanished between the listing and the description is
			// skipped rather than taking the whole listing down with it, since
			// every other group is still a real answer.
			if isAdminNotFound(err) {
				log.Debug().Str("group", name).Msg("minio group disappeared while being described")
				continue
			}
			return nil, err
		}
		if desc == nil {
			continue
		}
		resource, err := newGroupResource(a.MqlRuntime, *desc)
		if err != nil {
			return nil, err
		}
		res = append(res, resource)
	}
	return res, nil
}

// groupSchemaArgs maps a group description onto the schema.
func groupSchemaArgs(desc madmin.GroupDesc) map[string]*llx.RawData {
	return map[string]*llx.RawData{
		"__id":      llx.StringData("group/" + desc.Name),
		"name":      llx.StringData(desc.Name),
		"status":    llx.StringData(desc.Status),
		"enabled":   llx.BoolData(strings.EqualFold(desc.Status, string(madmin.AccountEnabled))),
		"updatedAt": timeData(desc.UpdatedAt),
	}
}

func newGroupResource(runtime *plugin.Runtime, desc madmin.GroupDesc) (plugin.Resource, error) {
	resource, err := CreateResource(runtime, "minio.group", groupSchemaArgs(desc))
	if err != nil {
		return nil, err
	}
	mqlGroup := resource.(*mqlMinioGroup)
	mqlGroup.cachedPolicyNames = splitPolicyNames(desc.Policy)
	mqlGroup.cachedMembers = desc.Members
	return resource, nil
}

func initMinioGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	nameArg, ok := args["name"]
	if !ok {
		return args, nil, nil
	}
	name, ok := nameArg.Value.(string)
	if !ok || name == "" {
		return nil, nil, errors.New("minio.group requires a name")
	}

	conn := runtime.Connection.(*connection.MinioConnection)
	desc, err := conn.Admin().GetGroupDescription(context.Background(), name)
	if err != nil {
		if isAdminNotFound(err) {
			return nil, nil, fmt.Errorf("minio.group with name %q not found", name)
		}
		return nil, nil, err
	}
	if desc == nil {
		return nil, nil, fmt.Errorf("minio.group with name %q not found", name)
	}
	resource, err := newGroupResource(runtime, *desc)
	if err != nil {
		return nil, nil, err
	}
	return nil, resource, nil
}

func (a *mqlMinioGroup) policies() ([]any, error) {
	root, err := rootResource(a.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return policiesByName(root, a.cachedPolicyNames)
}

func (a *mqlMinioGroup) members() ([]any, error) {
	root, err := rootResource(a.MqlRuntime)
	if err != nil {
		return nil, err
	}
	users := root.GetUsers()
	if users.Error != nil {
		return nil, users.Error
	}

	wanted := map[string]struct{}{}
	for _, name := range a.cachedMembers {
		wanted[name] = struct{}{}
	}

	res := []any{}
	for _, entry := range users.Data {
		user, ok := entry.(*mqlMinioUser)
		if !ok {
			continue
		}
		if _, ok := wanted[user.Name.Data]; ok {
			res = append(res, user)
		}
	}
	return res, nil
}

// ------------------------------------------------------- service accounts

type mqlMinioServiceAccountInternal struct {
	cachedParentUser string

	infoLock    sync.Mutex
	infoFetched bool
	info        madmin.InfoServiceAccountResp

	policyLock   sync.Mutex
	policyParsed bool
	policy       *iamPolicy
	policyErr    error
}

func (a *mqlMinioServiceAccount) id() (string, error) {
	if a.AccessKey.Data == "" {
		return "", errors.New("minio.serviceAccount requires an access key")
	}
	return "serviceAccount/" + a.AccessKey.Data, nil
}

func (a *mqlMinio) serviceAccounts() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.MinioConnection)
	ctx := context.Background()

	users, err := conn.Admin().ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	// The deployment's own account owns service accounts too, and it is not a
	// member of the user listing, so it is queried alongside the users.
	owners := make([]string, 0, len(users)+1)
	for name := range users {
		owners = append(owners, name)
	}
	sort.Strings(owners)
	owners = append([]string{""}, owners...)

	seen := map[string]struct{}{}
	res := []any{}
	for _, owner := range owners {
		listed, err := conn.Admin().ListServiceAccounts(ctx, owner)
		if err != nil {
			if isAdminNotFound(err) {
				continue
			}
			return nil, err
		}
		for _, account := range listed.Accounts {
			if _, ok := seen[account.AccessKey]; ok {
				continue
			}
			seen[account.AccessKey] = struct{}{}
			resource, err := newServiceAccountResource(a.MqlRuntime, account)
			if err != nil {
				return nil, err
			}
			res = append(res, resource)
		}
	}
	return res, nil
}

// serviceAccountSchemaArgs maps a service account onto the schema. The access
// key is the public half of the pair and is what identifies the account; the
// secret half is never read, and an account with no expiry reports null rather
// than a date, so "never expires" cannot be mistaken for a date in the past.
func serviceAccountSchemaArgs(account madmin.ServiceAccountInfo) map[string]*llx.RawData {
	return map[string]*llx.RawData{
		"__id":          llx.StringData("serviceAccount/" + account.AccessKey),
		"accessKey":     llx.StringData(account.AccessKey),
		"name":          llx.StringData(account.Name),
		"description":   llx.StringData(account.Description),
		"status":        llx.StringData(account.AccountStatus),
		"impliedPolicy": llx.BoolData(account.ImpliedPolicy),
		"expiresAt":     timePtrData(account.Expiration),
	}
}

func newServiceAccountResource(runtime *plugin.Runtime, account madmin.ServiceAccountInfo) (plugin.Resource, error) {
	resource, err := CreateResource(runtime, "minio.serviceAccount", serviceAccountSchemaArgs(account))
	if err != nil {
		return nil, err
	}
	resource.(*mqlMinioServiceAccount).cachedParentUser = account.ParentUser
	return resource, nil
}

func initMinioServiceAccount(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	keyArg, ok := args["accessKey"]
	if !ok {
		return args, nil, nil
	}
	accessKey, ok := keyArg.Value.(string)
	if !ok || accessKey == "" {
		return nil, nil, errors.New("minio.serviceAccount requires an access key")
	}

	conn := runtime.Connection.(*connection.MinioConnection)
	info, err := conn.Admin().InfoServiceAccount(context.Background(), accessKey)
	if err != nil {
		if isAdminNotFound(err) {
			return nil, nil, fmt.Errorf("minio.serviceAccount with access key %q not found", accessKey)
		}
		return nil, nil, err
	}

	resource, err := newServiceAccountResource(runtime, madmin.ServiceAccountInfo{
		ParentUser:    info.ParentUser,
		AccountStatus: info.AccountStatus,
		ImpliedPolicy: info.ImpliedPolicy,
		AccessKey:     accessKey,
		Name:          info.Name,
		Description:   info.Description,
		Expiration:    info.Expiration,
	})
	if err != nil {
		return nil, nil, err
	}
	return nil, resource, nil
}

// accountInfo reads the account's detail once. The listing does not carry the
// policy in force, so it takes a second request that only the policy fields
// pay for.
func (a *mqlMinioServiceAccount) accountInfo() (madmin.InfoServiceAccountResp, error) {
	a.infoLock.Lock()
	defer a.infoLock.Unlock()
	if a.infoFetched {
		return a.info, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.MinioConnection)
	info, err := conn.Admin().InfoServiceAccount(context.Background(), a.AccessKey.Data)
	if err != nil {
		return madmin.InfoServiceAccountResp{}, err
	}
	a.infoFetched = true
	a.info = info
	return a.info, nil
}

func (a *mqlMinioServiceAccount) policyDocument() (string, error) {
	info, err := a.accountInfo()
	if err != nil {
		return "", err
	}
	return info.Policy, nil
}

func (a *mqlMinioServiceAccount) accountPolicy() (*iamPolicy, error) {
	a.policyLock.Lock()
	defer a.policyLock.Unlock()
	if a.policyParsed {
		return a.policy, a.policyErr
	}

	info, err := a.accountInfo()
	if err != nil {
		return nil, err
	}
	policy, err := parsePolicyDocument(info.Policy)
	a.policyParsed = true
	a.policy = policy
	a.policyErr = err
	return a.policy, a.policyErr
}

func (a *mqlMinioServiceAccount) policyStatements() ([]any, error) {
	policy, err := a.accountPolicy()
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return []any{}, nil
	}
	return createPolicyStatements(a.MqlRuntime, "serviceAccount/"+a.AccessKey.Data, policy)
}

// parentUser resolves the user the account was issued against. An account
// issued through LDAP or OpenID has a parent that is not in the built-in
// identity store, which resolves to null rather than to an error.
func (a *mqlMinioServiceAccount) parentUser() (*mqlMinioUser, error) {
	markNull := func() (*mqlMinioUser, error) {
		a.ParentUser.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	if a.cachedParentUser == "" {
		return markNull()
	}

	root, err := rootResource(a.MqlRuntime)
	if err != nil {
		return nil, err
	}
	users := root.GetUsers()
	if users.Error != nil {
		return nil, users.Error
	}
	for _, entry := range users.Data {
		user, ok := entry.(*mqlMinioUser)
		if !ok {
			continue
		}
		if user.Name.Data == a.cachedParentUser {
			return user, nil
		}
	}
	return markNull()
}

// ------------------------------------------------------------- policies

type mqlMinioPolicyInternal struct {
	policyLock   sync.Mutex
	policyParsed bool
	policy       *iamPolicy
	policyErr    error

	datesLock    sync.Mutex
	datesFetched bool
	createDate   time.Time
	updateDate   time.Time
}

func (a *mqlMinioPolicy) id() (string, error) {
	if a.Name.Data == "" {
		return "", errors.New("minio.policy requires a name")
	}
	return "policy/" + a.Name.Data, nil
}

func (a *mqlMinio) policies() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.MinioConnection)
	listed, err := conn.Admin().ListCannedPolicies(context.Background())
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(listed))
	for name := range listed {
		names = append(names, name)
	}
	sort.Strings(names)

	res := make([]any, 0, len(names))
	for _, name := range names {
		resource, err := CreateResource(a.MqlRuntime, "minio.policy", map[string]*llx.RawData{
			"__id":     llx.StringData("policy/" + name),
			"name":     llx.StringData(name),
			"document": llx.StringData(string(listed[name])),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, resource)
	}
	return res, nil
}

func initMinioPolicy(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	nameArg, ok := args["name"]
	if !ok {
		return args, nil, nil
	}
	name, ok := nameArg.Value.(string)
	if !ok || name == "" {
		return nil, nil, errors.New("minio.policy requires a name")
	}

	conn := runtime.Connection.(*connection.MinioConnection)
	info, err := conn.Admin().InfoCannedPolicy(context.Background(), name)
	if err != nil {
		if isAdminNotFound(err) {
			return nil, nil, fmt.Errorf("minio.policy with name %q not found", name)
		}
		return nil, nil, err
	}
	if info == nil {
		return nil, nil, fmt.Errorf("minio.policy with name %q not found", name)
	}

	args["name"] = llx.StringData(name)
	args["document"] = llx.StringData(string(info.Policy))
	return args, nil, nil
}

// policiesByName resolves attachment names against the deployment's policy
// listing. A name with no matching policy is skipped and logged rather than
// failing the read, because an attachment can outlive the policy it names.
func policiesByName(root *mqlMinio, names []string) ([]any, error) {
	if len(names) == 0 {
		return []any{}, nil
	}
	policies := root.GetPolicies()
	if policies.Error != nil {
		return nil, policies.Error
	}

	byName := map[string]*mqlMinioPolicy{}
	for _, entry := range policies.Data {
		policy, ok := entry.(*mqlMinioPolicy)
		if !ok {
			continue
		}
		byName[policy.Name.Data] = policy
	}

	res := []any{}
	for _, name := range names {
		policy, ok := byName[name]
		if !ok {
			log.Debug().Str("policy", name).Msg("minio policy attachment names a policy that no longer exists")
			continue
		}
		res = append(res, policy)
	}
	return res, nil
}

func (a *mqlMinioPolicy) parsed() (*iamPolicy, error) {
	a.policyLock.Lock()
	defer a.policyLock.Unlock()
	if a.policyParsed {
		return a.policy, a.policyErr
	}
	policy, err := parsePolicyDocument(a.Document.Data)
	a.policyParsed = true
	a.policy = policy
	a.policyErr = err
	return a.policy, a.policyErr
}

func (a *mqlMinioPolicy) statements() ([]any, error) {
	policy, err := a.parsed()
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return []any{}, nil
	}
	return createPolicyStatements(a.MqlRuntime, "policy/"+a.Name.Data, policy)
}

func (a *mqlMinioPolicy) hasWildcardAction() (bool, error) {
	policy, err := a.parsed()
	if err != nil {
		return false, err
	}
	return policyHasWildcardAction(policy), nil
}

func (a *mqlMinioPolicy) hasWildcardResource() (bool, error) {
	policy, err := a.parsed()
	if err != nil {
		return false, err
	}
	return policyHasWildcardResource(policy), nil
}

func (a *mqlMinioPolicy) grantsAdminAccess() (bool, error) {
	policy, err := a.parsed()
	if err != nil {
		return false, err
	}
	return policyGrantsAdminAccess(policy), nil
}

// policyDates reads the policy's timestamps once. The listing does not carry
// them, so they cost a request per policy that only the date fields pay for.
func (a *mqlMinioPolicy) policyDates() (time.Time, time.Time, error) {
	a.datesLock.Lock()
	defer a.datesLock.Unlock()
	if a.datesFetched {
		return a.createDate, a.updateDate, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.MinioConnection)
	info, err := conn.Admin().InfoCannedPolicy(context.Background(), a.Name.Data)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	a.datesFetched = true
	if info != nil {
		a.createDate = info.CreateDate
		a.updateDate = info.UpdateDate
	}
	return a.createDate, a.updateDate, nil
}

func (a *mqlMinioPolicy) createdAt() (*time.Time, error) {
	created, _, err := a.policyDates()
	if err != nil {
		return nil, err
	}
	if created.IsZero() {
		a.CreatedAt.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return &created, nil
}

func (a *mqlMinioPolicy) updatedAt() (*time.Time, error) {
	_, updated, err := a.policyDates()
	if err != nil {
		return nil, err
	}
	if updated.IsZero() {
		a.UpdatedAt.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return &updated, nil
}

func (a *mqlMinioPolicy) users() ([]any, error) {
	root, err := rootResource(a.MqlRuntime)
	if err != nil {
		return nil, err
	}
	users := root.GetUsers()
	if users.Error != nil {
		return nil, users.Error
	}

	res := []any{}
	for _, entry := range users.Data {
		user, ok := entry.(*mqlMinioUser)
		if !ok {
			continue
		}
		if containsValue(user.cachedPolicyNames, a.Name.Data) {
			res = append(res, user)
		}
	}
	return res, nil
}

func (a *mqlMinioPolicy) groups() ([]any, error) {
	root, err := rootResource(a.MqlRuntime)
	if err != nil {
		return nil, err
	}
	groups := root.GetGroups()
	if groups.Error != nil {
		return nil, groups.Error
	}

	res := []any{}
	for _, entry := range groups.Data {
		group, ok := entry.(*mqlMinioGroup)
		if !ok {
			continue
		}
		if containsValue(group.cachedPolicyNames, a.Name.Data) {
			res = append(res, group)
		}
	}
	return res, nil
}

// ------------------------------------------------------ policy statements

// createPolicyStatements renders a parsed policy's statements. The cache key
// carries both the owner and the statement's position, because a statement ID
// is optional and two statements in one policy may share one.
func createPolicyStatements(runtime *plugin.Runtime, ownerID string, policy *iamPolicy) ([]any, error) {
	res := make([]any, 0, len(policy.Statements))
	for i, statement := range policy.Statements {
		args := map[string]*llx.RawData{
			"__id":         llx.StringData(fmt.Sprintf("%s/statement/%d", ownerID, i)),
			"sid":          llx.StringData(statement.SID),
			"effect":       llx.StringData(statement.Effect),
			"actions":      strSliceData(statement.Action),
			"notActions":   strSliceData(statement.NotAction),
			"resources":    strSliceData(statement.Resource),
			"notResources": strSliceData(statement.NotResource),
			"principals":   strSliceData(statement.Principal.Values),
			"conditions":   llx.DictData(conditionsToDict(statement.Condition)),
		}
		resource, err := CreateResource(runtime, "minio.policyStatement", args)
		if err != nil {
			return nil, err
		}
		res = append(res, resource)
	}
	return res, nil
}
