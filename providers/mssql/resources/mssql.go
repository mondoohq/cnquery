// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/mssql/connection"
)

func (r *mqlMssql) id() (string, error) {
	return "mssql", nil
}

// mssqlConnection returns the active connection for a resource's runtime.
func mssqlConnection(runtime *plugin.Runtime) *connection.MssqlConnection {
	return runtime.Connection.(*connection.MssqlConnection)
}

// mssqlClient returns the shared database handle for a resource's runtime.
func mssqlClient(runtime *plugin.Runtime) (*sql.DB, error) {
	return mssqlConnection(runtime).Client()
}

func mssqlContext() context.Context {
	return context.Background()
}

// quoteName brackets a SQL Server identifier and escapes embedded brackets so
// it is safe to interpolate a database or object name into a query.
func quoteName(name string) string {
	return "[" + strings.ReplaceAll(name, "]", "]]") + "]"
}

// sidString renders a binary security identifier in canonical S-1-5-... form so
// it joins to the identifier an Active Directory graph uses. Values that are
// not valid NT SIDs (for example the 16-byte SID of a SQL login) fall back to a
// 0x hex string, and an empty SID renders as an empty string.
func sidString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	// Guard the length before indexing the revision and sub-authority-count
	// bytes; short principal SIDs would otherwise panic.
	if len(b) < 8 {
		return "0x" + strings.ToUpper(hex.EncodeToString(b))
	}
	revision := b[0]
	subCount := int(b[1])
	if revision != 1 || len(b) != 8+4*subCount {
		return "0x" + strings.ToUpper(hex.EncodeToString(b))
	}
	var authority uint64
	for i := 2; i < 8; i++ {
		authority = authority<<8 | uint64(b[i])
	}
	s := fmt.Sprintf("S-%d-%d", revision, authority)
	for i := 0; i < subCount; i++ {
		off := 8 + 4*i
		s += fmt.Sprintf("-%d", binary.LittleEndian.Uint32(b[off:off+4]))
	}
	return s
}

// isActiveDirectoryType reports whether a SQL Server principal type maps to a
// Windows/Active Directory principal.
func isActiveDirectoryType(typeDesc string) bool {
	switch typeDesc {
	case "WINDOWS_LOGIN", "WINDOWS_GROUP", "WINDOWS_USER", "EXTERNAL_USER", "EXTERNAL_GROUP":
		return true
	default:
		return false
	}
}

// --- stable identifier builders ---------------------------------------------

func serverPrincipalID(instanceID, name string) string {
	return name + "@" + instanceID
}

func databaseIdentifier(instanceID, database string) string {
	return instanceID + "\\" + database
}

func databasePrincipalID(databaseID, name string) string {
	return name + "@" + databaseID
}

// permissionResourceID mints a stable, unique cache key for a permission row so
// each grant on a principal is a distinct resource rather than colliding on a
// shared id.
func permissionResourceID(parentID, class, permission, state, grantee string) string {
	return parentID + "/perm/" + class + "/" + permission + "/" + state + "/" + grantee
}

// --- shared builders --------------------------------------------------------

// newMssqlPermission creates a permission resource with a composite __id.
func newMssqlPermission(runtime *plugin.Runtime, parentID, class, permission, state, grantee string) (*mqlMssqlPermission, error) {
	res, err := CreateResource(runtime, "mssql.permission", map[string]*llx.RawData{
		"__id":           llx.StringData(permissionResourceID(parentID, class, permission, state, grantee)),
		"permissionName": llx.StringData(permission),
		"state":          llx.StringData(state),
		"class":          llx.StringData(class),
		"granteeName":    llx.StringData(grantee),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMssqlPermission), nil
}

// serverPermissionsFor lists the explicit server-level permissions granted to a
// single principal, or all server permissions when principalID is nil.
func serverPermissionsFor(runtime *plugin.Runtime, parentID string, principalID *int64) ([]any, error) {
	client, err := mssqlClient(runtime)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT p.class_desc, p.permission_name, p.state_desc, ISNULL(pr.name, '')
		FROM sys.server_permissions p
		LEFT JOIN sys.server_principals pr ON p.grantee_principal_id = pr.principal_id`
	if principalID != nil {
		query += "\n\t\tWHERE p.grantee_principal_id = @p1"
	}

	var rows *sql.Rows
	if principalID != nil {
		rows, err = client.QueryContext(mssqlContext(), query, sql.Named("p1", *principalID))
	} else {
		rows, err = client.QueryContext(mssqlContext(), query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []any{}
	for rows.Next() {
		var class, perm, state, grantee string
		if err := rows.Scan(&class, &perm, &state, &grantee); err != nil {
			return nil, err
		}
		p, err := newMssqlPermission(runtime, parentID, class, perm, state, grantee)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// databasePermissionsFor lists the explicit database-level permissions granted
// to a single principal, or all database permissions when principalID is nil.
func databasePermissionsFor(runtime *plugin.Runtime, database, parentID string, principalID *int64) ([]any, error) {
	client, err := mssqlClient(runtime)
	if err != nil {
		return nil, err
	}
	db := quoteName(database)

	query := `
		SELECT p.class_desc, p.permission_name, p.state_desc, ISNULL(pr.name, '')
		FROM ` + db + `.sys.database_permissions p
		LEFT JOIN ` + db + `.sys.database_principals pr ON p.grantee_principal_id = pr.principal_id`
	if principalID != nil {
		query += "\n\t\tWHERE p.grantee_principal_id = @p1"
	}

	var rows *sql.Rows
	if principalID != nil {
		rows, err = client.QueryContext(mssqlContext(), query, sql.Named("p1", *principalID))
	} else {
		rows, err = client.QueryContext(mssqlContext(), query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []any{}
	for rows.Next() {
		var class, perm, state, grantee string
		if err := rows.Scan(&class, &perm, &state, &grantee); err != nil {
			return nil, err
		}
		p, err := newMssqlPermission(runtime, parentID, class, perm, state, grantee)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}
