// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

const roleColumns = `r.rolname, r.oid::bigint, r.rolsuper, r.rolcanlogin, r.rolcreaterole,
	r.rolcreatedb, r.rolreplication, r.rolbypassrls, r.rolinherit, r.rolconnlimit,
	r.rolvaliduntil, r.rolconfig`

// passwordTypesByOid best-effort reads pg_authid (superuser-only) and maps each
// role oid to how its password is stored. An empty map means the catalog was
// not readable, so passwordType is left null.
func passwordTypesByOid(pool *pgxpool.Pool) map[int64]string {
	out := map[int64]string{}
	rows, err := pool.Query(pgContext(), "SELECT oid::bigint, rolpassword FROM pg_authid")
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var oid int64
		var rolpassword *string
		if err := rows.Scan(&oid, &rolpassword); err != nil {
			return out
		}
		out[oid] = classifyPassword(rolpassword)
	}
	return out
}

// newPostgresRole builds a role from a row selected with roleColumns.
func newPostgresRole(runtime *plugin.Runtime, systemID string, rows pgx.Rows, passwordTypes map[int64]string) (*mqlPostgresRole, error) {
	var name string
	var oid, connLimit int64
	var super, canLogin, createRole, createDb, replication, bypassRLS, inherit bool
	var validUntil *time.Time
	var config []string
	if err := rows.Scan(&name, &oid, &super, &canLogin, &createRole, &createDb,
		&replication, &bypassRLS, &inherit, &connLimit, &validUntil, &config); err != nil {
		return nil, err
	}

	fields := map[string]*llx.RawData{
		"__id":               llx.StringData(roleResourceID(systemID, name)),
		"name":               llx.StringData(name),
		"oid":                llx.IntData(oid),
		"isSuperuser":        llx.BoolData(super),
		"canLogin":           llx.BoolData(canLogin),
		"createRole":         llx.BoolData(createRole),
		"createDb":           llx.BoolData(createDb),
		"isReplication":      llx.BoolData(replication),
		"bypassRLS":          llx.BoolData(bypassRLS),
		"inheritsPrivileges": llx.BoolData(inherit),
		"connectionLimit":    llx.IntData(connLimit),
		"validUntil":         llx.TimeDataPtr(validUntil),
		"config":             llx.ArrayData(strSliceToAny(config), types.String),
	}
	if pt, ok := passwordTypes[oid]; ok {
		fields["passwordType"] = llx.StringData(pt)
	}

	res, err := CreateResource(runtime, "postgres.role", fields)
	if err != nil {
		return nil, err
	}
	return res.(*mqlPostgresRole), nil
}

func initPostgresRole(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	nameRaw, ok := args["name"]
	if !ok {
		return args, nil, nil
	}
	name, _ := nameRaw.Value.(string)
	if name == "" {
		return nil, nil, errors.New("postgres.role requires a non-empty name")
	}

	systemID, err := pgSystemID(runtime)
	if err != nil {
		return nil, nil, err
	}
	pool, err := pgPool(runtime, "")
	if err != nil {
		return nil, nil, err
	}
	rows, err := pool.Query(pgContext(), "SELECT "+roleColumns+" FROM pg_roles r WHERE r.rolname = $1", name)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, nil, err
		}
		return nil, nil, errors.New("postgres.role " + name + " not found")
	}
	res, err := newPostgresRole(runtime, systemID, rows, passwordTypesByOid(pool))
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func (r *mqlPostgresInstance) roles() ([]any, error) {
	pool, err := pgPool(r.MqlRuntime, "")
	if err != nil {
		return nil, err
	}
	passwordTypes := passwordTypesByOid(pool)
	rows, err := pool.Query(pgContext(), "SELECT "+roleColumns+" FROM pg_roles r ORDER BY r.rolname")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []any{}
	for rows.Next() {
		role, err := newPostgresRole(r.MqlRuntime, r.SystemIdentifier.Data, rows, passwordTypes)
		if err != nil {
			return nil, err
		}
		list = append(list, role)
	}
	return list, rows.Err()
}

// rolesByQuery resolves each role name returned by a membership query into a
// full postgres.role via its init.
func rolesByQuery(runtime *plugin.Runtime, oid int64, query string) ([]any, error) {
	pool, err := pgPool(runtime, "")
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(pgContext(), query, oid)
	if err != nil {
		return nil, err
	}
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		names = append(names, name)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	list := []any{}
	for _, name := range names {
		res, err := NewResource(runtime, "postgres.role", map[string]*llx.RawData{"name": llx.StringData(name)})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, nil
}

func (r *mqlPostgresRole) memberOf() ([]any, error) {
	return rolesByQuery(r.MqlRuntime, r.Oid.Data,
		"SELECT g.rolname FROM pg_auth_members m JOIN pg_roles g ON m.roleid = g.oid WHERE m.member = $1::oid")
}

func (r *mqlPostgresRole) members() ([]any, error) {
	return rolesByQuery(r.MqlRuntime, r.Oid.Data,
		"SELECT mr.rolname FROM pg_auth_members m JOIN pg_roles mr ON m.member = mr.oid WHERE m.roleid = $1::oid")
}
