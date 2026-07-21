// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package shadow

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// shadowFieldCount is the number of colon-separated fields in an /etc/shadow
// entry: name:password:lastchange:min:max:warn:inactive:expire:reserved.
const shadowFieldCount = 9

type ShadowEntry struct {
	User         string
	Password     string
	LastChanged  *time.Time
	MinDays      string
	MaxDays      string
	WarnDays     string
	InactiveDays string
	ExpiryDates  string
	Reserved     string
}

func ParseShadow(r io.Reader) ([]ShadowEntry, error) {
	res := []ShadowEntry{}

	// /etc/shadow is a simple line-oriented, colon-separated file. We split on
	// ':' directly rather than using encoding/csv: csv treats `"` as a quote
	// character (corrupting password hashes that contain one) and locks the
	// field count to the first record (erroring the whole file on any skew).
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// At most shadowFieldCount fields so an unexpected ':' in the final
		// reserved field stays attached to it rather than overflowing.
		record := strings.SplitN(line, ":", shadowFieldCount)
		if len(record) < shadowFieldCount {
			return nil, fmt.Errorf("invalid shadow entry, expected %d fields but got %d", shadowFieldCount, len(record))
		}

		// the /etc/shadow file gives the count of days since jan 1, 1970
		start := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		var lastChangedTime *time.Time
		if record[2] == "" {
			// if the last_changes field is an empty string, nothing was ever changed, return nil
			lastChangedTime = nil
		} else {
			i, err := strconv.Atoi(record[2])
			if err != nil {
				return nil, err
			}
			date := start.Add(time.Hour * 24 * time.Duration(i))
			lastChangedTime = &date
		}
		res = append(res, ShadowEntry{
			User:         strings.TrimSpace(record[0]),
			Password:     strings.TrimSpace(record[1]),
			LastChanged:  lastChangedTime,
			MinDays:      strings.TrimSpace(record[3]),
			MaxDays:      strings.TrimSpace(record[4]),
			WarnDays:     strings.TrimSpace(record[5]),
			InactiveDays: strings.TrimSpace(record[6]),
			ExpiryDates:  strings.TrimSpace(record[7]),
			Reserved:     strings.TrimSpace(record[8]),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return res, nil
}
