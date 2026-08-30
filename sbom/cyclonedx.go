// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"errors"
	"io"
	"strings"
	"time"

	cyclonedx "github.com/CycloneDX/cyclonedx-go"
	"github.com/google/uuid"
	"github.com/package-url/packageurl-go"
)

func NewCycloneDX(format string) *CycloneDX {
	switch format {
	case FormatCycloneDxXML:
		return &CycloneDX{
			Format: cyclonedx.BOMFileFormatXML,
		}
	default:
		return &CycloneDX{
			Format: cyclonedx.BOMFileFormatJSON,
		}
	}
}

var _ Decoder = &CycloneDX{}

type CycloneDX struct {
	opts   renderOpts
	Format cyclonedx.BOMFileFormat
}

func (ccx *CycloneDX) convertToCycloneDx(bom *Sbom) (*cyclonedx.BOM, error) {
	sbom := cyclonedx.NewBOM()
	sbom.SerialNumber = uuid.New().URN()
	sbom.Metadata = &cyclonedx.Metadata{
		Timestamp: time.Now().Format(time.RFC3339),
		Tools: &cyclonedx.ToolsChoice{
			Components: &[]cyclonedx.Component{
				{
					Type:    cyclonedx.ComponentTypeApplication,
					Author:  bom.Generator.Vendor,
					Name:    bom.Generator.Name,
					Version: bom.Generator.Version,
				},
			},
		},
		Component: &cyclonedx.Component{
			// Deterministic (was a per-render UUID) so the document's root
			// component id is stable across renders.
			BOMRef: "root:" + bom.Asset.Name,
			// TODO: understand the device type
			// Type: cyclonedx.ComponentTypeContainer,
			Type: cyclonedx.ComponentTypeDevice,
			Name: bom.Asset.Name,
		},
	}

	components := []cyclonedx.Component{}
	// emitted tracks the component bom-refs already added, so a document never
	// contains a duplicate bom-ref (invalid CycloneDX) when two packages share a
	// purl, and so the dependency graph below can drop edges to absent components.
	emitted := map[string]bool{}

	// add os as component
	cpe := ""
	if len(bom.Asset.Platform.Cpes) > 0 {
		cpe = bom.Asset.Platform.Cpes[0]
	}

	osRef := "os:" + bom.Asset.Platform.Name
	emitted[osRef] = true
	components = append(components, cyclonedx.Component{
		BOMRef:  osRef,
		Type:    cyclonedx.ComponentTypeOS,
		Name:    bom.Asset.Platform.Name,
		Version: bom.Asset.Platform.Version,
		CPE:     cpe,
	})

	// add os packages as components
	for i := range bom.Packages {
		pkg := bom.Packages[i]
		cpe := ""
		if len(pkg.Cpes) > 0 && ccx.opts.IncludeCPE {
			cpe = pkg.Cpes[0]
		}

		fileLocations := []cyclonedx.EvidenceOccurrence{}

		// pkg.Location is deprecated, use pkg.Evidences instead
		if pkg.Location != "" {
			fileLocations = append(fileLocations, cyclonedx.EvidenceOccurrence{
				Location: pkg.Location,
			})
		}

		if pkg.EvidenceList != nil && ccx.opts.IncludeEvidence {
			for i := range pkg.EvidenceList {
				e := pkg.EvidenceList[i]
				if e.Type == EvidenceType_EVIDENCE_TYPE_FILE {
					fileLocations = append(fileLocations, cyclonedx.EvidenceOccurrence{
						Location: e.Value,
					})
				}
			}
		}

		var evidence *cyclonedx.Evidence
		if len(fileLocations) > 0 {
			evidence = &cyclonedx.Evidence{
				Occurrences: &fileLocations,
			}
		}

		ref := BomRefFor(pkg)
		if emitted[ref] {
			// Same purl already emitted (a package present at multiple install
			// locations) — dedup so bom-refs stay unique.
			continue
		}
		emitted[ref] = true

		// Concluded licenses and copyright are evidence, not assertion: they were
		// read out of the files the package ships rather than stated by it, and
		// CycloneDX has a place that says exactly that.
		if concluded := cycloneDXLicenseList(concludedLicenses(pkg)); concluded != nil {
			if evidence == nil {
				evidence = &cyclonedx.Evidence{}
			}
			evidence.Licenses = concluded
		}
		if len(pkg.Copyright) > 0 {
			if evidence == nil {
				evidence = &cyclonedx.Evidence{}
			}
			copyrights := make([]cyclonedx.Copyright, 0, len(pkg.Copyright))
			for _, c := range pkg.Copyright {
				copyrights = append(copyrights, cyclonedx.Copyright{Text: c})
			}
			evidence.Copyright = &copyrights
		}

		bomPkg := cyclonedx.Component{
			BOMRef:      ref,
			Type:        cyclonedx.ComponentTypeLibrary,
			Name:        pkg.Name,
			Version:     pkg.Version,
			PackageURL:  pkg.Purl,
			CPE:         cpe,
			Evidence:    evidence,
			Description: pkg.Description,
			Licenses:    cycloneDXDeclared(pkg),
			Copyright:   strings.Join(pkg.Copyright, "\n"),
		}
		if pkg.Supplier != "" {
			bomPkg.Supplier = &cyclonedx.OrganizationalEntity{Name: pkg.Supplier}
		}
		if pkg.Scope == PackageScopeDev {
			bomPkg.Scope = cyclonedx.ScopeExcluded
		}
		if len(pkg.Hashes) > 0 {
			hashes := make([]cyclonedx.Hash, 0, len(pkg.Hashes))
			for _, h := range pkg.Hashes {
				hashes = append(hashes, cyclonedx.Hash{
					Algorithm: cyclonedx.HashAlgorithm(h.Alg),
					Value:     h.Value,
				})
			}
			bomPkg.Hashes = &hashes
		}

		components = append(components, bomPkg)
	}

	sbom.Components = &components

	// Emit the package→package dependency graph (CycloneDX `dependencies`), each
	// endpoint referenced by the component bom-ref set above.
	if len(bom.Dependencies) > 0 {
		deps := make([]cyclonedx.Dependency, 0, len(bom.Dependencies))
		for _, d := range bom.Dependencies {
			// Skip edges whose source isn't an emitted component, and prune
			// targets that aren't — a document must not reference an absent
			// bom-ref (matches the SPDX renderer's defensive behavior).
			if !emitted[d.Ref] {
				continue
			}
			dependsOn := make([]string, 0, len(d.DependencyRefs))
			for _, r := range d.DependencyRefs {
				if emitted[r] {
					dependsOn = append(dependsOn, r)
				}
			}
			deps = append(deps, cyclonedx.Dependency{
				Ref:          d.Ref,
				Dependencies: &dependsOn,
			})
		}
		sbom.Dependencies = &deps
	}

	return sbom, nil
}

