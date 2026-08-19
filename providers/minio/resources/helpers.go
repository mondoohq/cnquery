// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

// timeData renders a timestamp for the schema, reporting null rather than the
// zero time when the deployment has none. MinIO answers with 0001-01-01
// for a record it never stamped, and passing that through would report the
// first day of year 1 as a real date, which reads as a genuine value in every
// comparison an audit makes.
func timeData(t time.Time) *llx.RawData {
	if t.IsZero() {
		return llx.NilData
	}
	return llx.TimeData(t)
}

// timePtrData is timeData for an optional timestamp.
func timePtrData(t *time.Time) *llx.RawData {
	if t == nil {
		return llx.NilData
	}
	return timeData(*t)
}

// strSliceData converts a string slice for the schema. A nil slice becomes an
// empty list rather than null, because "no entries" is a real answer for every
// list this provider returns.
func strSliceData(values []string) *llx.RawData {
	out := make([]any, 0, len(values))
	for _, v := range values {
		out = append(out, v)
	}
	return llx.ArrayData(out, types.String)
}
