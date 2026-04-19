// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package firmware

import (
	"strings"

	packageurl "github.com/package-url/packageurl-go"
)

// Device represents a firmware component with version and vendor metadata.
type Device struct {
	Name          string
	DeviceId      string
	Version       string
	Vendor        string
	VendorId      string
	Summary       string
	Guid          []string
	Plugin        string
	Protocol      string
	Flags         []string
	VersionFormat string
	Updatable     bool
	Purl          string
}

// normalizePurlSegment normalises a name or vendor for use in a PURL path
// segment: lowercase, spaces → hyphens, strip non-alnum/hyphen.
func normalizePurlSegment(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '.' {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// GeneratePurl creates a generic PURL for a firmware device.
// Format: pkg:generic/<vendor>/<name>@<version>
func GeneratePurl(vendor, name, version string) string {
	if name == "" {
		return ""
	}
	ns := normalizePurlSegment(vendor)
	n := normalizePurlSegment(name)
	if n == "" {
		return ""
	}
	return packageurl.NewPackageURL(
		"generic",
		ns,
		n,
		version,
		nil,
		"",
	).String()
}
