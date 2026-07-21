// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/snowflake/connection"
	"go.mondoo.com/mql/v13/types"
)

type mqlSnowflakeUserInternal struct {
	cacheOwner string
}

// parseSecondaryRoles converts Snowflake's default_secondary_roles column into
// a list of role names. SHOW USERS returns it as a JSON array string (for
// example `["ALL"]`), so parse that; fall back to a single bare value if the
// server ever returns an unquoted string, and to an empty list when unset.
func parseSecondaryRoles(raw string) []any {
	out := []any{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out
	}
	var arr []string
	if err := json.Unmarshal([]byte(raw), &arr); err == nil {
		for _, s := range arr {
			out = append(out, s)
		}
		return out
	}
	return append(out, raw)
}

func (r *mqlSnowflakeAccount) users() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	users, err := client.Users.Show(ctx, &sdk.ShowUserOptions{})
	if err != nil {
		return nil, err
	}

	list := []any{}
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
		"__id":                  llx.StringData(user.ID().FullyQualifiedName()),
		"name":                  llx.StringData(user.Name),
		"login":                 llx.StringData(user.LoginName),
		"displayName":           llx.StringData(user.DisplayName),
		"firstName":             llx.StringData(user.FirstName),
		"lastName":              llx.StringData(user.LastName),
		"email":                 llx.StringData(user.Email),
		"comment":               llx.StringData(user.Comment),
		"defaultWarehouse":      llx.StringData(user.DefaultWarehouse),
		"defaultNamespace":      llx.StringData(user.DefaultNamespace),
		"defaultRole":           llx.StringData(user.DefaultRole),
		"disabled":              llx.BoolData(user.Disabled),
		"hasPassword":           llx.BoolData(user.HasPassword),
		"hasRsaPublicKey":       llx.BoolData(user.HasRsaPublicKey),
		"mustChangePassword":    llx.BoolData(user.MustChangePassword),
		"lastSuccessLogin":      llx.TimeData(user.LastSuccessLogin),
		"lockedUntil":           llx.TimeData(user.LockedUntilTime),
		"createdAt":             llx.TimeData(user.CreatedOn),
		"expiresAt":             llx.TimeData(user.ExpiresAtTime),
		"extAuthnDuo":           llx.BoolData(user.ExtAuthnDuo),
		"extAuthnUid":           llx.StringData(user.ExtAuthnUid),
		"type":                  llx.StringData(user.Type),
		"hasMfa":                llx.BoolData(user.HasMfa),
		"snowflakeLock":         llx.BoolData(user.SnowflakeLock),
		"defaultSecondaryRoles": llx.ArrayData(parseSecondaryRoles(user.DefaultSecondaryRoles), types.String),
		"minsToBypassMfa":       llx.StringData(user.MinsToBypassMfa),
	})
	if err != nil {
		return nil, err
	}
	mqlResource := r.(*mqlSnowflakeUser)
	mqlResource.cacheOwner = user.Owner
	return mqlResource, nil
}

// owner resolves the role that owns the user. The owning role is always an
// account role, so it hydrates through snowflake.role's init from the cached
// owner name.
func (r *mqlSnowflakeUser) owner() (*mqlSnowflakeRole, error) {
	if r.cacheOwner == "" {
		r.Owner.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	role, err := snowflakeRoleByName(r.MqlRuntime, r.cacheOwner)
	if err != nil {
		return nil, err
	}
	if role == nil {
		r.Owner.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return role, nil
}

// daysSinceLastLogin returns whole days since the user's last successful login.
// Returns -1 if the user has never logged in (no recorded last-login time).
func (r *mqlSnowflakeUser) daysSinceLastLogin() (int64, error) {
	last := r.LastSuccessLogin.Data
	if last == nil || last.IsZero() {
		return -1, nil
	}
	delta := time.Since(*last)
	if delta < 0 {
		return 0, nil
	}
	return int64(delta / (24 * time.Hour)), nil
}

func (r *mqlSnowflakeUser) parameters() ([]any, error) {
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

	list := []any{}
	for i := range parameters {
		mqlResource, err := newMqlSnowflakeParameter(r.MqlRuntime, "user/"+r.Name.Data, parameters[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlResource)
	}

	return list, nil
}

func (r *mqlSnowflakeUser) defaultRoleRef() (*mqlSnowflakeRole, error) {
	return resolveOwnerRole(r.MqlRuntime, r.DefaultRole.Data, &r.DefaultRoleRef)
}

func (r *mqlSnowflakeUser) defaultWarehouseRef() (*mqlSnowflakeWarehouse, error) {
	name := r.DefaultWarehouse.Data
	if name == "" {
		r.DefaultWarehouseRef.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	wh, err := NewResource(r.MqlRuntime, "snowflake.warehouse", map[string]*llx.RawData{
		"name": llx.StringData(name),
	})
	if err != nil {
		return nil, err
	}
	return wh.(*mqlSnowflakeWarehouse), nil
}
