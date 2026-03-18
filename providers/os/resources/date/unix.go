// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package date

import (
	"fmt"
	"io"
	"strings"
	"time"

	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

// Unix date command format: outputs ISO 8601 date and IANA timezone name.
// This works on Linux, macOS, FreeBSD, AIX, and Solaris.
// The command outputs two lines:
//
//	Line 1: date in RFC 3339 format (date -u +%Y-%m-%dT%H:%M:%SZ)
//	Line 2: timezone identifier from /etc/localtime or TZ env var
//
// We use two separate approaches:
//   - UTC time via `date -u` which is universally supported
//   - Timezone name via reading the TZ symlink or zone file
const (
	// Get current UTC time in RFC 3339 and the timezone name.
	// The date command with these format specifiers works across all target Unix platforms.
	// %Z gives abbreviated timezone (e.g., "EST"), but we prefer the IANA name.
	unixDateCmd = `date -u +%Y-%m-%dT%H:%M:%SZ`

	// Get IANA timezone name. Try multiple approaches for cross-platform compatibility:
	// 1. readlink /etc/localtime (Linux, macOS, FreeBSD)
	// 2. Read /etc/timezone (Debian/Ubuntu)
	// 3. Parse /etc/TIMEZONE (Solaris/AIX)
	// 4. Fall back to date +%Z (abbreviated name)
	unixTimezoneCmd = `if [ -L /etc/localtime ]; then readlink /etc/localtime | sed 's|.*/zoneinfo/||'; elif [ -f /etc/timezone ]; then cat /etc/timezone; elif [ -f /etc/TIMEZONE ]; then grep '^TZ=' /etc/TIMEZONE | cut -d= -f2; else date +%Z; fi`
)

type Unix struct {
	conn shared.Connection
}

func (u *Unix) Name() string {
	return "Unix Date"
}

func (u *Unix) Get() (*Result, error) {
	// Get UTC time
	cmd, err := u.conn.RunCommand(unixDateCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get system date: %w", err)
	}
	utcTime, err := parseUTCTime(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	// Get timezone
	cmd, err = u.conn.RunCommand(unixTimezoneCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get system timezone: %w", err)
	}
	tz, err := parseTimezone(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	// Load the timezone location and convert the UTC time
	loc, err := time.LoadLocation(tz)
	if err != nil {
		// If the timezone can't be loaded (e.g., abbreviated name like "EST"),
		// return the UTC time with the timezone string as-is
		return &Result{
			Time:     utcTime,
			Timezone: tz,
		}, nil
	}

	return &Result{
		Time:     utcTime.In(loc),
		Timezone: tz,
	}, nil
}

func parseUTCTime(r io.Reader) (time.Time, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to read date output: %w", err)
	}

	s := strings.TrimSpace(string(content))
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date output")
	}

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse date %q: %w", s, err)
	}
	return t, nil
}

func parseTimezone(r io.Reader) (string, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to read timezone output: %w", err)
	}

	s := strings.TrimSpace(string(content))
	if s == "" {
		return "UTC", nil
	}
	return s, nil
}
