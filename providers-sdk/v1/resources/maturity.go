// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "fmt"

// Maturity levels for providers, connectors, resources, and fields.
// The default is Stable, represented by an empty string.
const (
	MaturityExperimental = "experimental"
	MaturityPreview      = "preview"
	MaturityStable       = "stable"
	MaturityDeprecated   = "deprecated"
	MaturityEOL          = "eol"
)

// MaturityLevel returns the ordering index for a maturity string.
// Empty string and "stable" both return 0.
// experimental=1, preview=2, deprecated=3, eol=4.
//
// The ordering represents lifecycle progression, not instability. When
// combining resource + field maturity via EffectiveMaturity, the higher
// level wins. This means:
//   - A field explicitly marked "preview" on an "experimental" resource
//     is effectively "preview" (the field author promoted it).
//   - A field marked "experimental" on a "deprecated" resource is still
//     "deprecated" (the resource's end-of-life state dominates).
func MaturityLevel(m string) int {
	switch m {
	case "", MaturityStable:
		return 0
	case MaturityExperimental:
		return 1
	case MaturityPreview:
		return 2
	case MaturityDeprecated:
		return 3
	case MaturityEOL:
		return 4
	default:
		return -1
	}
}

// EffectiveMaturity combines two maturity values (e.g. from a resource and
// a field) and returns the one that should be shown to users.
//
// Rules:
//   - If both are stable (empty), return empty.
//   - If one is non-stable and the other is stable, the non-stable one wins.
//   - If both are non-stable, the one with the higher level wins.
func EffectiveMaturity(a, b string) string {
	la := MaturityLevel(a)
	lb := MaturityLevel(b)
	if la == 0 && lb == 0 {
		return ""
	}
	if la == 0 {
		return b
	}
	if lb == 0 {
		return a
	}
	if la >= lb {
		return a
	}
	return b
}

// EffectiveFieldMaturity returns the effective maturity for a field,
// combining the resource-level and field-level maturity.
func EffectiveFieldMaturity(resource *ResourceInfo, field *Field) string {
	return EffectiveMaturity(resource.GetMaturity(), field.GetMaturity())
}

// ValidateMaturity checks that a maturity string is a known value.
// Empty string is valid (means Stable).
func ValidateMaturity(m string) error {
	if MaturityLevel(m) >= 0 {
		return nil
	}
	return fmt.Errorf("invalid maturity %q, must be one of: experimental, preview, stable, deprecated, eol", m)
}

// MaturityLabel returns a human-readable label for a maturity string,
// with the first letter capitalized. Returns empty string for stable/empty.
func MaturityLabel(m string) string {
	switch m {
	case MaturityExperimental:
		return "Experimental"
	case MaturityPreview:
		return "Preview"
	case MaturityDeprecated:
		return "Deprecated"
	case MaturityEOL:
		return "EOL"
	default:
		return ""
	}
}
