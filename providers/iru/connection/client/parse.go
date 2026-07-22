// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package client

import (
	"strconv"
	"strings"
	"time"
)

// timeLayouts covers every timestamp shape the Iru API is known to emit.
// The device detail endpoint is inconsistent: the listing and library
// endpoints use RFC3339 ("2026-07-22T18:07:24.422285Z"), while the device
// detail sections use a space separator with a numeric offset
// ("2026-02-18 14:42:04.395339+00:00") and installed profiles use a space
// separator with a spaced offset ("2026-02-18 14:42:41 +0000").
var timeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05.999999-07:00",
	"2006-01-02 15:04:05-07:00",
	"2006-01-02 15:04:05 -0700",
	"2006-01-02 15:04:05",
}

// ParseTime parses an Iru timestamp, returning nil on empty or unparseable
// input so callers can hand the result straight to llx.TimeDataPtr and get
// a null field rather than the zero time.
func ParseTime(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}

// ParseBool interprets the several truthy spellings the Iru API uses. The
// device detail endpoint returns booleans as the strings "True"/"False",
// and some fields use "Yes"/"No" or "enabled"/"disabled"; the JSON boolean
// endpoints decode straight to bool and never reach this helper.
func ParseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "yes", "enabled", "on", "1":
		return true
	default:
		return false
	}
}

// ParseInt parses an integer that the API sometimes serializes as a string
// (for example hardware core counts), returning 0 when the value is empty
// or non-numeric.
func ParseInt(s string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return n
}
