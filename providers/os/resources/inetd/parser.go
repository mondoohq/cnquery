// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package inetd parses classic inetd super-server configuration files.
package inetd

import "strings"

// Entry is a single active inetd service line. The columns follow the classic
// inetd.conf layout:
//
//	service-name  socket-type  protocol  wait|nowait[.max]  user[.group]  server-program  server-arguments
type Entry struct {
	Name       string
	SocketType string
	Protocol   string
	Wait       string
	User       string
	Server     string
	Arguments  string
	// Line is the 1-based line number of this entry within its file.
	Line int
}

// Parse reads inetd.conf content and returns the active service entries. Blank
// lines and comment lines (those whose first non-whitespace character is #) are
// skipped, so disabled entries never show up. Lines with fewer than the six
// fixed columns are treated as malformed and ignored.
func Parse(content string) []Entry {
	entries := []Entry{}
	for i, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		e := Entry{
			Name:       fields[0],
			SocketType: fields[1],
			Protocol:   fields[2],
			Wait:       fields[3],
			User:       fields[4],
			Server:     fields[5],
			Line:       i + 1,
		}
		if len(fields) > 6 {
			e.Arguments = strings.Join(fields[6:], " ")
		}
		entries = append(entries, e)
	}
	return entries
}
