// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/okta/connection"
)

// factors returns the MFA factors enrolled by this user.
func (o *mqlOktaUser) factors() ([]any, error) {
	if o.Id.Error != nil {
		return nil, o.Id.Error
	}
	if o.Id.Data == "" {
		return nil, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	factors, _, err := client.UserFactor.ListFactors(ctx, o.Id.Data)
	if err != nil {
		return nil, err
	}

	if len(factors) == 0 {
		return []any{}, nil
	}

	list := []any{}
	for i := range factors {
		uf, ok := factors[i].(*okta.UserFactor)
		if !ok {
			continue
		}
		r, err := newMqlOktaUserFactor(o.MqlRuntime, o.Id.Data, uf)
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, nil
}

func newMqlOktaUserFactor(runtime *plugin.Runtime, userId string, factor *okta.UserFactor) (*mqlOktaUserFactor, error) {
	r, err := CreateResource(runtime, "okta.userFactor", map[string]*llx.RawData{
		"id":          llx.StringData(factor.Id),
		"factorType":  llx.StringData(factor.FactorType),
		"provider":    llx.StringData(factor.Provider),
		"status":      llx.StringData(factor.Status),
		"created":     llx.TimeDataPtr(factor.Created),
		"lastUpdated": llx.TimeDataPtr(factor.LastUpdated),
		"userId":      llx.StringData(userId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOktaUserFactor), nil
}

func (o *mqlOktaUserFactor) id() (string, error) {
	return "okta.userFactor/" + o.UserId.Data + "/" + o.Id.Data, o.Id.Error
}

// user resolves the typed user this factor belongs to.
func (o *mqlOktaUserFactor) user() (*mqlOktaUser, error) {
	if o.UserId.Error != nil {
		return nil, o.UserId.Error
	}
	if o.UserId.Data == "" {
		o.User.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	usr, _, err := client.User.GetUser(ctx, o.UserId.Data)
	if err != nil {
		return nil, err
	}
	return newMqlOktaUser(o.MqlRuntime, usr)
}
