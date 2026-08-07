// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"time"

	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/elasticsearch/connection"
)

func intToStr(i int64) string {
	return strconv.FormatInt(i, 10)
}

func (r *mqlElasticsearch) id() (string, error) {
	return "elasticsearch", nil
}

func esConnection(runtime *plugin.Runtime) *connection.ElasticsearchConnection {
	return runtime.Connection.(*connection.ElasticsearchConnection)
}

// epochMillisToTime converts an Elasticsearch epoch-millisecond timestamp to a
// time.Time. A zero or negative value yields the zero time.
func epochMillisToTime(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

// toStringSlice converts a decoded JSON string array to []any for llx.
func toStringSlice(in []string) []any {
	out := make([]any, 0, len(in))
	for _, s := range in {
		out = append(out, s)
	}
	return out
}
