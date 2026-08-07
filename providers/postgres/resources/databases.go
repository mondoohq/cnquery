// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

const databaseColumns = `d.datname, d.oid::bigint, COALESCE(o.rolname, ''),
	pg_encoding_to_char(d.encoding), d.datcollate, d.datctype,
	d.datistemplate, d.datallowconn, d.datconnlimit`

func newPostgresDatabase(runtime *plugin.Runtime, systemID string, pool *pgxpool.Pool, name string) (*mqlPostgresDatabase, error) {
	rows, err := pool.Query(pgContext(),
		"SELECT "+databaseColumns+" FROM pg_database d LEFT JOIN pg_roles o ON d.datdba = o.oid WHERE d.datname = $1", name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, errors.New("postgres.database " + name + " not found")
	}
	return scanDatabase(runtime, systemID, rows)
}

func scanDatabase(runtime *plugin.Runtime, systemID string, rows interface {
	Scan(...any) error
}) (*mqlPostgresDatabase, error) {
	var datname, ownerName, encoding, collate, ctype string
	var oid, connLimit int64
	var isTemplate, allowConn bool
	if err := rows.Scan(&datname, &oid, &ownerName, &encoding, &collate, &ctype,
		&isTemplate, &allowConn, &connLimit); err != nil {
		return nil, err
	}
	res, err := CreateResource(runtime, "postgres.database", map[string]*llx.RawData{
		"__id":             llx.StringData(databaseResourceID(systemID, datname)),
		"name":             llx.StringData(datname),
		"oid":              llx.IntData(oid),
		"encoding":         llx.StringData(encoding),
		"collate":          llx.StringData(collate),
		"ctype":            llx.StringData(ctype),
		"isTemplate":       llx.BoolData(isTemplate),
		"allowConnections": llx.BoolData(allowConn),
		"connectionLimit":  llx.IntData(connLimit),
	})
	if err != nil {
		return nil, err
	}
	db := res.(*mqlPostgresDatabase)
	db.cacheOwner = ownerName
	return db, nil
}

