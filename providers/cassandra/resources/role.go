// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sort"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/cassandra/connection"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlCassandraCluster) roles() ([]any, error) {
	conn := cassandraConnection(r.MqlRuntime)
	session, err := conn.Session()
	if err != nil {
		return nil, err
	}

	type roleRow struct {
		name        string
		canLogin    bool
		isSuperuser bool
		memberOf    []string
		hasPassword bool
	}
	var rows []roleRow
	iter := session.Query(`SELECT role, can_login, is_superuser, member_of, salted_hash FROM system_auth.roles`).Iter()
	var name string
	var canLogin, isSuperuser bool
	var memberOf []string
	var saltedHash string
	for iter.Scan(&name, &canLogin, &isSuperuser, &memberOf, &saltedHash) {
		mo := make([]string, len(memberOf))
		copy(mo, memberOf)
		rows = append(rows, roleRow{
			name:        name,
			canLogin:    canLogin,
			isSuperuser: isSuperuser,
			memberOf:    mo,
			// salted_hash is the bcrypt hash; expose only its presence, never the value.
			hasPassword: saltedHash != "",
		})
	}
	if err := iter.Close(); err != nil {
		// Reading all roles needs a privilege on system_auth (or superuser);
		// treat a denial as no visible roles rather than failing the asset. On
		// an AllowAll cluster the roles table is simply empty.
		if connection.IsUnauthorized(err) {
			return []any{}, nil
		}
		return nil, err
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })

	serverID := r.__id
	list := []any{}
	for _, row := range rows {
		res, err := CreateResource(r.MqlRuntime, "cassandra.role", map[string]*llx.RawData{
			"__id":        llx.StringData(serverID + "/role/" + row.name),
			"name":        llx.StringData(row.name),
			"canLogin":    llx.BoolData(row.canLogin),
			"isSuperuser": llx.BoolData(row.isSuperuser),
			"hasPassword": llx.BoolData(row.hasPassword),
			"memberOf":    llx.ArrayData(toAnySlice(row.memberOf), types.String),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, nil
}

func (r *mqlCassandraRole) permissions() ([]any, error) {
	conn := cassandraConnection(r.MqlRuntime)
	session, err := conn.Session()
	if err != nil {
		return nil, err
	}

	type permRow struct {
		resource    string
		permissions []string
	}
	var rows []permRow
	iter := session.Query(`SELECT resource, permissions FROM system_auth.role_permissions WHERE role = ?`, r.Name.Data).Iter()
	var resource string
	var perms []string
	for iter.Scan(&resource, &perms) {
		p := make([]string, len(perms))
		copy(p, perms)
		sort.Strings(p)
		rows = append(rows, permRow{resource: resource, permissions: p})
	}
	if err := iter.Close(); err != nil {
		if connection.IsUnauthorized(err) {
			return []any{}, nil
		}
		return nil, err
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].resource < rows[j].resource })

	list := []any{}
	for _, row := range rows {
		// resource is unique per role (role_permissions PK is (role, resource)),
		// so it makes a stable id regardless of the order rows are returned in.
		res, err := CreateResource(r.MqlRuntime, "cassandra.role.permission", map[string]*llx.RawData{
			"__id":        llx.StringData(r.__id + "/perm/" + row.resource),
			"resource":    llx.StringData(row.resource),
			"permissions": llx.ArrayData(toAnySlice(row.permissions), types.String),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, nil
}
