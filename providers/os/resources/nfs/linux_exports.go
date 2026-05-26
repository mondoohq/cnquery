// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package nfs

import (
	"io"
	"strings"
)

// parseLinuxExports parses /etc/exports in the syntax used by the
// Linux NFS server (nfs-utils). Each line is
//
//	path  client1(opt,opt,...) client2(opt,opt,...)
//
// where the path is the first whitespace-separated token and each
// subsequent token is a client name optionally followed by a
// parenthesized comma-separated option list. A bare client without
// parentheses is treated as having an empty option list.
func parseLinuxExports(r io.Reader) ([]ExportEntry, error) {
	var entries []ExportEntry
	err := scanExportLines(r, func(line string) error {
		path, rest := splitFirstField(line)
		if path == "" || !strings.HasPrefix(path, "/") {
			return nil
		}
		clients := tokenizeLinuxClients(rest)
		for _, c := range clients {
			entries = append(entries, ExportEntry{
				Path:         path,
				Client:       c.host,
				Options:      c.options,
				ReadOnly:     containsString(c.options, "ro"),
				NoRootSquash: containsString(c.options, "no_root_squash"),
			})
		}
		return nil
	})
	return entries, err
}

type linuxClient struct {
	host    string
	options []string
}

// tokenizeLinuxClients splits the post-path remainder of an exports
// line into client/options pairs. Tokens are whitespace-separated;
// each token is either `host(opt,opt,...)`, `(opt,opt,...)` (no host
// — treated as "*"), or a bare `host`.
func tokenizeLinuxClients(s string) []linuxClient {
	var clients []linuxClient
	i := 0
	for i < len(s) {
		// skip whitespace
		for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		if i >= len(s) {
			break
		}

		// read host (up to '(' or whitespace)
		start := i
		for i < len(s) && s[i] != '(' && s[i] != ' ' && s[i] != '\t' {
			i++
		}
		host := s[start:i]

		var options []string
		if i < len(s) && s[i] == '(' {
			i++ // consume '('
			optStart := i
			for i < len(s) && s[i] != ')' {
				i++
			}
			optRaw := s[optStart:i]
			if i < len(s) {
				i++ // consume ')'
			}
			options = splitOptionList(optRaw)
		}

		if host == "" {
			host = "*"
		}
		clients = append(clients, linuxClient{host: host, options: options})
	}
	return clients
}

// splitOptionList splits "ro,no_root_squash,sec=krb5p" into
// ["ro", "no_root_squash", "sec=krb5p"], trimming surrounding
// whitespace and dropping empty pieces.
func splitOptionList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// splitFirstField returns the first whitespace-separated field and
// the remainder of the line (with leading whitespace trimmed).
func splitFirstField(line string) (string, string) {
	line = strings.TrimSpace(line)
	for i := 0; i < len(line); i++ {
		if line[i] == ' ' || line[i] == '\t' {
			return line[:i], strings.TrimSpace(line[i:])
		}
	}
	return line, ""
}
