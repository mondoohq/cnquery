// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/clickhousedb/connection"
)

func (r *mqlClickhousedb) id() (string, error) {
	return "clickhousedb", nil
}

func clickhousedbConnection(runtime *plugin.Runtime) *connection.ClickhousedbConnection {
	return runtime.Connection.(*connection.ClickhousedbConnection)
}

// toAnySlice converts a string slice to []any for llx.
func toAnySlice(in []string) []any {
	out := make([]any, 0, len(in))
	for _, s := range in {
		out = append(out, s)
	}
	return out
}
