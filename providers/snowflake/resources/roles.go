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

func (r *mqlSnowflakeAccount) roles() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	roles, err := client.Roles.Show(ctx, &sdk.ShowRoleRequest{})
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range roles {
		mqlRole, err := newMqlSnowflakeRole(r.MqlRuntime, roles[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlRole)
	}

	return list, nil
}

func newMqlSnowflakeRole(runtime *plugin.Runtime, role sdk.Role) (*mqlSnowflakeRole, error) {
	r, err := CreateResource(runtime, "snowflake.role", map[string]*llx.RawData{
		"__id":            llx.StringData(role.ID().FullyQualifiedName()),
		"name":            llx.StringData(role.Name),
		"isDefault":       llx.BoolData(role.IsDefault),
		"isCurrent":       llx.BoolData(role.IsCurrent),
		"isInherited":     llx.BoolData(role.IsInherited),
		"assignedToUsers": llx.IntData(role.AssignedToUsers),
		"grantedToRoles":  llx.IntData(role.GrantedToRoles),
		"grantedRoles":    llx.IntData(role.GrantedRoles),
		"owner":           llx.StringData(role.Owner),
		"comment":         llx.StringData(role.Comment),
	})
	if err != nil {
		return nil, err
	}
	mqlResource := r.(*mqlSnowflakeRole)
	return mqlResource, nil
}
