// Copyright (c) Mondoo, Inc.
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

// shellEscape escapes a string for safe use in single-quoted shell strings.
// It handles single quotes by ending the single-quoted string, adding an escaped
// single quote, and starting a new single-quoted string.
func shellEscape(s string) string {
	// Replace ' with '\'' (end quote, escaped quote, start quote)
	return strings.ReplaceAll(s, "'", `'\''`)
}

func BuildFilesFindCmd(from string, xdev bool, fileType string, regex string, permission int64, search string, depth *int64) string {
	var call strings.Builder
	call.WriteString("find -L ")
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
		call.WriteString(" -regex '")
		call.WriteString(shellEscape(regex))
		call.WriteString("'")
	}

	if permission != 0o777 {
		call.WriteString(" -perm -")
		call.WriteString(Octal2string(permission))
	}

	if search != "" {
		call.WriteString(" -name '")
		call.WriteString(shellEscape(search))
		call.WriteString("'")
	}

	if depth != nil {
		call.WriteString(" -maxdepth ")
		call.WriteString(strconv.FormatInt(*depth, 10))
	}
	return call.String()
}
