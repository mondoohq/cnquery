// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package date

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

// unixDateCmd gets the current UTC time. Used when RunCommand is available.
const unixDateCmd = `date -u +%Y-%m-%dT%H:%M:%SZ`

type Unix struct {
	conn shared.Connection
}

func (u *Unix) Name() string {
	return "Unix Date"
}

func (u *Unix) Get() (*Result, error) {
	canRunCmd := u.conn.Capabilities().Has(shared.Capability_RunCommand)

	// Get UTC time only if we can actually ask the remote system.
	// For static targets (EBS snapshots, Docker images) there is no
	// meaningful current time, so we leave it nil.
	var utcTime *time.Time
	if canRunCmd {
		cmd, err := u.conn.RunCommand(unixDateCmd)
		if err != nil {
			return nil, fmt.Errorf("failed to get system date: %w", err)
		}
		t, err := parseUTCTime(cmd.Stdout)
		if err != nil {
			return nil, err
		}
		utcTime = &t
	}

	// Get timezone: try filesystem first (works on EBS snapshots, Docker images),
	// fall back to command if filesystem detection fails
	tz, err := timezoneFromFS(u.conn.FileSystem())
	if err != nil && canRunCmd {
		tz, err = timezoneFromCmd(u.conn)
	}
	if err != nil {
		// If all methods fail, default to UTC
		return &Result{
			Time:     utcTime,
			Timezone: "UTC",
		}, nil
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		return &Result{
			Time:     utcTime,
			Timezone: tz,
		}, nil
	}

	if utcTime != nil {
		t := utcTime.In(loc)
		utcTime = &t
	}

	return &Result{
		Time:     utcTime,
		Timezone: tz,
	}, nil
}

// timezoneFromFS detects the IANA timezone by reading filesystem artifacts.
// It tries these approaches in order:
//  1. readlink /etc/localtime → extract IANA name from symlink target
//  2. Read /etc/timezone (Debian/Ubuntu)
//  3. Parse TZ= from /etc/TIMEZONE (Solaris/AIX)
//  4. Match /etc/localtime contents against the system zoneinfo database
func timezoneFromFS(fs afero.Fs) (string, error) {
	// 1. Try readlink on /etc/localtime
	if lr, ok := fs.(afero.LinkReader); ok {
		if target, err := lr.ReadlinkIfPossible("/etc/localtime"); err == nil {
			if tz := extractTZFromPath(target); tz != "" {
				return tz, nil
			}
		}
	}

	// 2. Try /etc/timezone (Debian/Ubuntu)
	if f, err := fs.Open("/etc/timezone"); err == nil {
		defer f.Close()
		content, err := io.ReadAll(f)
		if err == nil {
			if tz := strings.TrimSpace(string(content)); tz != "" {
				return tz, nil
			}
		}
	}

	// 3. Try /etc/TIMEZONE (Solaris/AIX) - look for TZ=<value>
	if f, err := fs.Open("/etc/TIMEZONE"); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if tz, ok := strings.CutPrefix(line, "TZ="); ok && tz != "" {
				return tz, nil
			}
		}
	}

	// 4. Try matching /etc/localtime binary content against zoneinfo database
	if tz, err := matchLocaltimeToZoneinfo(fs); err == nil {
		return tz, nil
	}

	return "", fmt.Errorf("could not detect timezone from filesystem")
}

// extractTZFromPath extracts the IANA timezone name from a symlink target path.
// e.g., "/usr/share/zoneinfo/America/New_York" → "America/New_York"
func extractTZFromPath(path string) string {
	const marker = "zoneinfo/"
	if idx := strings.LastIndex(path, marker); idx >= 0 {
		tz := path[idx+len(marker):]
		if tz != "" && tz != "localtime" {
			return tz
		}
	}
	return ""
}

// matchLocaltimeToZoneinfo reads /etc/localtime and tries to find a matching
// zoneinfo file. This handles cases where /etc/localtime is a regular file
// (copy, not symlink), common in Docker images.
func matchLocaltimeToZoneinfo(fs afero.Fs) (string, error) {
	localtime, err := afero.ReadFile(fs, "/etc/localtime")
	if err != nil {
		return "", err
	}
	if len(localtime) < 4 || string(localtime[:4]) != "TZif" {
		return "", fmt.Errorf("/etc/localtime is not a valid TZif file")
	}

	// Walk common zoneinfo directories to find a match
	for _, base := range []string{"/usr/share/zoneinfo", "/usr/share/lib/zoneinfo"} {
		tz, err := findMatchingZoneinfo(fs, base, localtime)
		if err == nil {
			return tz, nil
		}
	}
	return "", fmt.Errorf("no matching zoneinfo file found")
}

// findMatchingZoneinfo walks a zoneinfo directory tree comparing file contents
// to the given localtime data.
func findMatchingZoneinfo(fs afero.Fs, base string, localtime []byte) (string, error) {
	var match string
	err := afero.Walk(fs, base, func(path string, info os.FileInfo, err error) error {
		if err != nil || match != "" {
			return err
		}
		if info.IsDir() {
			// Skip directories that aren't timezone data
			name := info.Name()
			if name == "posix" || name == "right" {
				return filepath.SkipDir
			}
			return nil
		}
		// Only compare regular files
		if !info.Mode().IsRegular() {
			return nil
		}

		candidate, err := afero.ReadFile(fs, path)
		if err != nil {
			return nil // skip unreadable files
		}
		if len(candidate) == len(localtime) && string(candidate) == string(localtime) {
			rel := strings.TrimPrefix(path, base+"/")
			// Validate it looks like an IANA name (contains a slash, e.g. "America/New_York")
			if strings.Contains(rel, "/") {
				match = rel
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if match == "" {
		return "", fmt.Errorf("no match found in %s", base)
	}
	return match, nil
}

// timezoneFromCmd gets the abbreviated timezone name via `date +%Z`.
// This is a last resort — it only returns short names like "EST", not IANA names.
func timezoneFromCmd(conn shared.Connection) (string, error) {
	cmd, err := conn.RunCommand("date +%Z")
	if err != nil {
		return "", fmt.Errorf("failed to get system timezone: %w", err)
	}
	return parseTimezone(cmd.Stdout)
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
