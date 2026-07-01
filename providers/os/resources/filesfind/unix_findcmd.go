// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package filesfind

import (
	"fmt"
	"strconv"
	"strings"
)

var findTypes = map[string]string{
	"file":      "f",
	"directory": "d",
	"character": "c",
	"block":     "b",
	"socket":    "s",
	"link":      "l",
}

func Octal2string(o int64) string {
	return fmt.Sprintf("%o", o)
}

// shellSingleQuote wraps s in single quotes so the shell passes it to the
// command verbatim, with no glob, variable, or command-substitution expansion.
// Any single quote in s is escaped by closing, inserting an escaped quote, then reopening.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func BuildFilesFindCmd(from string, xdev bool, fileType string, regex string, permission int64, search string, depth *int64) string {
	var call strings.Builder

	isLinkSearch := false
	if fileType != "" {
		if t, ok := findTypes[fileType]; ok && t == "l" {
			isLinkSearch = true
		}
	}

	// -L follows all symlinks, so -type l only matches dangling ones.
	// -xtype l fixes that on GNU find but is absent on BSD (macOS).
	// -H follows only command-line symlinks (resolving the start path)
	// while keeping -type l functional for discovered symlinks.
	if isLinkSearch {
		call.WriteString("find -H ")
	} else {
		call.WriteString("find -L ")
	}
	call.WriteString(strconv.Quote(from))

	if !xdev {
		call.WriteString(" -xdev")
	}

	if fileType != "" {
		t, ok := findTypes[fileType]
		if ok {
			call.WriteString(" -type " + t)
		}
	}

	if regex != "" {
		call.WriteString(" -regex ")
		call.WriteString(shellSingleQuote(regex))
	}

	if permission != 0o777 {
		call.WriteString(" -perm -")
		call.WriteString(Octal2string(permission))
	}

	if search != "" {
		call.WriteString(" -name ")
		// Single-quote the pattern so the shell passes it to find verbatim,
		// with no glob, variable, or command-substitution expansion.
		call.WriteString(shellSingleQuote(search))
	}

	if depth != nil {
		call.WriteString(" -maxdepth ")
		// -maxdepth takes a decimal level count, not an octal value.
		call.WriteString(strconv.FormatInt(*depth, 10))
	}
	return call.String()
}
