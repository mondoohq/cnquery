// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"database/sql"
	"fmt"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// initMssqlLogin resolves a single server login by name so typed references
// (for example mssql.databaseUser.login) return a fully populated login. Bulk
// listing goes through mssql.server.logins via CreateResource, which skips init.
func initMssqlLogin(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	nameRaw, ok := args["name"]
	if !ok {
		return args, nil, nil
	}
	name, _ := nameRaw.Value.(string)
	if name == "" {
		return nil, nil, fmt.Errorf("mssql.login requires a non-empty name")
	}

	client, err := mssqlClient(runtime)
	if err != nil {
		return nil, nil, err
	}
	q := `SELECT ` + loginColumns + `
		FROM sys.server_principals p
		LEFT JOIN sys.sql_logins sl ON p.principal_id = sl.principal_id
		WHERE p.name = @p1 AND p.type IN ('S','U','G','C','K')`
	rows, err := client.QueryContext(mssqlContext(), q, sql.Named("p1", name))
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, nil, err
		}
		return nil, nil, fmt.Errorf("mssql.login %q not found", name)
	}
	res, err := scanLogin(runtime, rows)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

// newMssqlLogin builds a login resource from a scanned server-principals row.
// The SID is canonicalized to S-1-5-... form for the exposed fields, and its
// binary form is cached for cross-database user matching.
func newMssqlLogin(runtime *plugin.Runtime, principalID int64, name, typeDesc, defaultDB string, sid []byte,
	createDate, modifyDate, passwordLastSet sql.NullTime, isDisabled, isFixedRole bool,
	isPolicyChecked, isExpirationChecked sql.NullBool, mustChange sql.NullInt64) (*mqlMssqlLogin, error) {
	instanceID := mssqlConnection(runtime).InstanceID()
	isAD := isActiveDirectoryType(typeDesc)
	canonicalSid := sidString(sid)

	fields := map[string]*llx.RawData{
		"__id":                       llx.StringData(serverPrincipalID(instanceID, name)),
		"name":                       llx.StringData(name),
		"principalId":                llx.IntData(principalID),
		"type":                       llx.StringData(typeDesc),
		"sid":                        llx.StringData(canonicalSid),
		"isDisabled":                 llx.BoolData(isDisabled),
		"isFixedRole":                llx.BoolData(isFixedRole),
		"defaultDatabase":            llx.StringData(defaultDB),
		"createDate":                 llx.TimeDataPtr(nullTime(createDate)),
		"modifyDate":                 llx.TimeDataPtr(nullTime(modifyDate)),
		"isActiveDirectoryPrincipal": llx.BoolData(isAD),
		"passwordLastSetTime":        llx.TimeDataPtr(nullTime(passwordLastSet)),
	}
	if isAD {
		fields["activeDirectoryPrincipal"] = llx.StringData(name)
		fields["activeDirectorySid"] = llx.StringData(canonicalSid)
	}
	if isPolicyChecked.Valid {
		fields["isPolicyChecked"] = llx.BoolData(isPolicyChecked.Bool)
	}
	if isExpirationChecked.Valid {
		fields["isExpirationChecked"] = llx.BoolData(isExpirationChecked.Bool)
	}
	if mustChange.Valid {
		fields["mustChange"] = llx.BoolData(mustChange.Int64 == 1)
	}

	res, err := CreateResource(runtime, "mssql.login", fields)
	if err != nil {
		return nil, err
	}
	login := res.(*mqlMssqlLogin)
	login.cacheSid = sid
	return login, nil
}

