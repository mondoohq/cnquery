// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/bitwarden/connection"
)

func (r *mqlBitwarden) id() (string, error) {
	return "bitwarden", nil
}

// conn returns the Bitwarden connection backing this runtime.
func (r *mqlBitwarden) conn() *connection.BitwardenConnection {
	return r.MqlRuntime.Connection.(*connection.BitwardenConnection)
}

// initBitwardenOrganization reads the organization's account-level settings.
// Its cache key is the organization UUID extracted from the connection's
// client ID, since there is exactly one organization per connection, so the
// resource is queried directly without a parent field.
func initBitwardenOrganization(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.BitwardenConnection)

	sub, err := conn.Client().GetOrganizationSubscription(context.Background())
	if err != nil {
		return nil, nil, err
	}

	args["__id"] = llx.StringData(conn.OrgID())
	args["id"] = llx.StringData(conn.OrgID())
	args["seats"] = llx.IntDataPtr(sub.Seats)
	args["occupiedSeats"] = llx.IntDataPtr(sub.OccupiedSeats)
	args["maxCollections"] = llx.IntDataPtr(sub.MaxCollections)
	args["maxStorageGb"] = llx.IntDataPtr(sub.MaxStorageGb)
	args["enabled"] = llx.BoolData(sub.Enabled)
	args["useSso"] = llx.BoolData(sub.UseSso)
	args["use2fa"] = llx.BoolData(sub.UseTotp)
	args["useDirectory"] = llx.BoolData(sub.UseDirectory)
	args["useEvents"] = llx.BoolData(sub.UseEvents)
	args["useGroups"] = llx.BoolData(sub.UseGroups)
	args["usePolicies"] = llx.BoolData(sub.UsePolicies)
	args["useSend"] = llx.BoolData(sub.UseSend)
	args["useApi"] = llx.BoolData(sub.UseApi)
	args["useResetPassword"] = llx.BoolData(sub.UseResetPassword)
	args["planName"] = llx.StringData(sub.PlanName)
	args["businessName"] = llx.StringData(sub.BusinessName)
	// The Public API's subscription endpoint has no dedicated "name"
	// field; the organization's business name is its display name.
	args["name"] = llx.StringData(sub.BusinessName)

	return args, nil, nil
}

// policies lists every security policy configured for the organization.
func (r *mqlBitwarden) policies() ([]any, error) {
	conn := r.conn()

	policies, err := conn.Client().ListPolicies(context.Background())
	if err != nil {
		return nil, err
	}

	var all []any
	for _, p := range policies {
		res, err := newMqlBitwardenPolicy(r.MqlRuntime, p)
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}

// members lists every member of the organization.
func (r *mqlBitwarden) members() ([]any, error) {
	conn := r.conn()

	members, err := conn.ListMembersCached(context.Background())
	if err != nil {
		return nil, err
	}

	var all []any
	for _, m := range members {
		res, err := newMqlBitwardenMember(r.MqlRuntime, m)
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}

// collections lists every collection defined in the organization.
func (r *mqlBitwarden) collections() ([]any, error) {
	conn := r.conn()

	collections, err := conn.Client().ListCollections(context.Background())
	if err != nil {
		return nil, err
	}

	var all []any
	for _, c := range collections {
		res, err := newMqlBitwardenCollection(r.MqlRuntime, c)
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}

// groups lists every group defined in the organization.
func (r *mqlBitwarden) groups() ([]any, error) {
	conn := r.conn()

	groups, err := conn.Client().ListGroups(context.Background())
	if err != nil {
		return nil, err
	}

	var all []any
	for _, g := range groups {
		res, err := newMqlBitwardenGroup(r.MqlRuntime, g)
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}