func (s *CycloneDX) ApplyOptions(opts ...renderOption) {
	for _, opt := range opts {
		opt(&s.opts)
	}
}

func (ccx *CycloneDX) Convert(bom *Sbom) (any, error) {
	return ccx.convertToCycloneDx(bom)
}

func (ccx *CycloneDX) Render(w io.Writer, bom *Sbom) error {
	sbom, err := ccx.convertToCycloneDx(bom)
	if err != nil {
		return err
	}
	enc := cyclonedx.NewBOMEncoder(w, ccx.Format)
	enc.SetPretty(true)
	return enc.Encode(sbom)
}

func (ccx *CycloneDX) Parse(r io.ReadSeeker) (*Sbom, error) {
	doc := &cyclonedx.BOM{
		Components: &[]cyclonedx.Component{},
	}
	err := cyclonedx.NewBOMDecoder(r, ccx.Format).Decode(doc)
	if err != nil {
		return nil, err
	}

	return ccx.convertCycloneDxToSbom(doc)
}

func (ccx *CycloneDX) convertCycloneDxToSbom(bom *cyclonedx.BOM) (*Sbom, error) {
	if bom == nil {
		return nil, nil
	}

	// check if the BOM is empty
	if bom.Metadata == nil || bom.Metadata.Component == nil || bom.Components == nil {
		return nil, errors.New("not a valid cyclone dx BOM")
	}

	rootComponent := bom.Metadata.Component
	title := rootComponent.Description
	version := rootComponent.Version
	if title == "" {
		title = "CycloneDX"
	}
	if version == "" {
		version = bom.SpecVersion.String()
	}
	sbom := &Sbom{
		Asset: &Asset{
			Name: rootComponent.Name,
			Platform: &Platform{
				Name:    "cyclonedx",
				Version: version,
				Title:   title,
			},
		},
		Packages: make([]*Package, 0),
	}

	switch rootComponent.Type {
	case cyclonedx.ComponentTypeOS:
		hostnameId := "//platformid.api.mondoo.app/hostname/" + rootComponent.Name
		sbom.Asset.PlatformIds = append(sbom.Asset.PlatformIds, hostnameId)
	case cyclonedx.ComponentTypeContainer:
		// we need to figure out where to get the container ID from properly. For now, we use the BOMRef
		bomRefId := "//platformid.api.mondoo.app/runtime/docker/images/" + rootComponent.BOMRef
		sbom.Asset.PlatformIds = append(sbom.Asset.PlatformIds, bomRefId)
	}

	if bom.Metadata.Tools != nil {
		if bom.Metadata.Tools.Components != nil {
			// last one wins :-) - we only support one tool
			for _, component := range *bom.Metadata.Tools.Components {
				sbom.Generator = &Generator{
					Name:    component.Name,
					Version: component.Version,
					Vendor:  component.Author,
				}
			}
		}

		// if we have no generator info, fallback to trying tools. these are deprecated
		// but might still be present
		if sbom.Generator == nil && bom.Metadata.Tools.Tools != nil {
			for _, tool := range *bom.Metadata.Tools.Tools {
				sbom.Generator = &Generator{
					Name:    tool.Name,
					Version: tool.Version,
					Vendor:  tool.Vendor,
				}
			}
		}
	}

	for _, component := range *bom.Components {
		pkg := &Package{
			Name:        component.Name,
			Version:     component.Version,
			Purl:        component.PackageURL,
			Description: component.Description,
		}

		// parse purl to gather package type
		if component.PackageURL != "" {
			url, err := packageurl.FromString(component.PackageURL)
			if err == nil {
				pkg.Type = url.Type
			}
		}

		if component.CPE != "" {
			pkg.Cpes = []string{component.CPE}
		}

		if component.Evidence != nil && component.Evidence.Occurrences != nil && ccx.opts.IncludeEvidence {
			pkg.EvidenceList = make([]*Evidence, 0)
			for _, e := range *component.Evidence.Occurrences {
				pkg.EvidenceList = append(pkg.EvidenceList, &Evidence{
					Type:  EvidenceType_EVIDENCE_TYPE_FILE,
					Value: e.Location,
				})
			}
		}

		switch component.Type {
		case cyclonedx.ComponentTypeOS:
			sbom.Asset.Platform.Name = component.Name
			sbom.Asset.Platform.Version = component.Version
			sbom.Asset.Platform.Title = component.Description
			sbom.Asset.Platform.Family = familyMap[strings.ToLower(component.Name)]
			if len(component.CPE) > 0 {
				sbom.Asset.Platform.Cpes = []string{component.CPE}
			}
			sbom.Packages = append(sbom.Packages, pkg)
		case cyclonedx.ComponentTypeLibrary:
			sbom.Packages = append(sbom.Packages, pkg)
		case cyclonedx.ComponentTypeApplication:
			sbom.Packages = append(sbom.Packages, pkg)
		}
	}

	return sbom, nil
}

