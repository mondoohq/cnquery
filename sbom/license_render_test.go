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

// The model extension exists for one case: a package whose own manifest
// declares one license while the files it ships say another. Flattening the two
// into a single value is what the scalar forced, and it is the error that
// matters most in a compliance document — it asserts a grant the shipped code
// does not make.

func splitLicenseBom() *Sbom {
	return &Sbom{
		Generator: &Generator{Vendor: "Mondoo, Inc", Name: "test", Version: "1"},
		Asset:     &Asset{Name: "test-asset", Platform: &Platform{Name: "linux", Version: "1"}},
		Packages: []*Package{{
			Name: "disagrees", Version: "1.0.0", Purl: "pkg:npm/disagrees@1.0.0",
			// The scalar stays populated, as the migration contract requires.
			License: "MIT",
			Licenses: []*License{
				{SpdxId: "MIT", Acquisition: LicenseAcquisition_LICENSE_ACQUISITION_DECLARED, Confidence: 1},
				{
					SpdxId:      "AGPL-3.0-only",
					Acquisition: LicenseAcquisition_LICENSE_ACQUISITION_CONCLUDED,
					Confidence:  0.98,
					Location:    "node_modules/disagrees/LICENSE",
				},
			},
			Copyright: []string{"Copyright (c) 2019 Example Corp"},
			Supplier:  "Example Corp",
		}},
	}
}

func renderBom(t *testing.T, format string, bom *Sbom) string {
	t.Helper()
	var b strings.Builder
	h := New(format)
	if h == nil {
		t.Fatalf("no handler for %q", format)
	}
	if err := h.Render(&b, bom); err != nil {
		t.Fatalf("render %s: %v", format, err)
	}
	return b.String()
}

func TestCycloneDXSeparatesDeclaredFromConcluded(t *testing.T) {
	var doc struct {
		Components []struct {
			Name     string `json:"name"`
			Licenses []struct {
				License    *struct{ ID, Name string } `json:"license"`
				Expression string                     `json:"expression"`
			} `json:"licenses"`
			Copyright string `json:"copyright"`
			Supplier  *struct {
				Name string `json:"name"`
			} `json:"supplier"`
			Evidence *struct {
				Licenses []struct {
					License *struct{ ID, Name string } `json:"license"`
				} `json:"licenses"`
				Copyright []struct {
					Text string `json:"text"`
				} `json:"copyright"`
			} `json:"evidence"`
		} `json:"components"`
	}
	if err := json.Unmarshal([]byte(renderBom(t, "cyclonedx-json", splitLicenseBom())), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// The document also carries the asset's platform component; the package is
	// the one under test.
	var c = doc.Components[0]
	found := false
	for _, comp := range doc.Components {
		if comp.Name == "disagrees" {
			c, found = comp, true
		}
	}
	if !found {
		t.Fatalf("no component named disagrees in %d components", len(doc.Components))
	}

	// Declared is an assertion the package makes about itself.
	if len(c.Licenses) != 1 || c.Licenses[0].License == nil || c.Licenses[0].License.ID != "MIT" {
		t.Errorf("licenses = %+v, want the declared MIT", c.Licenses)
	}
	// Concluded is evidence: it was read out of a file rather than stated.
	if c.Evidence == nil || len(c.Evidence.Licenses) != 1 ||
		c.Evidence.Licenses[0].License == nil || c.Evidence.Licenses[0].License.ID != "AGPL-3.0-only" {
		t.Errorf("evidence.licenses = %+v, want the concluded AGPL-3.0-only", c.Evidence)
	}
	if c.Copyright != "Copyright (c) 2019 Example Corp" {
		t.Errorf("copyright = %q", c.Copyright)
	}
	if c.Evidence == nil || len(c.Evidence.Copyright) != 1 {
		t.Errorf("evidence.copyright = %+v, want the statement that was found", c.Evidence)
	}
	if c.Supplier == nil || c.Supplier.Name != "Example Corp" {
		t.Errorf("supplier = %+v", c.Supplier)
	}
}

func TestSPDXConcludedIsNotAnEchoOfDeclared(t *testing.T) {
	out := renderBom(t, "spdx-json", splitLicenseBom())
	var doc struct {
		Packages []struct {
			LicenseDeclared  string `json:"licenseDeclared"`
			LicenseConcluded string `json:"licenseConcluded"`
			CopyrightText    string `json:"copyrightText"`
			Supplier         string `json:"supplier"`
		} `json:"packages"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(doc.Packages) != 1 {
		t.Fatalf("packages = %d, want 1", len(doc.Packages))
	}
	p := doc.Packages[0]
	if p.LicenseDeclared != "MIT" {
		t.Errorf("licenseDeclared = %q, want MIT", p.LicenseDeclared)
	}
	if p.LicenseConcluded != "AGPL-3.0-only" {
		t.Errorf("licenseConcluded = %q, want AGPL-3.0-only — the shipped text is the grant", p.LicenseConcluded)
	}
	if p.CopyrightText != "Copyright (c) 2019 Example Corp" {
		t.Errorf("copyrightText = %q", p.CopyrightText)
	}
	if !strings.Contains(p.Supplier, "Example Corp") {
		t.Errorf("supplier = %q", p.Supplier)
	}
}

// TestLegacyScalarStillRenders is the migration contract: a producer that has
// not adopted the structured list renders exactly as it did before.
func TestLegacyScalarStillRenders(t *testing.T) {
	cdx := renderTo(t, "cyclonedx-json")
	for _, want := range []string{`"MIT"`, `"MIT OR Apache-2.0"`, `"BSD-like, see LICENSE"`} {
		if !strings.Contains(cdx, want) {
			t.Errorf("cyclonedx output lost %s", want)
		}
	}
	spdxOut := renderTo(t, "spdx-json")
	if !strings.Contains(spdxOut, "LicenseRef-Acme-Internal") {
		t.Error("spdx output lost the LicenseRef identifier")
	}
}

// TestSPDXConcludedFallsBackToDeclared: with nothing concluded, echoing the
// declared value is the honest reading — but it must be the declared value,
// not NOASSERTION, or a document loses a license it was told about.
func TestSPDXConcludedFallsBackToDeclared(t *testing.T) {
	out := renderTo(t, "spdx-json")
	var doc struct {
		Packages []struct {
			Name             string `json:"name"`
			LicenseDeclared  string `json:"licenseDeclared"`
			LicenseConcluded string `json:"licenseConcluded"`
		} `json:"packages"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, p := range doc.Packages {
		if p.Name == "plain-id" && (p.LicenseDeclared != "MIT" || p.LicenseConcluded != "MIT") {
			t.Errorf("plain-id: declared=%q concluded=%q, want MIT for both", p.LicenseDeclared, p.LicenseConcluded)
		}
		if p.Name == "undeclared" && p.LicenseConcluded != "NOASSERTION" {
			t.Errorf("undeclared: concluded=%q, want NOASSERTION", p.LicenseConcluded)
		}
	}
}
