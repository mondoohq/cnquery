// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
//
// Note for unit tests: We are currently using `mql_test.go` in core tests
// as a proxy to test the stringify approach. Once we migrate all MQL
// string generation to the below functions, we should migrate and expand
// on testing in a more unified way.

package llx

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mondoo.com/cnquery/v12/types"
)

// StringifyValue turns a raw golang value into a string
func StringifyValue(value any, typHint types.Type) string {
	switch v := value.(type) {
	case string:
		switch typHint {
		case types.Regex:
			// FIXME: doesn't properly escape the contents of the regex
			var res strings.Builder
			res.WriteByte('/')
			res.WriteString(v)
			res.WriteByte('/')
			return res.String()
		case types.Version:
			return "v" + v
		default:
			return v
		}
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case nil:
		return "null"
	case *time.Time:
		if v.Equal(NeverPastTime) || v.Equal(NeverFutureTime) {
			return "Never"
		}

		if v.Unix() > 0 {
			return v.String()
		}

		return TimeToDurationString(*v)

	case RawIP:
		return v.String()

	default:
		return fmt.Sprintf("%#v", value)
	}
}

func TimeToDurationString(t time.Time) string {
	seconds := TimeToDuration(&t)
	minutes := seconds / 60
	hours := minutes / 60
	days := hours / 24

	var res strings.Builder
	if days > 0 {
		res.WriteString(fmt.Sprintf("%d days ", days))
	}
	if hours%24 != 0 {
		res.WriteString(fmt.Sprintf("%d hours ", hours%24))
	}
	if minutes%60 != 0 {
		res.WriteString(fmt.Sprintf("%d minutes ", minutes%60))
	}
	// if we haven't printed any of the other pieces (days/hours/minutes) then print this
	// if we have, then check if this is non-zero
	if minutes == 0 || seconds%60 != 0 {
		res.WriteString(fmt.Sprintf("%d seconds", seconds%60))
	}

	return strings.TrimSpace(res.String())
}
