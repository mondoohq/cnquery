// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/snowflake/connection"
)

func (r *mqlSnowflakeDatabase) roles() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	dbID := sdk.NewAccountObjectIdentifier(r.Name.Data)

	roles, err := conn.Client().DatabaseRoles.Show(
		context.Background(),
		sdk.NewShowDatabaseRoleRequest(dbID),
	)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(roles))
	for i := range roles {
		mqlRole, err := newMqlSnowflakeDatabaseRole(r.MqlRuntime, r.Name.Data, roles[i])
		if err != nil {
			return nil, err
		}
		out = append(out, mqlRole)
	}
	return out, nil
}

func newMqlSnowflakeDatabaseRole(runtime *plugin.Runtime, databaseName string, role sdk.DatabaseRole) (*mqlSnowflakeDatabaseRole, error) {
	// SHOW DATABASE ROLES does not populate DatabaseName on the row; backfill from caller.
	if role.DatabaseName == "" {
		role.DatabaseName = databaseName
	}

	r, err := CreateResource(runtime, "snowflake.databaseRole", map[string]*llx.RawData{
		"__id":                   llx.StringData(role.ID().FullyQualifiedName()),
		"name":                   llx.StringData(role.Name),
		"databaseName":           llx.StringData(role.DatabaseName),
		"owner":                  llx.StringData(role.Owner),
		"ownerRoleType":          llx.StringData(role.OwnerRoleType),
		"comment":                llx.StringData(role.Comment),
		"isCurrent":              llx.BoolData(role.IsCurrent),
		"isInherited":            llx.BoolData(role.IsInherited),
		"grantedToRoles":         llx.IntData(role.GrantedToRoles),
		"grantedToDatabaseRoles": llx.IntData(role.GrantedToDatabaseRoles),
		"grantedDatabaseRoles":   llx.IntData(role.GrantedDatabaseRoles),
		"createdOn":              llx.StringData(role.CreatedOn),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlSnowflakeDatabaseRole), nil
}
