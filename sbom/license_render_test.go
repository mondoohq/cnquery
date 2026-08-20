// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"encoding/json"
	"strings"
	"testing"
)

// The renderers dropped every package license for as long as they existed,
// because nothing asserted on one: the SBOM tests all check packages, none
// checked licenses. These tests exist so that cannot recur.

func licenseBom() *Sbom {
	return &Sbom{
		Generator: &Generator{Vendor: "Mondoo, Inc", Name: "test", Version: "1"},
		Asset:     &Asset{Name: "test-asset", Platform: &Platform{Name: "linux", Version: "1"}},
		Packages: []*Package{
			{Name: "plain-id", Version: "1.0.0", Purl: "pkg:npm/plain-id@1.0.0", License: "MIT",
				Description: "a plain MIT-licensed package"},
			{Name: "expression", Version: "1.0.0", Purl: "pkg:npm/expression@1.0.0", License: "MIT OR Apache-2.0"},
			{Name: "with-exception", Version: "1.0.0", Purl: "pkg:npm/with-exception@1.0.0",
				License: "GPL-2.0-only WITH Classpath-exception-2.0"},
			{Name: "free-text", Version: "1.0.0", Purl: "pkg:npm/free-text@1.0.0", License: "BSD-like, see LICENSE"},
			{Name: "custom-ref", Version: "1.0.0", Purl: "pkg:npm/custom-ref@1.0.0", License: "LicenseRef-Acme-Internal"},
			{Name: "undeclared", Version: "1.0.0", Purl: "pkg:npm/undeclared@1.0.0"},
		},
	}
}

func renderTo(t *testing.T, format string) string {
	t.Helper()
	var b strings.Builder
	h := New(format)
	if h == nil {
		t.Fatalf("no handler for %q", format)
	}
	if err := h.Render(&b, licenseBom()); err != nil {
		t.Fatalf("Render(%s): %v", format, err)
	}
	return b.String()
}

