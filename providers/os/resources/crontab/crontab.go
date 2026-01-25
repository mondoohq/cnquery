// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package crontab

import (
	"bufio"
	"io"
	"strings"
)

// Entry represents a single crontab entry
type Entry struct {
	LineNumber int
	Minute     string
	Hour       string
	DayOfMonth string
	Month      string
	DayOfWeek  string
	User       string
	Command    string
}

// ParseCrontab parses a crontab file content and returns the entries.
// If hasUserField is true, it expects the user field after the time fields
// (system crontab format). Otherwise, it parses user crontab format.
func ParseCrontab(r io.Reader, hasUserField bool) ([]Entry, error) {
	var entries []Entry

	scanner := bufio.NewScanner(r)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Skip empty lines and comments
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Skip environment variable assignments (e.g., SHELL=/bin/bash)
		if strings.Contains(line, "=") && !strings.HasPrefix(line, "@") {
			// Check if this looks like a variable assignment (key=value without spaces before =)
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 && !strings.ContainsAny(parts[0], " \t") {
				continue
			}
		}

		entry, ok := parseLine(line, lineNumber, hasUserField)
		if ok {
			entries = append(entries, entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

// parseLine parses a single crontab line and returns an Entry
func parseLine(line string, lineNumber int, hasUserField bool) (Entry, bool) {
	entry := Entry{LineNumber: lineNumber}

	// Handle special time strings (@reboot, @hourly, etc.)
	if strings.HasPrefix(line, "@") {
		return parseSpecialLine(line, lineNumber, hasUserField)
	}

	// Standard crontab format: min hour dom mon dow [user] command
	fields := strings.Fields(line)

	minFields := 6 // min hour dom mon dow command
	if hasUserField {
		minFields = 7 // min hour dom mon dow user command
	}

	if len(fields) < minFields {
		return entry, false
	}

	entry.Minute = fields[0]
	entry.Hour = fields[1]
	entry.DayOfMonth = fields[2]
	entry.Month = fields[3]
	entry.DayOfWeek = fields[4]

	if hasUserField {
		entry.User = fields[5]
		// Command is everything after the user field
		entry.Command = strings.Join(fields[6:], " ")
	} else {
		// For user crontabs, no user field - command starts at field 5
		entry.Command = strings.Join(fields[5:], " ")
	}

	return entry, true
}

// parseSpecialLine parses lines with special time strings like @reboot, @hourly, etc.
func parseSpecialLine(line string, lineNumber int, hasUserField bool) (Entry, bool) {
	entry := Entry{LineNumber: lineNumber}

	fields := strings.Fields(line)

	minFields := 2 // @special command
	if hasUserField {
		minFields = 3 // @special user command
	}

	if len(fields) < minFields {
		return entry, false
	}

	// Map special strings to their crontab equivalents for display
	special := fields[0]
	switch special {
	case "@reboot":
		entry.Minute = "@reboot"
		entry.Hour = ""
		entry.DayOfMonth = ""
		entry.Month = ""
		entry.DayOfWeek = ""
	case "@yearly", "@annually":
		entry.Minute = "0"
		entry.Hour = "0"
		entry.DayOfMonth = "1"
		entry.Month = "1"
		entry.DayOfWeek = "*"
	case "@monthly":
		entry.Minute = "0"
		entry.Hour = "0"
		entry.DayOfMonth = "1"
		entry.Month = "*"
		entry.DayOfWeek = "*"
	case "@weekly":
		entry.Minute = "0"
		entry.Hour = "0"
		entry.DayOfMonth = "*"
		entry.Month = "*"
		entry.DayOfWeek = "0"
	case "@daily", "@midnight":
		entry.Minute = "0"
		entry.Hour = "0"
		entry.DayOfMonth = "*"
		entry.Month = "*"
		entry.DayOfWeek = "*"
	case "@hourly":
		entry.Minute = "0"
		entry.Hour = "*"
		entry.DayOfMonth = "*"
		entry.Month = "*"
		entry.DayOfWeek = "*"
	default:
		// Unknown special string, store as-is
		entry.Minute = special
		entry.Hour = ""
		entry.DayOfMonth = ""
		entry.Month = ""
		entry.DayOfWeek = ""
	}

	if hasUserField {
		entry.User = fields[1]
		entry.Command = strings.Join(fields[2:], " ")
	} else {
		entry.Command = strings.Join(fields[1:], " ")
	}

	return entry, true
}
