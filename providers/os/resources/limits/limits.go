// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package limits

import (
	"regexp"
	"strings"
)

var (
	// Regular expression for parsing limits entries
	// Format: <domain> <type> <item> <value>
	EntryRegex = regexp.MustCompile(`^(\S+)\s+(soft|hard|-)\s+(\S+)\s+(\S+)`)
)

// Entry represents a parsed limits entry
type Entry struct {
	File       string
	LineNumber int
	Domain     string
	Type       string
	Item       string
	Value      string
}

// ParseLines parses the content of a limits file and returns structured entries
func ParseLines(filePath string, content string) []Entry {
	var entries []Entry
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		actualLineNum := lineNum + 1

		// Skip empty lines and comments
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse the line using regex
		matches := EntryRegex.FindStringSubmatch(line)
		if matches == nil {
			// Invalid format, skip
			continue
		}

		entries = append(entries, Entry{
			File:       filePath,
			LineNumber: actualLineNum,
			Domain:     matches[1],
			Type:       matches[2],
			Item:       matches[3],
			Value:      matches[4],
		})
	}

	return entries
}
