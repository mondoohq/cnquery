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

// account-level grants: SHOW GRANTS ON ACCOUNT
func (r *mqlSnowflakeAccount) grants() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	on := true
	grants, err := client.Grants.Show(ctx, &sdk.ShowGrantOptions{
		On: &sdk.ShowGrantsOn{Account: &on},
	})
	if err != nil {
		return nil, err
	}

	return convertGrants(r.MqlRuntime, grants)
}

// SHOW GRANTS TO ROLE <role>
func (r *mqlSnowflakeRole) grants() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	grants, err := client.Grants.Show(ctx, &sdk.ShowGrantOptions{
		To: &sdk.ShowGrantsTo{Role: sdk.NewAccountObjectIdentifier(r.Name.Data)},
	})
	if err != nil {
		return nil, err
	}

	return convertGrants(r.MqlRuntime, grants)
}

// SHOW GRANTS TO USER <user>
func (r *mqlSnowflakeUser) grants() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	grants, err := client.Grants.Show(ctx, &sdk.ShowGrantOptions{
		To: &sdk.ShowGrantsTo{User: sdk.NewAccountObjectIdentifier(r.Name.Data)},
	})
	if err != nil {
		return nil, err
	}

	return convertGrants(r.MqlRuntime, grants)
}

// SHOW GRANTS OF ROLE <role> -- enumerate grantees (users and roles) holding this role.
// Each grant entry has granteeName + grantedTo so callers can filter by USER vs ROLE.
func (r *mqlSnowflakeRole) grantees() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	grants, err := client.Grants.Show(ctx, &sdk.ShowGrantOptions{
		Of: &sdk.ShowGrantsOf{Role: sdk.NewAccountObjectIdentifier(r.Name.Data)},
	})
	if err != nil {
		return nil, err
	}

	return convertGrants(r.MqlRuntime, grants)
}

// accountAdmins returns all users that hold the ACCOUNTADMIN role, either directly
// or transitively via role-to-role grants.
func (r *mqlSnowflakeAccount) accountAdmins() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	holders, err := collectRoleHolders(ctx, client, "ACCOUNTADMIN")
	if err != nil {
		return nil, err
	}

	// Pull the user list once and filter to the holders set.
	users, err := client.Users.Show(ctx, &sdk.ShowUserOptions{})
	if err != nil {
		return nil, err
	}

	list := []any{}
	for i := range users {
		// Normalize user name through NewAccountObjectIdentifier so quote-stripping
		// matches the holders set (which was built from GranteeName.Name()).
		key := sdk.NewAccountObjectIdentifier(users[i].Name).Name()
		if !holders[key] {
			continue
		}
		mqlUser, err := newMqlSnowflakeUser(r.MqlRuntime, users[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlUser)
	}
	return list, nil
}

// collectRoleHolders walks role-to-role grants transitively and returns the set
// of user names that ultimately hold `roleName`. Walks users granted the role
// directly plus users granted any role that itself was granted the target role.
func collectRoleHolders(ctx context.Context, client *sdk.Client, roleName string) (map[string]bool, error) {
	users := map[string]bool{}
	visited := map[string]bool{}
	queue := []string{roleName}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current] {
			continue
		}
		visited[current] = true

		grants, err := client.Grants.Show(ctx, &sdk.ShowGrantOptions{
			Of: &sdk.ShowGrantsOf{Role: sdk.NewAccountObjectIdentifier(current)},
		})
		if err != nil {
			return nil, err
		}

		for _, g := range grants {
			switch g.GrantedTo {
			case sdk.ObjectTypeUser:
				users[g.GranteeName.Name()] = true
			case sdk.ObjectTypeRole:
				queue = append(queue, g.GranteeName.Name())
			}
		}
	}

	return users, nil
}

func convertGrants(runtime *plugin.Runtime, grants []sdk.Grant) ([]any, error) {
	list := make([]any, 0, len(grants))
	for i := range grants {
		mqlGrant, err := newMqlSnowflakeGrant(runtime, grants[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlGrant)
	}
	return list, nil
}

func newMqlSnowflakeGrant(runtime *plugin.Runtime, grant sdk.Grant) (*mqlSnowflakeGrant, error) {
	objectName := ""
	if grant.Name != nil {
		objectName = grant.Name.FullyQualifiedName()
	}
	granteeName := grant.GranteeName.Name()
	grantedOn := string(grant.GrantedOn)
	grantedTo := string(grant.GrantedTo)

	// stable composite ID: grantee + privilege + object + objectName + grantTo direction
	id := granteeName + "/" + grantedTo + "/" + grant.Privilege + "/" + grantedOn + "/" + objectName

	r, err := CreateResource(runtime, "snowflake.grant", map[string]*llx.RawData{
		"__id":        llx.StringData(id),
		"privilege":   llx.StringData(grant.Privilege),
		"grantedOn":   llx.StringData(grantedOn),
		"name":        llx.StringData(objectName),
		"grantedTo":   llx.StringData(grantedTo),
		"granteeName": llx.StringData(granteeName),
		"grantOption": llx.BoolData(grant.GrantOption),
		"grantedBy":   llx.StringData(grant.GrantedBy.Name()),
		"createdAt":   llx.TimeData(grant.CreatedOn),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlSnowflakeGrant), nil
}
