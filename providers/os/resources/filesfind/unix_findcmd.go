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
		// TODO: we need to escape regex here
		call.WriteString(" -regex '")
		call.WriteString(regex)
		call.WriteString("'")
	}

	if permission != 0o777 {
		call.WriteString(" -perm -")
		call.WriteString(Octal2string(permission))
	}

	if search != "" {
		call.WriteString(" -name ")
		call.WriteString(search)
	}

	if depth != nil {
		call.WriteString(" -maxdepth ")
		call.WriteString(Octal2string(*depth))
	}
	return call.String()
}
