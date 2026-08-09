// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strings"

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
	account, err := snowflakeAccount(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	grants, err := account.grantsToRole(r.Name.Data)
	if err != nil {
		return nil, err
	}
	return convertGrants(r.MqlRuntime, grants)
}

// SHOW GRANTS TO USER <user>: the roles granted to the user.
//
// This statement reports only the role, not a privilege, so privilege and
// grantOption resolve to null rather than to a zero value that would read as a
// privilege named "" that cannot be re-granted.
func (r *mqlSnowflakeUser) grants() ([]any, error) {
	account, err := snowflakeAccount(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	grants, err := account.grantsToUser(r.Name.Data)
	if err != nil {
		return nil, err
	}

	list := make([]any, 0, len(grants))
	for _, g := range grants {
		id := r.Name.Data + "/" + string(sdk.ObjectTypeUser) + "//" + string(sdk.ObjectTypeRole) + "/" + g.role
		res, err := CreateResource(r.MqlRuntime, "snowflake.grant", map[string]*llx.RawData{
			"__id":        llx.StringData(id),
			"privilege":   llx.NilData,
			"grantedOn":   llx.StringData(string(sdk.ObjectTypeRole)),
			"name":        llx.StringData(g.role),
			"grantedTo":   llx.StringData(string(sdk.ObjectTypeUser)),
			"granteeName": llx.StringData(r.Name.Data),
			"grantOption": llx.NilData,
			"grantedBy":   llx.StringData(g.grantedBy),
			"createdAt":   g.createdOn,
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, nil
}

// SHOW GRANTS OF ROLE <role> -- enumerate grantees (users and roles) holding this role.
// Each grant entry has granteeName + grantedTo so callers can filter by USER vs ROLE.
func (r *mqlSnowflakeRole) grantees() ([]any, error) {
	account, err := snowflakeAccount(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	grants, err := account.grantsOfRole(r.Name.Data)
	if err != nil {
		return nil, err
	}
	return convertGrants(r.MqlRuntime, grants)
}

// childRoles returns the roles granted to this role, one level down.
func (r *mqlSnowflakeRole) childRoles() ([]any, error) {
	account, err := snowflakeAccount(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	names, err := directChildRoles(account, r.Name.Data)
	if err != nil {
		return nil, err
	}
	return resolveRoles(r.MqlRuntime, account, names)
}

// parentRoles returns the roles this role is granted to, one level up.
func (r *mqlSnowflakeRole) parentRoles() ([]any, error) {
	account, err := snowflakeAccount(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	names, err := directParentRoles(account, r.Name.Data)
	if err != nil {
		return nil, err
	}
	return resolveRoles(r.MqlRuntime, account, names)
}

// users returns the users this role is granted to directly.
func (r *mqlSnowflakeRole) users() ([]any, error) {
	account, err := snowflakeAccount(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	names, err := directRoleUsers(account, r.Name.Data)
	if err != nil {
		return nil, err
	}
	return resolveUsers(r.MqlRuntime, account, names)
}

// effectiveRoles returns every role whose privileges this role inherits.
func (r *mqlSnowflakeRole) effectiveRoles() ([]any, error) {
	account, err := snowflakeAccount(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	names, err := inheritedRoles(account, []string{r.Name.Data})
	if err != nil {
		return nil, err
	}
	return resolveRoles(r.MqlRuntime, account, names)
}

// effectiveUsers returns every user that holds this role, directly or through
// an intermediate role.
func (r *mqlSnowflakeRole) effectiveUsers() ([]any, error) {
	account, err := snowflakeAccount(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	names, err := roleHolders(account, r.Name.Data)
	if err != nil {
		return nil, err
	}
	return resolveUsers(r.MqlRuntime, account, names)
}

// effectiveGrants returns the privileges this role holds directly plus those it
// inherits from the roles granted to it.
func (r *mqlSnowflakeRole) effectiveGrants() ([]any, error) {
	account, err := snowflakeAccount(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	inherited, err := inheritedRoles(account, []string{r.Name.Data})
	if err != nil {
		return nil, err
	}
	return collectGrants(r.MqlRuntime, account, append([]string{r.Name.Data}, inherited...))
}

// roles returns the roles granted to the user directly.
func (r *mqlSnowflakeUser) roles() ([]any, error) {
	account, err := snowflakeAccount(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	names, err := directUserRoles(account, r.Name.Data)
	if err != nil {
		return nil, err
	}
	return resolveRoles(r.MqlRuntime, account, names)
}

// effectiveRoles returns the user's direct roles plus every role those roles
// inherit.
func (r *mqlSnowflakeUser) effectiveRoles() ([]any, error) {
	account, names, err := userEffectiveRoleNames(r)
	if err != nil {
		return nil, err
	}
	return resolveRoles(r.MqlRuntime, account, names)
}

// effectiveGrants returns every privilege the user reaches through their roles.
func (r *mqlSnowflakeUser) effectiveGrants() ([]any, error) {
	account, names, err := userEffectiveRoleNames(r)
	if err != nil {
		return nil, err
	}
	return collectGrants(r.MqlRuntime, account, names)
}

// userEffectiveRoleNames resolves the user's direct roles and the roles those
// inherit, in that order, so the direct grants lead the result.
func userEffectiveRoleNames(r *mqlSnowflakeUser) (*mqlSnowflakeAccount, []string, error) {
	account, err := snowflakeAccount(r.MqlRuntime)
	if err != nil {
		return nil, nil, err
	}
	direct, err := directUserRoles(account, r.Name.Data)
	if err != nil {
		return nil, nil, err
	}
	inherited, err := inheritedRoles(account, direct)
	if err != nil {
		return nil, nil, err
	}
	return account, append(direct, inherited...), nil
}

// accountAdmins returns all users that hold the ACCOUNTADMIN role, either
// directly or transitively via role-to-role grants.
func (r *mqlSnowflakeAccount) accountAdmins() ([]any, error) {
	names, err := roleHolders(r, "ACCOUNTADMIN")
	if err != nil {
		return nil, err
	}
	return resolveUsers(r.MqlRuntime, r, names)
}

// granteeRole resolves the grantee when the privilege was granted to a role.
func (r *mqlSnowflakeGrant) granteeRole() (*mqlSnowflakeRole, error) {
	if r.GrantedTo.Data != string(sdk.ObjectTypeRole) {
		r.GranteeRole.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return resolveOwnerRole(r.MqlRuntime, r.GranteeName.Data, &r.GranteeRole)
}

// granteeUser resolves the grantee when the role was granted to a user.
func (r *mqlSnowflakeGrant) granteeUser() (*mqlSnowflakeUser, error) {
	if r.GrantedTo.Data != string(sdk.ObjectTypeUser) {
		r.GranteeUser.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return resolveUser(r.MqlRuntime, r.GranteeName.Data, &r.GranteeUser)
}

// grantedByRole resolves the role that issued the grant.
func (r *mqlSnowflakeGrant) grantedByRole() (*mqlSnowflakeRole, error) {
	return resolveOwnerRole(r.MqlRuntime, r.GrantedBy.Data, &r.GrantedByRole)
}

// resolveUser resolves a user name to a user resource, reporting the reference
// as null when the session cannot list the user rather than failing the query.
func resolveUser(runtime *plugin.Runtime, name string, field *plugin.TValue[*mqlSnowflakeUser]) (*mqlSnowflakeUser, error) {
	if name == "" {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	account, err := snowflakeAccount(runtime)
	if err != nil {
		return nil, err
	}
	index, err := account.userIndex()
	if err != nil {
		return nil, err
	}
	user, ok := index[name]
	if !ok {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return newMqlSnowflakeUser(runtime, user)
}

// grants returns the privileges granted on the database (SHOW GRANTS ON DATABASE).
func (r *mqlSnowflakeDatabase) grants() ([]any, error) {
	return showObjectGrants(r.MqlRuntime, sdk.ObjectTypeDatabase, sdk.NewAccountObjectIdentifier(r.Name.Data))
}

// futureGrants returns the grants that will apply to objects created in the
// database later (SHOW FUTURE GRANTS IN DATABASE).
func (r *mqlSnowflakeDatabase) futureGrants() ([]any, error) {
	id := sdk.NewAccountObjectIdentifier(r.Name.Data)
	return showFutureGrants(r.MqlRuntime, &sdk.ShowGrantsIn{Database: &id})
}

// grants returns the privileges granted on the schema (SHOW GRANTS ON SCHEMA).
func (r *mqlSnowflakeSchema) grants() ([]any, error) {
	return showObjectGrants(r.MqlRuntime, sdk.ObjectTypeSchema,
		sdk.NewDatabaseObjectIdentifier(r.DatabaseName.Data, r.Name.Data))
}

// futureGrants returns the grants that will apply to objects created in the
// schema later (SHOW FUTURE GRANTS IN SCHEMA).
func (r *mqlSnowflakeSchema) futureGrants() ([]any, error) {
	id := sdk.NewDatabaseObjectIdentifier(r.DatabaseName.Data, r.Name.Data)
	return showFutureGrants(r.MqlRuntime, &sdk.ShowGrantsIn{Schema: &id})
}

// grants returns the objects an outbound share exposes (SHOW GRANTS TO SHARE).
//
// An inbound share is owned by the account that created it, so this account
// grants nothing to it and the statement does not accept its qualified name.
func (r *mqlSnowflakeShare) grants() ([]any, error) {
	if !strings.EqualFold(r.Kind.Data, "OUTBOUND") {
		return []any{}, nil
	}

	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	grants, err := conn.Client().Grants.Show(context.Background(), &sdk.ShowGrantOptions{
		To: &sdk.ShowGrantsTo{
			Share: &sdk.ShowGrantsToShare{Name: sdk.NewAccountObjectIdentifier(r.Name.Data)},
		},
	})
	if err != nil {
		return nil, err
	}
	return convertGrants(r.MqlRuntime, grants)
}

// grants returns the privileges granted to the database role.
func (r *mqlSnowflakeDatabaseRole) grants() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	grants, err := conn.Client().Grants.Show(context.Background(), &sdk.ShowGrantOptions{
		To: &sdk.ShowGrantsTo{
			DatabaseRole: sdk.NewDatabaseObjectIdentifier(r.DatabaseName.Data, r.Name.Data),
		},
	})
	if err != nil {
		return nil, err
	}
	return convertGrants(r.MqlRuntime, grants)
}

// grantees returns the roles and database roles this database role is granted to.
func (r *mqlSnowflakeDatabaseRole) grantees() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	grants, err := conn.Client().Grants.Show(context.Background(), &sdk.ShowGrantOptions{
		Of: &sdk.ShowGrantsOf{
			DatabaseRole: sdk.NewDatabaseObjectIdentifier(r.DatabaseName.Data, r.Name.Data),
		},
	})
	if err != nil {
		return nil, err
	}
	return convertGrants(r.MqlRuntime, grants)
}

func showObjectGrants(runtime *plugin.Runtime, objectType sdk.ObjectType, id sdk.ObjectIdentifier) ([]any, error) {
	conn := runtime.Connection.(*connection.SnowflakeConnection)
	grants, err := conn.Client().Grants.Show(context.Background(), &sdk.ShowGrantOptions{
		On: &sdk.ShowGrantsOn{Object: &sdk.Object{ObjectType: objectType, Name: id}},
	})
	if err != nil {
		return nil, err
	}
	return convertGrants(runtime, grants)
}

func showFutureGrants(runtime *plugin.Runtime, in *sdk.ShowGrantsIn) ([]any, error) {
	conn := runtime.Connection.(*connection.SnowflakeConnection)
	future := true
	grants, err := conn.Client().Grants.Show(context.Background(), &sdk.ShowGrantOptions{
		Future: &future,
		In:     in,
	})
	if err != nil {
		return nil, err
	}
	return convertGrants(runtime, grants)
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

// snowflakeGrantID is the stable cache key for a grant: grantee, direction,
// privilege, and the object the privilege is on. Callers that merge grants from
// several statements deduplicate on this so a privilege reached twice through
// the role hierarchy yields one resource.
func snowflakeGrantID(grant sdk.Grant) string {
	objectName := ""
	if grant.Name != nil {
		objectName = grant.Name.FullyQualifiedName()
	}
	return grant.GranteeName.Name() + "/" + string(grant.GrantedTo) + "/" +
		grant.Privilege + "/" + string(grant.GrantedOn) + "/" + objectName
}

func newMqlSnowflakeGrant(runtime *plugin.Runtime, grant sdk.Grant) (*mqlSnowflakeGrant, error) {
	objectName := ""
	if grant.Name != nil {
		objectName = grant.Name.FullyQualifiedName()
	}

	// A future grant reports the object type it will apply to in grant_on
	// rather than granted_on, which is empty until the object exists.
	grantedOn := grant.GrantedOn
	if grantedOn == "" {
		grantedOn = grant.GrantOn
	}
	grantedTo := grant.GrantedTo
	if grantedTo == "" {
		grantedTo = grant.GrantTo
	}

	r, err := CreateResource(runtime, "snowflake.grant", map[string]*llx.RawData{
		"__id":        llx.StringData(snowflakeGrantID(grant)),
		"privilege":   llx.StringData(grant.Privilege),
		"grantedOn":   llx.StringData(string(grantedOn)),
		"name":        llx.StringData(objectName),
		"grantedTo":   llx.StringData(string(grantedTo)),
		"granteeName": llx.StringData(grant.GranteeName.Name()),
		"grantOption": llx.BoolData(grant.GrantOption),
		"grantedBy":   llx.StringData(grant.GrantedBy.Name()),
		"createdAt":   llx.TimeData(grant.CreatedOn),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlSnowflakeGrant), nil
}
