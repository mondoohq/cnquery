// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/cassandra/connection"
)

func (r *mqlCassandra) id() (string, error) {
	return "cassandra", nil
}

func intToStr(i int) string {
	return strconv.Itoa(i)
}

func cassandraConnection(runtime *plugin.Runtime) *connection.CassandraConnection {
	return runtime.Connection.(*connection.CassandraConnection)
}

// toAnySlice converts a string slice to []any for llx.
func toAnySlice(in []string) []any {
	out := make([]any, 0, len(in))
	for _, s := range in {
		out = append(out, s)
	}
	return out
}
