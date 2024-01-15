// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package python

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

// firstWordRegexp is just trying to catch everything leading up the >, >=, = in a requires.txt
// Example:
//
// nose>=1.2
// Mock>=1.0
// pycryptodome
//
// [crypto]
// pycryptopp>=0.5.12
//
// [cryptography]
// cryptography
//
// would match nose / Mock / pycrptodome / etc

var firstWordRegexp = regexp.MustCompile(`^[a-zA-Z0-9\._-]*`)

func ParseRequiresTxtDependencies(r io.Reader) ([]string, error) {
	fileScanner := bufio.NewScanner(r)
	fileScanner.Split(bufio.ScanLines)

	dependencies := []string{}
	for fileScanner.Scan() {
		line := fileScanner.Text()
		if strings.HasPrefix(line, "[") {
			// this means a new optional section of dependencies
			// so stop processing
			break
		}
		matched := firstWordRegexp.FindString(line)
		if matched == "" {
			continue
		}
		dependencies = append(dependencies, matched)
	}

	return dependencies, nil
}
