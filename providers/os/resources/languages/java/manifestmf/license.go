// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package manifestmf

import (
	"regexp"
	"strings"
)

// OSGi states a bundle's license in Bundle-License, and the header is
// structured rather than a plain name (OSGi Core, Module Layer):
//
//	Bundle-License ::= '<<EXTERNAL>>' | ( license ( ',' license ) * )
//	license        ::= license-identifier ( ';' license-attr ) *
//	license-attr   ::= description | link
//	description    ::= 'description' '=' string
//	link           ::= 'link' '=' <url>
//
// so a real header looks like any of:
//
//	Bundle-License: Apache-2.0;link="https://www.apache.org/licenses/LICENSE-2.0"
//	Bundle-License: The Apache License, Version 2.0;link="http://www.apache.org/..."
//	Bundle-License: https://www.eclipse.org/legal/epl-2.0/
//	Bundle-License: <<EXTERNAL>>
//
// Only the license-identifier is carried into the inventory. The attributes are
// dropped: `link` points at the license text and `description` is prose — the
// bnd documentation's own example is a full sentence — and neither is the
// identity of the license.

// bundleLicenseExternal is the magic identifier meaning "this artifact states
// no license here; it is provided some other way". It is a statement that the
// header is empty, so it is treated as one.
const bundleLicenseExternal = "<<EXTERNAL>>"

// urlPattern matches a value that is a URL rather than a license name.
var urlPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.\-]*://`)

// license returns the license the manifest declares, or "" when it declares
// none this can name.
//
// The value is returned as the manifest wrote it. Normalizing a name to a
// canonical SPDX identifier is a consumer's decision, not a parser's.
func (m *manifest) license() string {
	raw := strings.TrimSpace(m.Headers[headerBundleLicense])
	if raw == "" {
		return ""
	}

	names := make([]string, 0, 1)
	for _, entry := range splitOutsideQuotes(raw, ',') {
		name := licenseIdentifier(entry)
		// A URL is a link to the terms, not the identity of the terms.
		// Reporting one in a field consumers compare against "Apache-2.0" or
		// "MIT" matches nothing while reading as a stated identifier, which is
		// worse than the honest blank they get today. The header is
		// URL-only on most real bundles, so most of them keep reporting
		// nothing — this only adds the ones that name a license.
		if name == "" || name == bundleLicenseExternal || urlPattern.MatchString(name) {
			continue
		}
		names = append(names, name)
	}

	// The grammar makes ',' a separator between licenses, but real manifests
	// also write an identifier that contains one: "The Apache License, Version
	// 2.0;link=..." is a single license, and it is what Eclipse/Tycho bundles
	// emit. Nothing distinguishes that from a genuine two-license header, so
	// the members are rejoined into one value rather than OR-joined. Rejoining
	// reproduces what the manifest said; OR-joining would report a bundle as
	// dual-licensed under "The Apache License" or "Version 2.0", inventing a
	// choice the bundle never offered.
	return strings.Join(names, ", ")
}

// licenseIdentifier returns the license-identifier of one entry: everything
// before its first attribute, unquoted and trimmed.
func licenseIdentifier(entry string) string {
	parts := splitOutsideQuotes(entry, ';')
	if len(parts) == 0 {
		return ""
	}
	return unquote(strings.TrimSpace(parts[0]))
}

// splitOutsideQuotes splits on sep, ignoring separators inside a quoted string.
// Attribute values are quoted and do carry both separators —
// `description="Apache License, Version 2.0"` holds a comma, and a `link` URL
// can hold anything.
func splitOutsideQuotes(s string, sep byte) []string {
	var (
		parts []string
		start int
		quote byte
	)
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == sep:
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	return append(parts, s[start:])
}

// unquote strips one layer of matching surrounding quotes.
func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return strings.TrimSpace(s[1 : len(s)-1])
	}
	return s
}
