// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

// MCP transport types. These mirror the values AI tools write into their MCP
// config files; when a config omits the type we infer it from the connection
// shape (see deriveMcpTransport).
const (
	mcpTransportStdio = "stdio"
	mcpTransportHTTP  = "http"
)

// deriveMcpTransport returns the transport type for an MCP server. An explicit
// value from the config wins; otherwise it is inferred from the connection
// shape: a local launch command implies stdio, a remote endpoint URL implies
// http. Returns an empty string when neither is present.
func deriveMcpTransport(explicitType, command, url string) string {
	if explicitType != "" {
		return explicitType
	}
	if command != "" {
		return mcpTransportStdio
	}
	if url != "" {
		return mcpTransportHTTP
	}
	return ""
}

// strSliceToArrayData converts a string slice into an llx string-array RawData,
// the shape every mcpServer resource uses for its args field.
func strSliceToArrayData(in []string) *llx.RawData {
	out := make([]interface{}, len(in))
	for i, v := range in {
		out[i] = v
	}
	return llx.ArrayData(out, types.String)
}
