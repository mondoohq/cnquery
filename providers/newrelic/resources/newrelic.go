// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/newrelic/connection"
)

func (r *mqlNewrelic) id() (string, error) {
	conn, err := newrelicConn(r.MqlRuntime)
	if err != nil {
		return "", err
	}
	return connection.NewAccountIdentifier(conn.Region(), conn.AccountID()), nil
}

// newrelicConn pulls the connection off the runtime.
func newrelicConn(runtime *plugin.Runtime) (*connection.NewrelicConnection, error) {
	conn, ok := runtime.Connection.(*connection.NewrelicConnection)
	if !ok {
		return nil, errors.New("no New Relic connection on the runtime")
	}
	return conn, nil
}

// memoized runs fetch at most once per connection for the given key. Every list
// on this provider goes through it, which is what keeps a typed reference from
// costing an API call per row: the reference resolves against the list that has
// already been read rather than looking its target up on its own.
func memoized[T any](runtime *plugin.Runtime, key string, fetch func(ctx context.Context, conn *connection.NewrelicConnection) (T, error)) (T, error) {
	var zero T

	conn, err := newrelicConn(runtime)
	if err != nil {
		return zero, err
	}

	value, err := conn.Memoize(key, func() (any, error) {
		return fetch(context.Background(), conn)
	})
	if err != nil {
		return zero, err
	}

	typed, ok := value.(T)
	if !ok {
		return zero, fmt.Errorf("the cached New Relic %s has an unexpected type %T", key, value)
	}
	return typed, nil
}

// -----------------------------------------------------------------------------
// memoized collections
// -----------------------------------------------------------------------------

func cachedAccounts(runtime *plugin.Runtime) ([]apiAccount, error) {
	return memoized(runtime, "accounts", func(ctx context.Context, conn *connection.NewrelicConnection) ([]apiAccount, error) {
		return fetchAccounts(ctx, conn.Client())
	})
}

func cachedOrganization(runtime *plugin.Runtime) (*apiOrganization, error) {
	return memoized(runtime, "organization", func(ctx context.Context, conn *connection.NewrelicConnection) (*apiOrganization, error) {
		return fetchOrganization(ctx, conn.Client())
	})
}

func cachedAuthDomains(runtime *plugin.Runtime) ([]apiAuthDomain, error) {
	return memoized(runtime, "authDomains", func(ctx context.Context, conn *connection.NewrelicConnection) ([]apiAuthDomain, error) {
		return fetchAuthDomainsWithUsers(ctx, conn.Client())
	})
}

// cachedAuthTypes maps an authentication domain ID onto the login method the
// domain accepts. It is read separately from the domains themselves because the
// login method is only exposed through the customer administration view, which
// needs a broader permission than the rest of this provider.
func cachedAuthTypes(runtime *plugin.Runtime) (map[string]apiAdminAuthDomain, error) {
	return memoized(runtime, "authTypes", func(ctx context.Context, conn *connection.NewrelicConnection) (map[string]apiAdminAuthDomain, error) {
		items, err := fetchAdminAuthDomains(ctx, conn.Client())
		if err != nil {
			return nil, err
		}
		byID := make(map[string]apiAdminAuthDomain, len(items))
		for _, item := range items {
			byID[item.ID] = item
		}
		return byID, nil
	})
}

func cachedGroups(runtime *plugin.Runtime) ([]apiGroup, error) {
	return memoized(runtime, "groups", func(ctx context.Context, conn *connection.NewrelicConnection) ([]apiGroup, error) {
		return fetchGroupsWithGrants(ctx, conn.Client())
	})
}

func cachedRoles(runtime *plugin.Runtime) ([]apiRole, error) {
	return memoized(runtime, "roles", func(ctx context.Context, conn *connection.NewrelicConnection) ([]apiRole, error) {
		return fetchRoles(ctx, conn.Client())
	})
}

func cachedAPIKeys(runtime *plugin.Runtime) ([]apiKey, error) {
	return memoized(runtime, "apiKeys", func(ctx context.Context, conn *connection.NewrelicConnection) ([]apiKey, error) {
		return fetchAPIKeys(ctx, conn.Client(), conn.AccountID())
	})
}

func cachedAlertPolicies(runtime *plugin.Runtime) ([]apiAlertPolicy, error) {
	return memoized(runtime, "alertPolicies", func(ctx context.Context, conn *connection.NewrelicConnection) ([]apiAlertPolicy, error) {
		return fetchAlertPolicies(ctx, conn.Client(), conn.AccountID())
	})
}

func cachedAlertConditions(runtime *plugin.Runtime) ([]apiAlertCondition, error) {
	return memoized(runtime, "alertConditions", func(ctx context.Context, conn *connection.NewrelicConnection) ([]apiAlertCondition, error) {
		return fetchAlertConditions(ctx, conn.Client(), conn.AccountID())
	})
}

func cachedNotificationDestinations(runtime *plugin.Runtime) ([]apiNotificationDestination, error) {
	return memoized(runtime, "notificationDestinations", func(ctx context.Context, conn *connection.NewrelicConnection) ([]apiNotificationDestination, error) {
		return fetchNotificationDestinations(ctx, conn.Client(), conn.AccountID())
	})
}

func cachedNotificationChannels(runtime *plugin.Runtime) ([]apiNotificationChannel, error) {
	return memoized(runtime, "notificationChannels", func(ctx context.Context, conn *connection.NewrelicConnection) ([]apiNotificationChannel, error) {
		return fetchNotificationChannels(ctx, conn.Client(), conn.AccountID())
	})
}

