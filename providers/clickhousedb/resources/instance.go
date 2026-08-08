// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"database/sql"
	"fmt"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/clickhousedb/connection"
)

func initClickhousedbInstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	conn := clickhousedbConnection(runtime)
	db, err := conn.Client()
	if err != nil {
		return nil, nil, err
	}

	var version string
	if err := db.QueryRowContext(conn.Context(), `SELECT version()`).Scan(&version); err != nil {
		return nil, nil, fmt.Errorf("clickhousedb: cannot read version: %w", err)
	}

	args["__id"] = llx.StringData(connection.NewClickhousedbInstanceIdentifier(conn.ServerID()))
	args["version"] = llx.StringData(version)
	return args, nil, nil
}

func (r *mqlClickhousedbInstance) roles() ([]any, error) {
	conn := clickhousedbConnection(r.MqlRuntime)
	db, err := conn.Client()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(conn.Context(),
		`SELECT name, storage FROM system.roles ORDER BY name`)
	if err != nil {
		if connection.IsPermissionError(err) {
			return []any{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	serverID := conn.ServerID()
	list := []any{}
	for rows.Next() {
		var name, storage string
		if err := rows.Scan(&name, &storage); err != nil {
			return nil, err
		}
		res, err := CreateResource(r.MqlRuntime, "clickhousedb.role", map[string]*llx.RawData{
			"__id":    llx.StringData(serverID + "/role/" + name),
			"name":    llx.StringData(name),
			"storage": llx.StringData(storage),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, rows.Err()
}

func (r *mqlClickhousedbInstance) settingsProfiles() ([]any, error) {
	conn := clickhousedbConnection(r.MqlRuntime)
	db, err := conn.Client()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(conn.Context(),
		`SELECT name, storage, num_elements, apply_to_all, apply_to_list
		 FROM system.settings_profiles ORDER BY name`)
	if err != nil {
		if connection.IsPermissionError(err) {
			return []any{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	serverID := conn.ServerID()
	list := []any{}
	for rows.Next() {
		var name, storage string
		var numElements uint64
		var applyToAll bool
		var applyToList []string
		if err := rows.Scan(&name, &storage, &numElements, &applyToAll, &applyToList); err != nil {
			return nil, err
		}
		res, err := CreateResource(r.MqlRuntime, "clickhousedb.settingsProfile", map[string]*llx.RawData{
			"__id":         llx.StringData(serverID + "/settingsProfile/" + name),
			"name":         llx.StringData(name),
			"storage":      llx.StringData(storage),
			"numElements":  llx.IntData(int64(numElements)),
			"appliesToAll": llx.BoolData(applyToAll),
			"appliesTo":    llx.ArrayData(toAnySlice(applyToList), "string"),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, rows.Err()
}

func (r *mqlClickhousedbInstance) quotas() ([]any, error) {
	conn := clickhousedbConnection(r.MqlRuntime)
	db, err := conn.Client()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(conn.Context(),
		`SELECT name, storage, keys, apply_to_all, apply_to_list
		 FROM system.quotas ORDER BY name`)
	if err != nil {
		if connection.IsPermissionError(err) {
			return []any{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	serverID := conn.ServerID()
	list := []any{}
	for rows.Next() {
		var name, storage string
		var keys, applyToList []string
		var applyToAll bool
		if err := rows.Scan(&name, &storage, &keys, &applyToAll, &applyToList); err != nil {
			return nil, err
		}
		res, err := CreateResource(r.MqlRuntime, "clickhousedb.quota", map[string]*llx.RawData{
			"__id":         llx.StringData(serverID + "/quota/" + name),
			"name":         llx.StringData(name),
			"storage":      llx.StringData(storage),
			"keys":         llx.ArrayData(toAnySlice(keys), "string"),
			"appliesToAll": llx.BoolData(applyToAll),
			"appliesTo":    llx.ArrayData(toAnySlice(applyToList), "string"),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, rows.Err()
}

func (r *mqlClickhousedbInstance) clusters() ([]any, error) {
	conn := clickhousedbConnection(r.MqlRuntime)
	db, err := conn.Client()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(conn.Context(),
		`SELECT cluster, countDistinct(shard_num) AS shards, max(replica_num) AS replicas
		 FROM system.clusters GROUP BY cluster ORDER BY cluster`)
	if err != nil {
		if connection.IsPermissionError(err) {
			return []any{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	serverID := conn.ServerID()
	list := []any{}
	for rows.Next() {
		var name string
		var shards, replicas uint64
		if err := rows.Scan(&name, &shards, &replicas); err != nil {
			return nil, err
		}
		res, err := CreateResource(r.MqlRuntime, "clickhousedb.cluster", map[string]*llx.RawData{
			"__id":            llx.StringData(serverID + "/cluster/" + name),
			"name":            llx.StringData(name),
			"shardCount":      llx.IntData(int64(shards)),
			"maxReplicaCount": llx.IntData(int64(replicas)),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, rows.Err()
}

func (r *mqlClickhousedbInstance) serverSettings() ([]any, error) {
	conn := clickhousedbConnection(r.MqlRuntime)
	db, err := conn.Client()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(conn.Context(),
		// `default` is a SQL reserved word; backtick-quote it so the query stays
		// valid across ClickHouse versions.
		"SELECT name, value, `default`, changed, description"+
			" FROM system.server_settings ORDER BY name")
	if err != nil {
		if connection.IsPermissionError(err) {
			return []any{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	serverID := conn.ServerID()
	list := []any{}
	for rows.Next() {
		var name, value, def, description string
		var changed bool
		if err := rows.Scan(&name, &value, &def, &changed, &description); err != nil {
			return nil, err
		}
		res, err := CreateResource(r.MqlRuntime, "clickhousedb.serverSetting", map[string]*llx.RawData{
			"__id":        llx.StringData(serverID + "/serverSetting/" + name),
			"name":        llx.StringData(name),
			"value":       llx.StringData(value),
			"default":     llx.StringData(def),
			"changed":     llx.BoolData(changed),
			"description": llx.StringData(description),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, rows.Err()
}

// grantsFor renders the privileges granted to a user or role from system.grants
// into readable "<privilege> ON <scope>" strings. The grantee `name` is bound as
// a query parameter; `column` is concatenated into the SQL, so it must be a
// trusted literal column name ("user_name" or "role_name") and never user input.
func grantsFor(db *sql.DB, ctx context.Context, column, name string) ([]any, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT access_type, database, table, column, is_partial_revoke, grant_option
		 FROM system.grants WHERE `+column+` = ? ORDER BY access_type`, name)
	if err != nil {
		if connection.IsPermissionError(err) {
			return []any{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var accessType string
		var database, table, col sql.NullString
		var partialRevoke, grantOption bool
		if err := rows.Scan(&accessType, &database, &table, &col, &partialRevoke, &grantOption); err != nil {
			return nil, err
		}
		scope := grantScope(database, table, col)
		line := accessType + " ON " + scope
		if grantOption {
			line += " WITH GRANT OPTION"
		}
		if partialRevoke {
			line = "REVOKE " + line
		}
		out = append(out, line)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return toAnySlice(out), nil
}

// grantScope formats the database/table/column of a grant, using "*" for the
// wildcard (NULL) parts.
func grantScope(database, table, column sql.NullString) string {
	db := "*"
	if database.Valid && database.String != "" {
		db = database.String
	}
	tbl := "*"
	if table.Valid && table.String != "" {
		tbl = table.String
	}
	scope := db + "." + tbl
	if column.Valid && column.String != "" {
		scope += "(" + column.String + ")"
	}
	return scope
}
