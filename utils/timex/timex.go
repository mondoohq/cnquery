// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package timex

import (
	"errors"
	"time"
)

var timeFormats = map[string]string{
	"ansic":    time.ANSIC,
	"rfc822":   time.RFC822,
	"rfc822z":  time.RFC822Z,
	"rfc850":   time.RFC850,
	"rfc1123":  time.RFC1123,
	"rfc1123z": time.RFC1123Z,
	"rfc3339":  time.RFC3339,
	"kitchen":  time.Kitchen,
	"stamp":    time.Stamp,
	"datetime": time.DateTime,
	"date":     time.DateOnly,
	"time":     time.TimeOnly,
}

// Note: the list of recognized timeFormats is mutually exclusive.
// This means that for any given timestamp for one format it won't
// parse with any of the other formats. Should this ever change,
// the order in which formats are parsed will play a more important role.
var defaultTimeFormatsOrder = []string{
	time.RFC3339,
	time.DateTime,
	time.DateOnly,
	time.TimeOnly,
	time.RFC1123,
	time.RFC1123Z,
	time.ANSIC,
	time.RFC822,
	time.RFC822Z,
	time.RFC850,
	time.Kitchen,
	time.Stamp,
}

// Parse a date and/or time string into a unix timestamp.
//
// The format is optional. We will try most common formats one
// by one if the format is left empty. This is slower but more
// convenient when dealing with human input.
//
// If the format is provided, it must be one of the formats
// defined in the time package or a custom format
// (see the time package documentation).
//
// If the format is not recognized, an error is returned.
// If the format is recognized, the parsed time is returned.
func Parse(s string, format string) (time.Time, error) {
	if format != "" {
		if f, ok := timeFormats[format]; ok {
			format = f
		}
	}

	if format != "" {
		parsed, err := time.Parse(format, s)
		if err != nil {
			return time.Time{}, err
		}
		return parsed, nil
	}

	// Note: Yes, this approach is much slower than giving us a hint
	// about which time format is used.
	for _, format := range defaultTimeFormatsOrder {
		parsed, err := time.Parse(format, s)
		if err != nil {
			continue
		}
		return parsed, nil
	}
	return time.Time{}, errors.New("no supported date/time format found")
}
