// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/clickhousedb/connection"
)

func (r *mqlClickhousedb) id() (string, error) {
	return "clickhousedb", nil
}

func clickhousedbConnection(runtime *plugin.Runtime) *connection.ClickhousedbConnection {
	return runtime.Connection.(*connection.ClickhousedbConnection)
}

// toAnySlice converts a string slice to []any for llx.
// stringList normalizes a column whose arity changed between ClickHouse
// releases into the list the resources expose.
//
// system.users.auth_type is the case this exists for: a scalar Enum8 up to and
// including the 24.8 LTS line, and an Array(Enum8) from 25.x, where an account
// may carry several authentication methods. Scanning it into a fixed Go type
// fails on one line or the other -- against 24.8 the driver reports
//
//	sql: Scan error on column index 1, name "auth_type": unsupported Scan,
//	storing driver.Value type string into type *[]string
//
// and every check reading clickhousedb.instance.users errors rather than
// reaching a verdict. Scanning into any and flattening here keeps one code path
// across both schemas without a server-version check.
//
// An unrecognised shape is an error rather than an empty list. Callers read
// these lists to decide whether an account is reachable without a credential,
// and requiresCredential treats an empty list as requiring one -- so swallowing
// a shape we do not understand would report a password-less account as secure.
// Failing here surfaces the new shape instead.
func stringList(column string, v any) ([]string, error) {
	switch t := v.(type) {
	case nil:
		return nil, nil
	case []string:
		return t, nil
	case string:
		// A scalar column. An empty value is no entry rather than one
		// nameless entry, so it flattens to an empty list.
		if t == "" {
			return nil, nil
		}
		return []string{t}, nil
	case *string:
		if t == nil {
			return nil, nil
		}
		return stringList(column, *t)
	case []any:
		out := make([]string, 0, len(t))
		for i, item := range t {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("column %s: element %d is %T, expected a string", column, i, item)
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("column %s: unexpected type %T; the server schema may have changed", column, v)
	}
}

func toAnySlice(in []string) []any {
	out := make([]any, 0, len(in))
	for _, s := range in {
		out = append(out, s)
	}
	return out
}