var familyMap = map[string][]string{
	"windows": {"windows", "os"},
	"macos":   {"darwin", "bsd", "unix", "os"},
	"debian":  {"linux", "unix", "os"},
	"ubuntu":  {"linux", "unix", "os"},
	"centos":  {"linux", "unix", "os"},
	"alpine":  {"linux", "unix", "os"},
	"fedora":  {"linux", "unix", "os"},
	"rhel":    {"linux", "unix", "os"},
}

// cycloneDXDeclared renders the licenses a package declares, preferring the
// structured list and falling back to the legacy scalar.
//
// The fallback is what lets consumers migrate without a flag day: a producer
// that has not yet populated Licenses still renders exactly as it did before.
func cycloneDXDeclared(pkg *Package) *cyclonedx.Licenses {
	if l := cycloneDXLicenseList(declaredLicenses(pkg)); l != nil {
		return l
	}
	return cycloneDXLicenses(pkg.License)
}

// declaredLicenses returns the package's declared entries. An entry whose
// acquisition the producer did not set counts as declared: it is what the
// scalar always meant, so an unset value keeps the old reading rather than
// silently demoting a license to evidence.
func declaredLicenses(pkg *Package) []*License {
	var out []*License
	for _, l := range pkg.Licenses {
		if l.GetAcquisition() != LicenseAcquisition_LICENSE_ACQUISITION_CONCLUDED {
			out = append(out, l)
		}
	}
	return out
}

