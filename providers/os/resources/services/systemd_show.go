// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package services

import (
	"bufio"
	"io"
	"strings"
)

// parseShowProperties parses key=value output from systemctl show.
func parseShowProperties(input io.Reader) (map[string]string, error) {
	props := map[string]string{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		props[key] = value
	}
	return props, scanner.Err()
}
