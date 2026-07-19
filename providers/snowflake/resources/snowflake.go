// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strings"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/snowflake/connection"
)

func (r *mqlSnowflake) id() (string, error) {
	return "snowflake", nil
}

// snowflakeTimeLayouts are the timestamp formats Snowflake renders in the
// string CreatedOn fields that some SDK types leave unparsed (e.g. row-access
// policies, managed accounts). SHOW output uses a space-separated timestamp
// with a numeric zone offset; we also accept RFC3339 as a fallback.
var snowflakeTimeLayouts = []string{
	time.RFC3339,
	"2006-01-02 15:04:05.999999999 -0700",
	"2006-01-02 15:04:05.999 -0700",
	"2006-01-02 15:04:05 -0700",
}

// parseSnowflakeTime converts a Snowflake SHOW timestamp string into time
// RawData, trying each known layout. It returns null RawData for an empty
// string or an unrecognized format so the field resolves to null rather than
// erroring the whole query.
func parseSnowflakeTime(value string) *llx.RawData {
	value = strings.TrimSpace(value)
	if value == "" {
		return llx.NilData
	}
	for _, layout := range snowflakeTimeLayouts {
		if t, err := time.Parse(layout, value); err == nil {
			return llx.TimeData(t)
		}
	}
	return llx.NilData
}

// splitCommaList parses a comma-separated value from Snowflake DESCRIBE output
// (e.g. STORAGE_ALLOWED_LOCATIONS) into a slice. Snowflake sometimes wraps the
// list in square brackets, so those are stripped first. Empty entries are
// dropped so a trailing comma or an empty value yields an empty slice.
func splitCommaList(value string) []any {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	if value == "" {
		return []any{}
	}
	parts := strings.Split(value, ",")
	out := make([]any, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func (r *mqlSnowflake) currentRole() (string, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	current, err := client.ContextFunctions.CurrentSessionDetails(ctx)
	if err != nil {
		return "", err
	}
	return current.Role, nil
}