// TestCycloneDXEmitsLicenses covers the three mutually exclusive shapes
// CycloneDX models a license with. Choosing the wrong one fails schema
// validation, so the shape has to be decided per value rather than assumed.
func TestCycloneDXEmitsLicenses(t *testing.T) {
	var doc struct {
		Components []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Licenses    []struct {
				Expression string `json:"expression"`
				License    *struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"license"`
			} `json:"licenses"`
		} `json:"components"`
	}
	if err := json.Unmarshal([]byte(renderTo(t, FormatCycloneDxJSON)), &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	byName := map[string]int{}
	for i, c := range doc.Components {
		byName[c.Name] = i
	}
	get := func(name string) []struct {
		Expression string `json:"expression"`
		License    *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"license"`
	} {
		i, ok := byName[name]
		if !ok {
			t.Fatalf("component %q missing from the document", name)
		}
		return doc.Components[i].Licenses
	}

	// A bare SPDX identifier goes in license.id.
	if ls := get("plain-id"); len(ls) != 1 || ls[0].License == nil || ls[0].License.ID != "MIT" {
		t.Errorf("plain-id licenses = %+v, want license.id MIT", ls)
	}
	// The component description is carried through (it was previously dropped).
	if got := doc.Components[byName["plain-id"]].Description; got != "a plain MIT-licensed package" {
		t.Errorf("plain-id description = %q, want it carried through", got)
	}
	// An expression goes in expression, never in id.
	for _, name := range []string{"expression", "with-exception"} {
		ls := get(name)
		if len(ls) != 1 || ls[0].Expression == "" {
			t.Errorf("%s licenses = %+v, want an expression", name, ls)
			continue
		}
		if ls[0].License != nil {
			t.Errorf("%s set both expression and license — they are mutually exclusive", name)
		}
	}
	// Anything else goes in license.name rather than being dropped.
	if ls := get("free-text"); len(ls) != 1 || ls[0].License == nil || ls[0].License.Name == "" {
		t.Errorf("free-text licenses = %+v, want license.name", ls)
	}
	// A custom identifier is still an identifier.
	if ls := get("custom-ref"); len(ls) != 1 || ls[0].License == nil ||
		ls[0].License.ID != "LicenseRef-Acme-Internal" {
		t.Errorf("custom-ref licenses = %+v, want license.id", ls)
	}
	// An undeclared license emits no licenses key at all, which is valid
	// CycloneDX — unlike SPDX, it has no NOASSERTION to state.
	if ls := get("undeclared"); len(ls) != 0 {
		t.Errorf("undeclared licenses = %+v, want none", ls)
	}
}

// TestSPDXEmitsLicenseFields pins the three SPDX requirements the renderer used
// to miss: a concluded value, a copyright field, and NOASSERTION rather than an
// empty string.
func TestSPDXEmitsLicenseFields(t *testing.T) {
	out := renderTo(t, FormatSpdxJSON)

	var doc struct {
		Packages []struct {
			Name            string `json:"name"`
			LicenseDeclared string `json:"licenseDeclared"`
			LicenseCncluded string `json:"licenseConcluded"`
			CopyrightText   string `json:"copyrightText"`
		} `json:"packages"`
		HasExtractedLicensingInfos []struct {
			LicenseID     string `json:"licenseId"`
			ExtractedText string `json:"extractedText"`
		} `json:"hasExtractedLicensingInfos"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	for _, p := range doc.Packages {
		if p.LicenseDeclared == "" {
			t.Errorf("%s: licenseDeclared is empty; SPDX requires NOASSERTION", p.Name)
		}
		if p.LicenseCncluded == "" {
			t.Errorf("%s: licenseConcluded is missing", p.Name)
		}
		if p.CopyrightText == "" {
			t.Errorf("%s: copyrightText is empty; SPDX requires NOASSERTION", p.Name)
		}
		if p.Name == "undeclared" {
			if p.LicenseDeclared != "NOASSERTION" {
				t.Errorf("an undeclared license rendered as %q, want NOASSERTION", p.LicenseDeclared)
			}
		}
		if p.Name == "plain-id" && p.LicenseDeclared != "MIT" {
			t.Errorf("plain-id licenseDeclared = %q, want MIT", p.LicenseDeclared)
		}
	}

	// A LicenseRef-* identifier must be declared before it is referenced.
	if len(doc.HasExtractedLicensingInfos) != 1 {
		t.Fatalf("hasExtractedLicensingInfos = %+v, want the one LicenseRef", doc.HasExtractedLicensingInfos)
	}
	if got := doc.HasExtractedLicensingInfos[0].LicenseID; got != "LicenseRef-Acme-Internal" {
		t.Errorf("extracted license id = %q", got)
	}
	if doc.HasExtractedLicensingInfos[0].ExtractedText == "" {
		t.Error("extractedText is required and must not be empty")
	}
}

// TestLicenseContentIsDeterministic keeps a committed SBOM diffable: rendering
// the same input twice must not change what it says about licenses.
//
// It compares content rather than whole documents, because a CycloneDX BOM
// carries a fresh serialNumber and timestamp per render by design — each
// document is a distinct instance, and asserting otherwise would be asserting
// against the format.
func TestLicenseContentIsDeterministic(t *testing.T) {
	strip := func(s string) string {
		var kept []string
		for _, line := range strings.Split(s, "\n") {
			t := strings.TrimSpace(line)
			if strings.HasPrefix(t, `"serialNumber"`) || strings.HasPrefix(t, `"timestamp"`) ||
				strings.HasPrefix(t, `"created"`) || strings.HasPrefix(t, `"documentNamespace"`) {
				continue
			}
			kept = append(kept, line)
		}
		return strings.Join(kept, "\n")
	}

	for _, format := range []string{FormatCycloneDxJSON, FormatSpdxJSON} {
		first := strip(renderTo(t, format))
		for i := 0; i < 3; i++ {
			if strip(renderTo(t, format)) != first {
				t.Errorf("%s content is not stable across renders", format)
				break
			}
		}
		// The licenses themselves must be in there, or the comparison is vacuous.
		if !strings.Contains(first, "MIT") {
			t.Errorf("%s output carries no license content to compare", format)
		}
	}
}