// newMssqlServerRoleRef builds a server-role resource with the fields available
// from a role-membership join; unfetched scalars resolve to null.
func newMssqlServerRoleRef(runtime *plugin.Runtime, principalID int64, name string, isFixedRole bool) (*mqlMssqlServerRole, error) {
	instanceID := mssqlConnection(runtime).InstanceID()
	res, err := CreateResource(runtime, "mssql.serverRole", map[string]*llx.RawData{
		"__id":        llx.StringData(serverPrincipalID(instanceID, name)),
		"name":        llx.StringData(name),
		"principalId": llx.IntData(principalID),
		"isFixedRole": llx.BoolData(isFixedRole),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMssqlServerRole), nil
}

// --- mssql.login ------------------------------------------------------------

func (c *mqlMssqlLogin) explicitPermissions() ([]any, error) {
	pid := c.PrincipalId.Data
	return serverPermissionsFor(c.MqlRuntime, c.__id, &pid)
}

func (c *mqlMssqlLogin) memberOfRoles() ([]any, error) {
	return serverRolesForMember(c.MqlRuntime, c.PrincipalId.Data)
}

func (c *mqlMssqlLogin) databaseUsers() ([]any, error) {
	client, err := mssqlClient(c.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if len(c.cacheSid) == 0 {
		return []any{}, nil
	}

	dbRows, err := client.QueryContext(mssqlContext(),
		"SELECT name FROM sys.databases WHERE state = 0")
	if err != nil {
		return nil, err
	}
	var dbNames []string
	for dbRows.Next() {
		var n string
		if err := dbRows.Scan(&n); err != nil {
			dbRows.Close()
			return nil, err
		}
		dbNames = append(dbNames, n)
	}
	dbRows.Close()
	if err := dbRows.Err(); err != nil {
		return nil, err
	}

	list := []any{}
	for _, dbName := range dbNames {
		users, err := databaseUsersMatching(c.MqlRuntime, dbName, c.cacheSid)
		if err != nil {
			// A single inaccessible database should not fail the whole lookup.
			continue
		}
		list = append(list, users...)
	}
	return list, nil
}

// serverRolesForMember lists the server roles a principal is a direct member of.
func serverRolesForMember(runtime *plugin.Runtime, memberPrincipalID int64) ([]any, error) {
	client, err := mssqlClient(runtime)
	if err != nil {
		return nil, err
	}
	const q = `SELECT r.principal_id, r.name, r.is_fixed_role
		FROM sys.server_role_members rm
		JOIN sys.server_principals r ON rm.role_principal_id = r.principal_id
		WHERE rm.member_principal_id = @p1`
	rows, err := client.QueryContext(mssqlContext(), q, sql.Named("p1", memberPrincipalID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []any{}
	for rows.Next() {
		var pid int64
		var name string
		var isFixedRole bool
		if err := rows.Scan(&pid, &name, &isFixedRole); err != nil {
			return nil, err
		}
		role, err := newMssqlServerRoleRef(runtime, pid, name, isFixedRole)
		if err != nil {
			return nil, err
		}
		list = append(list, role)
	}
	return list, rows.Err()
}

// --- mssql.serverRole -------------------------------------------------------

func (c *mqlMssqlServerRole) members() ([]any, error) {
	client, err := mssqlClient(c.MqlRuntime)
	if err != nil {
		return nil, err
	}
	// Only login-type members are returned as mssql.login; nested role members
	// are reachable through each role's memberOfRoles.
	q := `SELECT ` + loginColumns + `
		FROM sys.server_role_members rm
		JOIN sys.server_principals p ON rm.member_principal_id = p.principal_id
		LEFT JOIN sys.sql_logins sl ON p.principal_id = sl.principal_id
		WHERE rm.role_principal_id = @p1 AND p.type IN ('S','U','G','C','K')`
	rows, err := client.QueryContext(mssqlContext(), q, sql.Named("p1", c.PrincipalId.Data))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []any{}
	for rows.Next() {
		login, err := scanLogin(c.MqlRuntime, rows)
		if err != nil {
			return nil, err
		}
		list = append(list, login)
	}
	return list, rows.Err()
}

func (c *mqlMssqlServerRole) memberOfRoles() ([]any, error) {
	return serverRolesForMember(c.MqlRuntime, c.PrincipalId.Data)
}

func (c *mqlMssqlServerRole) explicitPermissions() ([]any, error) {
	pid := c.PrincipalId.Data
	return serverPermissionsFor(c.MqlRuntime, c.__id, &pid)
}

// --- mssql.databaseUser -----------------------------------------------------

func (c *mqlMssqlDatabaseUser) login() (*mqlMssqlLogin, error) {
	if c.cacheLoginName == "" {
		c.Login.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(c.MqlRuntime, "mssql.login", map[string]*llx.RawData{
		"name": llx.StringData(c.cacheLoginName),
	})
	if err != nil {
		// The mapped login may not be visible (orphaned user); treat as null.
		c.Login.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return res.(*mqlMssqlLogin), nil
}

func (c *mqlMssqlDatabaseUser) explicitPermissions() ([]any, error) {
	pid := c.PrincipalId.Data
	return databasePermissionsFor(c.MqlRuntime, c.cacheDatabase, c.__id, &pid)
}

func (c *mqlMssqlDatabaseUser) memberOfRoles() ([]any, error) {
	return databaseRolesForMember(c.MqlRuntime, c.cacheDatabase, c.PrincipalId.Data)
}

// --- mssql.databaseRole -----------------------------------------------------

func (c *mqlMssqlDatabaseRole) members() ([]any, error) {
	client, err := mssqlClient(c.MqlRuntime)
	if err != nil {
		return nil, err
	}
	db := quoteName(c.cacheDatabase)
	q := `SELECT m.principal_id, m.name, m.type_desc, m.authentication_type_desc,
		ISNULL(m.default_schema_name, ''), CONVERT(VARCHAR(85), m.sid, 1),
		m.create_date, m.modify_date, ISNULL(SUSER_SNAME(m.sid), '')
		FROM ` + db + `.sys.database_role_members rm
		JOIN ` + db + `.sys.database_principals m ON rm.member_principal_id = m.principal_id
		WHERE rm.role_principal_id = @p1`
	rows, err := client.QueryContext(mssqlContext(), q, sql.Named("p1", c.PrincipalId.Data))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []any{}
	for rows.Next() {
		u, err := scanDatabaseUser(c.MqlRuntime, c.cacheDatabase, rows)
		if err != nil {
			return nil, err
		}
		list = append(list, u)
	}
	return list, rows.Err()
}

func (c *mqlMssqlDatabaseRole) memberOfRoles() ([]any, error) {
	return databaseRolesForMember(c.MqlRuntime, c.cacheDatabase, c.PrincipalId.Data)
}

func (c *mqlMssqlDatabaseRole) explicitPermissions() ([]any, error) {
	pid := c.PrincipalId.Data
	return databasePermissionsFor(c.MqlRuntime, c.cacheDatabase, c.__id, &pid)
}

// databaseRolesForMember lists the database roles a principal is a direct
// member of, within one database.
func databaseRolesForMember(runtime *plugin.Runtime, database string, memberPrincipalID int64) ([]any, error) {
	client, err := mssqlClient(runtime)
	if err != nil {
		return nil, err
	}
	db := quoteName(database)
	q := `SELECT r.principal_id, r.name, r.is_fixed_role
		FROM ` + db + `.sys.database_role_members rm
		JOIN ` + db + `.sys.database_principals r ON rm.role_principal_id = r.principal_id
		WHERE rm.member_principal_id = @p1`
	rows, err := client.QueryContext(mssqlContext(), q, sql.Named("p1", memberPrincipalID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	instanceID := mssqlConnection(runtime).InstanceID()
	dbID := databaseIdentifier(instanceID, database)
	list := []any{}
	for rows.Next() {
		var pid int64
		var name string
		var isFixedRole bool
		if err := rows.Scan(&pid, &name, &isFixedRole); err != nil {
			return nil, err
		}
		res, err := CreateResource(runtime, "mssql.databaseRole", map[string]*llx.RawData{
			"__id":        llx.StringData(databasePrincipalID(dbID, name)),
			"name":        llx.StringData(name),
			"principalId": llx.IntData(pid),
			"isFixedRole": llx.BoolData(isFixedRole),
		})
		if err != nil {
			return nil, err
		}
		role := res.(*mqlMssqlDatabaseRole)
		role.cacheDatabase = database
		list = append(list, role)
	}
	return list, rows.Err()
}

// --- mssql.applicationRole --------------------------------------------------

func (c *mqlMssqlApplicationRole) explicitPermissions() ([]any, error) {
	pid := c.PrincipalId.Data
	return databasePermissionsFor(c.MqlRuntime, c.cacheDatabase, c.__id, &pid)
}

func (c *mqlMssqlApplicationRole) memberOfRoles() ([]any, error) {
	return databaseRolesForMember(c.MqlRuntime, c.cacheDatabase, c.PrincipalId.Data)
}
