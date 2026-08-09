// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/bitwarden/connection"
)

func (r *mqlBitwarden) id() (string, error) {
	return "bitwarden", nil
}

// conn returns the Bitwarden connection backing this runtime.
func (r *mqlBitwarden) conn() *connection.BitwardenConnection {
	return r.MqlRuntime.Connection.(*connection.BitwardenConnection)
}

// organization reads the organization's account-level settings. Its cache
// key is the organization UUID extracted from the connection's client ID,
// since there is exactly one organization per connection.
func (r *mqlBitwarden) organization() (*mqlBitwardenOrganization, error) {
	conn := r.conn()

	sub, err := conn.Client().GetOrganizationSubscription(context.Background())
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(r.MqlRuntime, "bitwarden.organization", map[string]*llx.RawData{
		"__id":             llx.StringData(conn.OrgID()),
		"id":               llx.StringData(conn.OrgID()),
		"seats":            llx.IntDataPtr(sub.Seats),
		"occupiedSeats":    llx.IntDataPtr(sub.OccupiedSeats),
		"maxCollections":   llx.IntDataPtr(sub.MaxCollections),
		"maxStorageGb":     llx.IntDataPtr(sub.MaxStorageGb),
		"enabled":          llx.BoolData(sub.Enabled),
		"useSso":           llx.BoolData(sub.UseSso),
		"use2fa":           llx.BoolData(sub.UseTotp),
		"useDirectory":     llx.BoolData(sub.UseDirectory),
		"useEvents":        llx.BoolData(sub.UseEvents),
		"useGroups":        llx.BoolData(sub.UseGroups),
		"usePolicies":      llx.BoolData(sub.UsePolicies),
		"useSend":          llx.BoolData(sub.UseSend),
		"useApi":           llx.BoolData(sub.UseApi),
		"useResetPassword": llx.BoolData(sub.UseResetPassword),
		"planName":         llx.StringData(sub.PlanName),
		"businessName":     llx.StringData(sub.BusinessName),
		// The Public API's subscription endpoint has no dedicated "name"
		// field; the organization's business name is its display name.
		"name": llx.StringData(sub.BusinessName),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlBitwardenOrganization), nil
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

	members, err := conn.Client().ListMembers(context.Background())
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
