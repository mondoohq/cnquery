// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"slices"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/clickhousedb/connection"
)

func (r *mqlClickhousedbInstance) users() ([]any, error) {
	conn := clickhousedbConnection(r.MqlRuntime)
	db, err := conn.Client()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(conn.Context(),
		`SELECT name, auth_type, storage, host_ip, host_names, default_roles_list
		 FROM system.users ORDER BY name`)
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
		var hostIps, hostNames, defaultRoles []string
		// auth_type is taken as-is because its arity changed between releases;
		// see stringList.
		var authType any
		if err := rows.Scan(&name, &authType, &storage, &hostIps, &hostNames, &defaultRoles); err != nil {
			return nil, err
		}
		authTypes, err := stringList("auth_type", authType)
		if err != nil {
			return nil, err
		}
		res, err := CreateResource(r.MqlRuntime, "clickhousedb.user", map[string]*llx.RawData{
			"__id":         llx.StringData(serverID + "/user/" + name),
			"name":         llx.StringData(name),
			"authTypes":    llx.ArrayData(toAnySlice(authTypes), "string"),
			"hasPassword":  llx.BoolData(requiresCredential(authTypes)),
			"anyHost":      llx.BoolData(allowsAnyHost(hostIps)),
			"storage":      llx.StringData(storage),
			"hostIps":      llx.ArrayData(toAnySlice(hostIps), "string"),
			"hostNames":    llx.ArrayData(toAnySlice(hostNames), "string"),
			"defaultRoles": llx.ArrayData(toAnySlice(defaultRoles), "string"),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, rows.Err()
}

// grants renders the privileges granted directly to the user.
func (r *mqlClickhousedbUser) grants() ([]any, error) {
	conn := clickhousedbConnection(r.MqlRuntime)
	db, err := conn.Client()
	if err != nil {
		return nil, err
	}
	return grantsFor(db, conn.Context(), "user_name", r.Name.Data)
}

// grants renders the privileges granted to the role.
func (r *mqlClickhousedbRole) grants() ([]any, error) {
	conn := clickhousedbConnection(r.MqlRuntime)
	db, err := conn.Client()
	if err != nil {
		return nil, err
	}
	return grantsFor(db, conn.Context(), "role_name", r.Name.Data)
}

// requiresCredential reports whether a user needs a credential to authenticate.
// ClickHouse admits a login if any authentication method matches, so a single
// "no_password" method makes the account reachable without a credential even
// when it also has a real one; that is treated as password-less on purpose.
func requiresCredential(authTypes []string) bool {
	// An empty method list is anomalous — ClickHouse always defines at least one
	// method. Treat it defensively as requiring a credential rather than emitting
	// a password-less finding on incomplete data.
	if len(authTypes) == 0 {
		return true
	}
	return !slices.Contains(authTypes, "no_password")
}

// allowsAnyHost reports whether the user's host restriction permits any address.
func allowsAnyHost(hostIps []string) bool {
	for _, ip := range hostIps {
		if ip == "::/0" || ip == "0.0.0.0/0" {
			return true
		}
	}
	return false
}
