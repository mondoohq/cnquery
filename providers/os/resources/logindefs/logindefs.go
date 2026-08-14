// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package logindefs

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

// ignore line if it starts with a comment, allow trailing comments though.
// Compiled once: matched against every line of login.defs.
var logindefEntry = regexp.MustCompile(`^\s*([^#]\S+)\s+(\S+)\s*(?:#.*)?$`)

func Parse(r io.Reader) map[string]string {
	res := map[string]string{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		noWhitespace := strings.TrimSpace(line)

		m := logindefEntry.FindStringSubmatch(noWhitespace)
		if len(m) == 3 {
			res[m[1]] = m[2]
		}
	}

	return res
}
