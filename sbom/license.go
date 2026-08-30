// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sbom

import "strings"

// DeclaredLicense builds the License model entry for a license a package
// DECLARED — the string its manifest or metadata stated about itself, as
// opposed to one concluded by reading the files it ships.
//
// It returns nil when there is nothing to record. An entry naming no license
// asserts a licensing fact the producer does not have, and is worse than an
// absent one: a consumer cannot tell it apart from a real determination.
//
// Which of the three mutually exclusive value fields gets set is decided by
// what the string actually is, using the same rule the CycloneDX renderer
// applies to the legacy scalar — one decision in one place, so the model and
// the rendering of it cannot disagree about whether "MIT OR Apache-2.0" is an
// identifier.
//
// Confidence is 1.0 and not a parameter. A declared license is a statement the
// package made about itself rather than a measurement somebody took of it, so
// there is nothing to be less than certain about. A concluded license — which
// carries its match score — is produced by whatever did the concluding, not
// here.
func DeclaredLicense(value string) *License {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	l := &License{
		Acquisition: LicenseAcquisition_LICENSE_ACQUISITION_DECLARED,
		Confidence:  1.0,
	}
	switch {
	case isSPDXExpression(value):
		// Carried exactly as written. Re-serializing an expression means
		// parsing and re-printing it, and that round trip drops grouping:
		// "(MIT OR Apache-2.0) AND BSD-3-Clause" comes back as
		// "MIT OR Apache-2.0 AND BSD-3-Clause", which is a different license,
		// since AND binds tighter and the parentheses were carrying the meaning.
		l.Expression = value
	case isSPDXIdentifierShaped(value):
		l.SpdxId = value
	default:
		// Neither an identifier nor an expression: free text such as
		// "BSD-like, see LICENSE". It travels as a name, which is honest —
		// forcing it into the id field would be a claim the source does not
		// support, and would produce a document a consumer cannot resolve.
		l.Name = value
	}
	return l
}

// DeclaredLicenses is DeclaredLicense as the repeated field wants it: a
// one-entry list, or nil when the value states no license.
func DeclaredLicenses(value string) []*License {
	if l := DeclaredLicense(value); l != nil {
		return []*License{l}
	}
	return nil
}
