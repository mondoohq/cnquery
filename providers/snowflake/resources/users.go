// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/snowflake/connection"
)

func (r *mqlSnowflakeAccount) users() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	users, err := client.Users.Show(ctx, &sdk.ShowUserOptions{})
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range users {
		mqlUser, err := newMqlSnowflakeUser(r.MqlRuntime, users[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlUser)
	}

	return list, nil
}

// https://docs.snowflake.com/en/sql-reference/sql/create-user
func newMqlSnowflakeUser(runtime *plugin.Runtime, user sdk.User) (*mqlSnowflakeUser, error) {
	r, err := CreateResource(runtime, "snowflake.user", map[string]*llx.RawData{
		"__id":               llx.StringData(user.ID().FullyQualifiedName()),
		"name":               llx.StringData(user.Name),
		"login":              llx.StringData(user.LoginName),
		"displayName":        llx.StringData(user.DisplayName),
		"firstName":          llx.StringData(user.FirstName),
		"lastName":           llx.StringData(user.LastName),
		"email":              llx.StringData(user.Email),
		"comment":            llx.StringData(user.Comment),
		"defaultWarehouse":   llx.StringData(user.DefaultWarehouse),
		"defaultNamespace":   llx.StringData(user.DefaultNamespace),
		"defaultRole":        llx.StringData(user.DefaultRole),
		"disabled":           llx.BoolData(user.Disabled),
		"hasPassword":        llx.BoolData(user.HasPassword),
		"hasRsaPublicKey":    llx.BoolData(user.HasRsaPublicKey),
		"mustChangePassword": llx.BoolData(user.MustChangePassword),
		"lastSuccessLogin":   llx.TimeData(user.LastSuccessLogin),
		"lockedUntil":        llx.TimeData(user.LockedUntilTime),
		"createdAt":          llx.TimeData(user.CreatedOn),
		"expiresAt":          llx.TimeData(user.ExpiresAtTime),
		"extAuthnDuo":        llx.BoolData(user.ExtAuthnDuo),
		"extAuthnUid":        llx.StringData(user.ExtAuthnUid),
	})
	if err != nil {
		return nil, err
	}
	mqlResource := r.(*mqlSnowflakeUser)
	return mqlResource, nil
}

func (r *mqlSnowflakeUser) parameters() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	parameters, err := client.Parameters.ShowParameters(ctx, &sdk.ShowParametersOptions{
		In: &sdk.ParametersIn{
			User: sdk.NewAccountObjectIdentifier(r.Name.Data),
		},
	})
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range parameters {
		mqlResource, err := newMqlSnowflakeParameter(r.MqlRuntime, parameters[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlResource)
	}

	return list, nil
}
