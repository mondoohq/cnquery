// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"strings"

	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/redisdb/connection"
)

func (r *mqlRedisdb) id() (string, error) {
	return "redisdb", nil
}

func redisdbConnection(runtime *plugin.Runtime) *connection.RedisdbConnection {
	return runtime.Connection.(*connection.RedisdbConnection)
}

// isNoPerm reports whether an error is a Redis access-control denial. These are
// treated as "not visible" for privilege-gated fetches; other errors propagate.
func isNoPerm(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "NOPERM") || strings.Contains(msg, "WRONGPASS")
}

func atoiOr(s string, fallback int64) int64 {
	if s == "" {
		return fallback
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fallback
	}
	return v
}
