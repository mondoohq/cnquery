// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/postgres/connection"
)

func (r *mqlPostgres) id() (string, error) {
	return "postgres", nil
}

func pgConnection(runtime *plugin.Runtime) *connection.PostgresConnection {
	return runtime.Connection.(*connection.PostgresConnection)
}

// pgPool returns the connection pool for a database (empty = server database).
func pgPool(runtime *plugin.Runtime, database string) (*pgxpool.Pool, error) {
	return pgConnection(runtime).Client(database)
}

func pgContext() context.Context {
	return context.Background()
}

func pgSystemID(runtime *plugin.Runtime) (string, error) {
	return pgConnection(runtime).SystemID()
}

// --- stable identifier builders ---------------------------------------------

func roleResourceID(systemID, name string) string {
	return name + "@" + systemID
}

func databaseResourceID(systemID, database string) string {
	return systemID + "/" + database
}

func privilegeResourceID(parentID, grantee, privilegeType string) string {
	return parentID + "/priv/" + grantee + "/" + privilegeType
}

// --- password classification ------------------------------------------------

// classifyPassword maps a pg_authid.rolpassword value to how the password is
// stored. A nil pointer (unreadable or no password) yields "none".
func classifyPassword(rolpassword *string) string {
	if rolpassword == nil || *rolpassword == "" {
		return "none"
	}
	if strings.HasPrefix(*rolpassword, "SCRAM-SHA-256$") {
		return "scram-sha-256"
	}
	if strings.HasPrefix(*rolpassword, "md5") {
		return "md5"
	}
	return "other"
}

// --- privileges (ACL) -------------------------------------------------------

func newPostgresPrivilege(runtime *plugin.Runtime, parentID, grantee, privilegeType string, grantable bool) (*mqlPostgresPrivilege, error) {
	res, err := CreateResource(runtime, "postgres.privilege", map[string]*llx.RawData{
		"__id":          llx.StringData(privilegeResourceID(parentID, grantee, privilegeType)),
		"grantee":       llx.StringData(grantee),
		"privilegeType": llx.StringData(privilegeType),
		"isGrantable":   llx.BoolData(grantable),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlPostgresPrivilege), nil
}

// aclPrivileges runs an aclexplode-based query that must return
// (grantee text, privilege_type text, is_grantable bool) and builds the
// corresponding privilege resources.
func aclPrivileges(runtime *plugin.Runtime, pool *pgxpool.Pool, parentID, query string, args ...any) ([]any, error) {
	rows, err := pool.Query(pgContext(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []any{}
	for rows.Next() {
		var grantee, privilegeType string
		var grantable bool
		if err := rows.Scan(&grantee, &privilegeType, &grantable); err != nil {
			return nil, err
		}
		p, err := newPostgresPrivilege(runtime, parentID, grantee, privilegeType, grantable)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// --- typed role references --------------------------------------------------

// resolveRoleRef resolves a role by name for a typed owner/member reference,
// setting the field null when the name is empty.
func resolveRoleRef(runtime *plugin.Runtime, name string, field *plugin.TValue[*mqlPostgresRole]) (*mqlPostgresRole, error) {
	if name == "" {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(runtime, "postgres.role", map[string]*llx.RawData{
		"name": llx.StringData(name),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlPostgresRole), nil
}

// intToStr is a tiny helper for building ids from oids.
func intToStr(i int64) string {
	return strconv.FormatInt(i, 10)
}
