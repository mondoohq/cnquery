// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/opensearch/connection"
)

func (r *mqlOpensearch) id() (string, error) {
	return "opensearch", nil
}

func osConnection(runtime *plugin.Runtime) *connection.OpensearchConnection {
	return runtime.Connection.(*connection.OpensearchConnection)
}

// toStringSlice converts a decoded JSON string array to []any for llx.
func toStringSlice(in []string) []any {
	out := make([]any, 0, len(in))
	for _, s := range in {
		out = append(out, s)
	}
	return out
}
