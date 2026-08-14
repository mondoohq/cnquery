// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cpe

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/facebookincubator/nvdtools/wfn"
)

// epochRegex matches a leading epoch such as "2:" in "2:4.17.12". NewPackage2Cpe runs once
// per package, so the pattern is compiled once here and not on every call.
var epochRegex = regexp.MustCompile(`^\d+:(.*)$`)

func NewPackage2Cpe(vendor, name, version, release, arch string) ([]string, error) {
	cpes := []string{}
	vendor = strings.ToLower(vendor)
	name = strings.ToLower(name)
	version = strings.ToLower(version)
	release = strings.ToLower(release)
	arch = strings.ToLower(arch)

	// Remove epoch when present; otherwise WFNize will only use the epoch as
	// the version.
	if matches := epochRegex.FindStringSubmatch(version); len(matches) > 1 {
		version = matches[1]
	}

	var err error
	// A fixed array avoids building a map on every call, and it reports the same field
	// first when more than one field fails.
	for _, field := range [...]struct {
		name string
		addr *string
	}{
		{"vendor", &vendor},
		{"name", &name},
		{"version", &version},
		{"release", &release},
		{"arch", &arch},
	} {
		if *field.addr, err = wfn.WFNize(*field.addr); err != nil {
			return cpes, fmt.Errorf("couldn't wfnize %s %q: %v", field.name, *field.addr, err)
		}
	}

	// A CPE needs both a product name and a version. When either is missing we
	// simply cannot build one — that is not an error worth surfacing, since CPEs
	// are optional vulnerability-matching enrichment. Return no CPEs and no error
	// so callers don't log spurious warnings for nameless/versionless packages
	// (common in JS lockfiles).
	if name == "" || version == "" {
		return cpes, nil
	}

	attr := wfn.Attributes{}
	attr.Part = "a"
	attr.Vendor = vendor
	attr.Product = name
	attr.Version = version
	attr.Update = release
	attr.TargetHW = arch

	cpes = append(cpes, attr.BindToFmtString())
	return cpes, nil
}