func cachedDropRules(runtime *plugin.Runtime) ([]apiDropRule, error) {
	return memoized(runtime, "dropRules", func(ctx context.Context, conn *connection.NewrelicConnection) ([]apiDropRule, error) {
		return fetchDropRules(ctx, conn.Client(), conn.AccountID())
	})
}

func cachedRetentionRules(runtime *plugin.Runtime) ([]apiRetentionRule, error) {
	return memoized(runtime, "retentionRules", func(ctx context.Context, conn *connection.NewrelicConnection) ([]apiRetentionRule, error) {
		return fetchRetentionRules(ctx, conn.Client(), conn.AccountID())
	})
}

// cachedUsers flattens the users of every authentication domain into one list.
func cachedUsers(runtime *plugin.Runtime) ([]apiUser, error) {
	domains, err := cachedAuthDomains(runtime)
	if err != nil {
		return nil, err
	}
	var users []apiUser
	for _, domain := range domains {
		users = append(users, domain.Users.Users...)
	}
	return users, nil
}

// -----------------------------------------------------------------------------
// root resource fields
// -----------------------------------------------------------------------------

func (r *mqlNewrelic) currentAccount() (*mqlNewrelicAccount, error) {
	conn, err := newrelicConn(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	accounts, err := cachedAccounts(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	for _, account := range accounts {
		if account.ID == conn.AccountID() {
			return newAccountResource(r.MqlRuntime, account)
		}
	}

	// The connection names an account the supplied key cannot see. That is a
	// misconfiguration worth reporting: every account-scoped list below would
	// otherwise come back empty and read as a clean account.
	return nil, fmt.Errorf("the supplied New Relic key cannot read account %d", conn.AccountID())
}

func (r *mqlNewrelic) accounts() ([]any, error) {
	accounts, err := cachedAccounts(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(accounts))
	for _, account := range accounts {
		mqlAccount, err := newAccountResource(r.MqlRuntime, account)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAccount)
	}
	return res, nil
}

func (r *mqlNewrelic) org() (*mqlNewrelicOrganization, error) {
	org, err := cachedOrganization(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(r.MqlRuntime, "newrelic.organization", map[string]*llx.RawData{
		"__id": llx.StringData("organization/" + org.ID),
		"id":   llx.StringData(org.ID),
		"name": llx.StringData(org.Name),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlNewrelicOrganization), nil
}

func (r *mqlNewrelic) authenticationDomains() ([]any, error) {
	domains, err := cachedAuthDomains(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(domains))
	for _, domain := range domains {
		mqlDomain, err := newAuthDomainResource(r.MqlRuntime, domain)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlDomain)
	}
	return res, nil
}

func (r *mqlNewrelic) users() ([]any, error) {
	users, err := cachedUsers(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(users))
	for _, user := range users {
		mqlUser, err := newUserResource(r.MqlRuntime, user)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlUser)
	}
	return res, nil
}

func (r *mqlNewrelic) groups() ([]any, error) {
	groups, err := cachedGroups(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(groups))
	for _, group := range groups {
		mqlGroup, err := newGroupResource(r.MqlRuntime, group)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlGroup)
	}
	return res, nil
}

func (r *mqlNewrelic) roles() ([]any, error) {
	roles, err := cachedRoles(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(roles))
	for _, role := range roles {
		mqlRole, err := newRoleResource(r.MqlRuntime, role)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRole)
	}
	return res, nil
}

func (r *mqlNewrelic) accessGrants() ([]any, error) {
	groups, err := cachedGroups(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	var res []any
	for _, group := range groups {
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

func (r *mqlNewrelic) apiKeys() ([]any, error) {
	keys, err := cachedAPIKeys(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(keys))
	for _, key := range keys {
		mqlKey, err := newAPIKeyResource(r.MqlRuntime, key)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlKey)
	}
	return res, nil
}

func (r *mqlNewrelic) alertPolicies() ([]any, error) {
	policies, err := cachedAlertPolicies(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(policies))
	for _, policy := range policies {
		mqlPolicy, err := newAlertPolicyResource(r.MqlRuntime, policy)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlPolicy)
	}
	return res, nil
}

func (r *mqlNewrelic) alertConditions() ([]any, error) {
	conditions, err := cachedAlertConditions(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(conditions))
	for _, condition := range conditions {
		mqlCondition, err := newAlertConditionResource(r.MqlRuntime, condition)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCondition)
	}
	return res, nil
}

func (r *mqlNewrelic) notificationDestinations() ([]any, error) {
	destinations, err := cachedNotificationDestinations(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(destinations))
	for _, destination := range destinations {
		mqlDestination, err := newNotificationDestinationResource(r.MqlRuntime, destination)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlDestination)
	}
	return res, nil
}

func (r *mqlNewrelic) notificationChannels() ([]any, error) {
	channels, err := cachedNotificationChannels(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(channels))
	for _, channel := range channels {
		mqlChannel, err := newNotificationChannelResource(r.MqlRuntime, channel)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlChannel)
	}
	return res, nil
}

func (r *mqlNewrelic) dropRules() ([]any, error) {
	rules, err := cachedDropRules(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(rules))
	for _, rule := range rules {
		mqlRule, err := newDropRuleResource(r.MqlRuntime, rule)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRule)
	}
	return res, nil
}

func (r *mqlNewrelic) dataRetentionRules() ([]any, error) {
	rules, err := cachedRetentionRules(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(rules))
	for _, rule := range rules {
		mqlRule, err := newRetentionRuleResource(r.MqlRuntime, rule)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRule)
	}
	return res, nil
}