// concludedLicenses returns the entries read from the files a package ships.
func concludedLicenses(pkg *Package) []*License {
	var out []*License
	for _, l := range pkg.Licenses {
		if l.GetAcquisition() == LicenseAcquisition_LICENSE_ACQUISITION_CONCLUDED {
			out = append(out, l)
		}
	}
	return out
}

// cycloneDXLicenseList renders license entries into the CycloneDX shape each
// one's populated field calls for, dropping entries that carry no value.
//
// The three shapes are mutually exclusive in the schema, which is why the
// producer says which kind it has rather than the renderer guessing: a value
// already known to be an expression must not be emitted as an id.
func cycloneDXLicenseList(licenses []*License) *cyclonedx.Licenses {
	out := make(cyclonedx.Licenses, 0, len(licenses))
	for _, l := range licenses {
		switch {
		case strings.TrimSpace(l.GetExpression()) != "":
			out = append(out, cyclonedx.LicenseChoice{Expression: strings.TrimSpace(l.GetExpression())})
		case strings.TrimSpace(l.GetSpdxId()) != "":
			out = append(out, cyclonedx.LicenseChoice{
				License: &cyclonedx.License{ID: strings.TrimSpace(l.GetSpdxId())},
			})
		case strings.TrimSpace(l.GetName()) != "":
			out = append(out, cyclonedx.LicenseChoice{
				License: &cyclonedx.License{Name: strings.TrimSpace(l.GetName())},
			})
		}
	}
	if len(out) == 0 {
		return nil
	}
	return &out
}

// cycloneDXLicenses renders a package's declared license as a CycloneDX license
// list, or nil when there is none.
//
// CycloneDX models a license three mutually exclusive ways, and choosing the
// wrong one fails schema validation:
//
//   - expression, for a full SPDX expression such as "MIT OR Apache-2.0";
//   - license.id, for a bare SPDX identifier;
//   - license.name, for anything that is neither.
//
// Package license strings in the wild are all three, so the shape is decided
// per value rather than assumed. A string containing an SPDX operator is an
// expression; one that looks like a bare identifier is an id; everything else —
// "BSD-like", "see LICENSE" — is a name, which is honest rather than lossy.
func cycloneDXLicenses(license string) *cyclonedx.Licenses {
	license = strings.TrimSpace(license)
	if license == "" {
		return nil
	}
	if isSPDXExpression(license) {
		return &cyclonedx.Licenses{{Expression: license}}
	}
	if isSPDXIdentifierShaped(license) {
		return &cyclonedx.Licenses{{License: &cyclonedx.License{ID: license}}}
	}
	return &cyclonedx.Licenses{{License: &cyclonedx.License{Name: license}}}
}

// isSPDXExpression reports whether a license string joins operands with SPDX
// operators, which is what makes it an expression rather than an identifier.
//
// The operators are matched case-SENSITIVELY, because that is what the grammar
// defines: SPDX spells them as upper-case keywords, so a license whose *name*
// contains the ordinary English word — "Sleepycat and others" — is not a
// conjunction. Upper-casing before the comparison read that as an expression
// and emitted it as one, which is not parseable SPDX and is exactly the
// malformed document the three mutually exclusive fields exist to prevent.
//
// Parentheses are deliberately not a signal on their own either. "BSD (see
// LICENSE)" is a free-text name, and grouping only ever exists to order
// operators, so a genuinely grouped expression contains one anyway and is
// caught above.
func isSPDXExpression(s string) bool {
	for _, t := range strings.Fields(s) {
		switch t {
		case "AND", "OR", "WITH":
			return true
		}
	}
	return false
}

// isSPDXIdentifierShaped reports whether a license string could be a bare SPDX
// identifier: one token of the characters identifiers actually use.
//
// It deliberately does not check the value against the SPDX list. This package
// renders whatever the extractors found and must not silently drop a license it
// does not recognize; an unlisted identifier still belongs in the document, and
// a consumer validating against the SPDX list will say so more usefully than a
// renderer that omitted it.
func isSPDXIdentifierShaped(s string) bool {
	if s == "" || strings.ContainsAny(s, " \t") {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-' || r == '.' || r == '+' || r == '_':
		default:
			return false
		}
	}
	return true
}