func initPostgresDatabase(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	conn := pgConnection(runtime)
	if _, ok := args["name"]; !ok && conn.ScopedDatabase() != "" {
		args["name"] = llx.StringData(conn.ScopedDatabase())
	}
	nameRaw, ok := args["name"]
	if !ok {
		return args, nil, nil
	}
	name, _ := nameRaw.Value.(string)
	if name == "" {
		return nil, nil, errors.New("postgres.database requires a non-empty name")
	}

	systemID, err := conn.SystemID()
	if err != nil {
		return nil, nil, err
	}
	pool, err := conn.Client("")
	if err != nil {
		return nil, nil, err
	}
	res, err := newPostgresDatabase(runtime, systemID, pool, name)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func (r *mqlPostgresInstance) databases() ([]any, error) {
	pool, err := pgPool(r.MqlRuntime, "")
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(pgContext(),
		"SELECT "+databaseColumns+" FROM pg_database d LEFT JOIN pg_roles o ON d.datdba = o.oid ORDER BY d.datname")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []any{}
	for rows.Next() {
		db, err := scanDatabase(r.MqlRuntime, r.SystemIdentifier.Data, rows)
		if err != nil {
			return nil, err
		}
		list = append(list, db)
	}
	return list, rows.Err()
}

func (r *mqlPostgresDatabase) owner() (*mqlPostgresRole, error) {
	return resolveRoleRef(r.MqlRuntime, r.cacheOwner, &r.Owner)
}

func (r *mqlPostgresDatabase) privileges() ([]any, error) {
	// datacl lives in the cluster-global pg_database, so use the server pool.
	pool, err := pgPool(r.MqlRuntime, "")
	if err != nil {
		return nil, err
	}
	return aclPrivileges(r.MqlRuntime, pool, r.__id,
		`SELECT COALESCE(gr.rolname, 'PUBLIC'), a.privilege_type, a.is_grantable
		 FROM pg_database d, aclexplode(d.datacl) a
		 LEFT JOIN pg_roles gr ON gr.oid = a.grantee
		 WHERE d.datname = $1`, r.Name.Data)
}

func (r *mqlPostgresDatabase) schemas() ([]any, error) {
	// Schemas are per-database, so connect to this database.
	if !r.AllowConnections.Data {
		return []any{}, nil
	}
	pool, err := pgPool(r.MqlRuntime, r.Name.Data)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(pgContext(),
		`SELECT n.nspname, n.oid::bigint, COALESCE(o.rolname, '')
		 FROM pg_namespace n LEFT JOIN pg_roles o ON n.nspowner = o.oid
		 WHERE n.nspname NOT LIKE 'pg_temp_%' AND n.nspname NOT LIKE 'pg_toast%'
		 ORDER BY n.nspname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []any{}
	for rows.Next() {
		var name, ownerName string
		var oid int64
		if err := rows.Scan(&name, &oid, &ownerName); err != nil {
			return nil, err
		}
		res, err := CreateResource(r.MqlRuntime, "postgres.schema", map[string]*llx.RawData{
			"__id": llx.StringData(r.__id + "/schema/" + name),
			"name": llx.StringData(name),
			"oid":  llx.IntData(oid),
		})
		if err != nil {
			return nil, err
		}
		schema := res.(*mqlPostgresSchema)
		schema.cacheDatabase = r.Name.Data
		schema.cacheOwner = ownerName
		list = append(list, schema)
	}
	return list, rows.Err()
}

func (r *mqlPostgresDatabase) functions() ([]any, error) {
	if !r.AllowConnections.Data {
		return []any{}, nil
	}
	pool, err := pgPool(r.MqlRuntime, r.Name.Data)
	if err != nil {
		return nil, err
	}
	// Exclude built-in system functions to keep the result to user objects.
	rows, err := pool.Query(pgContext(),
		`SELECT p.proname, p.oid::bigint, n.nspname, l.lanname, p.prosecdef, COALESCE(o.rolname, '')
		 FROM pg_proc p
		 JOIN pg_namespace n ON p.pronamespace = n.oid
		 JOIN pg_language l ON p.prolang = l.oid
		 LEFT JOIN pg_roles o ON p.proowner = o.oid
		 WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
		 ORDER BY n.nspname, p.proname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []any{}
	for rows.Next() {
		var name, schema, language, ownerName string
		var oid int64
		var securityDefiner bool
		if err := rows.Scan(&name, &oid, &schema, &language, &securityDefiner, &ownerName); err != nil {
			return nil, err
		}
		res, err := CreateResource(r.MqlRuntime, "postgres.function", map[string]*llx.RawData{
			"__id":              llx.StringData(r.__id + "/function/" + schema + "/" + name + "/" + intToStr(oid)),
			"name":              llx.StringData(name),
			"oid":               llx.IntData(oid),
			"schema":            llx.StringData(schema),
			"language":          llx.StringData(language),
			"isSecurityDefiner": llx.BoolData(securityDefiner),
		})
		if err != nil {
			return nil, err
		}
		fn := res.(*mqlPostgresFunction)
		fn.cacheDatabase = r.Name.Data
		fn.cacheOwner = ownerName
		list = append(list, fn)
	}
	return list, rows.Err()
}

func (r *mqlPostgresDatabase) extensions() ([]any, error) {
	if !r.AllowConnections.Data {
		return []any{}, nil
	}
	pool, err := pgPool(r.MqlRuntime, r.Name.Data)
	if err != nil {
		return nil, err
	}
	return listExtensions(r.MqlRuntime, pool, r.__id)
}

// --- schema -----------------------------------------------------------------

func (r *mqlPostgresSchema) owner() (*mqlPostgresRole, error) {
	return resolveRoleRef(r.MqlRuntime, r.cacheOwner, &r.Owner)
}

func (r *mqlPostgresSchema) privileges() ([]any, error) {
	pool, err := pgPool(r.MqlRuntime, r.cacheDatabase)
	if err != nil {
		return nil, err
	}
	return aclPrivileges(r.MqlRuntime, pool, r.__id,
		`SELECT COALESCE(gr.rolname, 'PUBLIC'), a.privilege_type, a.is_grantable
		 FROM pg_namespace n, aclexplode(n.nspacl) a
		 LEFT JOIN pg_roles gr ON gr.oid = a.grantee
		 WHERE n.nspname = $1`, r.Name.Data)
}

// --- function ---------------------------------------------------------------

func (r *mqlPostgresFunction) owner() (*mqlPostgresRole, error) {
	return resolveRoleRef(r.MqlRuntime, r.cacheOwner, &r.Owner)
}

func (r *mqlPostgresFunction) privileges() ([]any, error) {
	pool, err := pgPool(r.MqlRuntime, r.cacheDatabase)
	if err != nil {
		return nil, err
	}
	return aclPrivileges(r.MqlRuntime, pool, r.__id,
		`SELECT COALESCE(gr.rolname, 'PUBLIC'), a.privilege_type, a.is_grantable
		 FROM pg_proc p, aclexplode(p.proacl) a
		 LEFT JOIN pg_roles gr ON gr.oid = a.grantee
		 WHERE p.oid = $1::oid`, r.Oid.Data)
}

// --- shared extension listing -----------------------------------------------

func listExtensions(runtime *plugin.Runtime, pool *pgxpool.Pool, scopeID string) ([]any, error) {
	rows, err := pool.Query(pgContext(),
		`SELECT e.extname, COALESCE(e.extversion, ''), COALESCE(n.nspname, ''), COALESCE(o.rolname, '')
		 FROM pg_extension e
		 LEFT JOIN pg_namespace n ON e.extnamespace = n.oid
		 LEFT JOIN pg_roles o ON e.extowner = o.oid
		 ORDER BY e.extname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []any{}
	for rows.Next() {
		var name, version, schema, ownerName string
		if err := rows.Scan(&name, &version, &schema, &ownerName); err != nil {
			return nil, err
		}
		res, err := CreateResource(runtime, "postgres.extension", map[string]*llx.RawData{
			"__id":    llx.StringData(scopeID + "/extension/" + name),
			"name":    llx.StringData(name),
			"version": llx.StringData(version),
			"schema":  llx.StringData(schema),
		})
		if err != nil {
			return nil, err
		}
		ext := res.(*mqlPostgresExtension)
		ext.cacheOwner = ownerName
		list = append(list, ext)
	}
	return list, rows.Err()
}

func (r *mqlPostgresExtension) owner() (*mqlPostgresRole, error) {
	return resolveRoleRef(r.MqlRuntime, r.cacheOwner, &r.Owner)
}
