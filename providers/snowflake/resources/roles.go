// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/snowflake/connection"
)

// initSnowflakeRole resolves a single role by name so typed references (such as
// snowflake.user.owner) can hydrate a full role from just its name. A caller
// that already supplied more than the name is left untouched.
func initSnowflakeRole(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	nameRaw, ok := args["name"]
	if !ok {
		return args, nil, nil
	}
	name, _ := nameRaw.Value.(string)
	if name == "" {
		return nil, nil, fmt.Errorf("snowflake.role requires a non-empty name")
	}

	conn := runtime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	roles, err := client.Roles.Show(ctx, sdk.NewShowRoleRequest().WithLike(sdk.NewLikeRequest(name)))
	if err != nil {
		return nil, nil, err
	}
	for i := range roles {
		if roles[i].Name == name {
			res, err := newMqlSnowflakeRole(runtime, roles[i])
			if err != nil {
				return nil, nil, err
			}
			return nil, res, nil
		}
	}
	return nil, nil, fmt.Errorf("snowflake.role %q not found", name)
}

// snowflakeRoleByName resolves a role name to a typed snowflake.role, hydrated
// through the role's init. It returns nil for an empty name; callers should set
// the field's null state before calling in that case.
func snowflakeRoleByName(runtime *plugin.Runtime, name string) (*mqlSnowflakeRole, error) {
	if name == "" {
		return nil, nil
	}
	role, err := NewResource(runtime, "snowflake.role", map[string]*llx.RawData{
		"name": llx.StringData(name),
	})
	if err != nil {
		return nil, err
	}
	return role.(*mqlSnowflakeRole), nil
}

func (r *mqlSnowflakeAccount) roles() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	roles, err := client.Roles.Show(ctx, &sdk.ShowRoleRequest{})
	if err != nil {
		return nil, err
	}

	list := []any{}
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

// resolveOwnerRole resolves an account role name to a typed snowflake.role,
// hydrated through the role's init. It sets the field's null state when the
// owner name is empty (an unowned object) so the runtime records the null.
func resolveOwnerRole(runtime *plugin.Runtime, name string, field *plugin.TValue[*mqlSnowflakeRole]) (*mqlSnowflakeRole, error) {
	if name == "" {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return snowflakeRoleByName(runtime, name)
}

func (r *mqlSnowflakeRole) ownerRole() (*mqlSnowflakeRole, error) {
	return resolveOwnerRole(r.MqlRuntime, r.Owner.Data, &r.OwnerRole)
}
