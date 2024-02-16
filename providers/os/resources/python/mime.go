// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package python

import (
	"bufio"
	"fmt"
	"io"
	"net/textproto"
	"strings"
)

type PackageDetails struct {
	Name         string
	File         string
	License      string
	Author       string
	AuthorEmail  string
	Summary      string
	Version      string
	Dependencies []string
	IsLeaf       bool
	Purl         string
	Cpes         []string
}

// extractMimeDeps will go through each of the listed dependencies
// from the "Requires-Dist" values, and strip off everything but
// the name of the package/dependency itself
func extractMimeDeps(deps []string) []string {
	parsedDeps := []string{}
	for _, dep := range deps {
		// the semicolon indicates an optional dependency
		if strings.Contains(dep, ";") {
			continue
		}
		parsedDep := strings.Split(dep, " ")
		if len(parsedDep) > 0 {
			parsedDeps = append(parsedDeps, parsedDep[0])
		}
	}
	return parsedDeps
}

func ParseMIME(r io.Reader, pythonMIMEFilepath string) (*PackageDetails, error) {
	textReader := textproto.NewReader(bufio.NewReader(r))
	mimeData, err := textReader.ReadMIMEHeader()
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("error reading MIME data: %s", err)
	}

	deps := extractMimeDeps(mimeData.Values("Requires-Dist"))

	return &PackageDetails{
		Name:         mimeData.Get("Name"),
		Summary:      mimeData.Get("Summary"),
		Author:       mimeData.Get("Author"),
		AuthorEmail:  mimeData.Get("Author-email"),
		License:      mimeData.Get("License"),
		Version:      mimeData.Get("Version"),
		Dependencies: deps,
		File:         pythonMIMEFilepath,
		Purl:         NewPackageUrl(mimeData.Get("Name"), mimeData.Get("Version")),
		Cpes:         NewCpes(mimeData.Get("Name"), mimeData.Get("Version")),
	}, nil
}
