// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/package-url/packageurl-go"
	"github.com/spdx/tools-golang/convert"
	"github.com/spdx/tools-golang/spdx"
	"github.com/spdx/tools-golang/spdx/v2/common"
	"github.com/spdx/tools-golang/spdx/v2/v2_1"
	"github.com/spdx/tools-golang/spdx/v2/v2_2"
	"github.com/spdx/tools-golang/spdx/v2/v2_3"
	"github.com/spdx/tools-golang/tagvalue"
)

func NewSPDX(format string) *Spdx {
	return &Spdx{
		Version: "2.3",
		Format:  format,
	}
}

var _ Decoder = &Spdx{}

type Spdx struct {
	opts    renderOpts
	Version string
	Format  string
}

func (s *Spdx) ApplyOptions(opts ...renderOption) {
	for _, opt := range opts {
		opt(&s.opts)
	}
}

func (s *Spdx) convertToSpdx(bom *Sbom) *spdx.Document {
	extractedLicenses := extractedLicenseSet{}
	doc := &spdx.Document{
		SPDXVersion:                spdx.Version,
		SPDXIdentifier:             "DOCUMENT",
		ExternalDocumentReferences: nil,
		DocumentComment:            "",

		CreationInfo: &spdx.CreationInfo{
			Creators: []spdx.Creator{
				{
					Creator:     bom.Generator.Vendor,
					CreatorType: "Organization",
				},
				{
					Creator:     bom.Generator.Name + "-" + bom.Generator.Version,
					CreatorType: "Tool",
				},
			},
			Created: time.Now().UTC().Format(time.RFC3339),
		},
	}

	// bom_ref → SPDX element id, so the dependency graph (which references
	// components by bom_ref) can be translated to SPDX DEPENDS_ON relationships.
	refToID := map[string]spdx.ElementID{}
	// emitted dedups by bom-ref: a package present at multiple install locations
	// shares a purl, so emit it once (matching the CycloneDX renderer). Without
	// this, duplicate SPDX packages are written and refToID is silently
	// overwritten (with a possibly different id, since Hash() may differ).
	emitted := map[string]bool{}

	for i := range bom.Packages {
		pkg := bom.Packages[i]

		ref := BomRefFor(pkg)
		if emitted[ref] {
			continue
		}
		emitted[ref] = true

		refs := []*spdx.PackageExternalReference{}

		if len(pkg.Cpes) > 0 {
			for _, cpe := range pkg.Cpes {
				refs = append(refs, &spdx.PackageExternalReference{
					RefType:  spdx.SecurityCPE23Type,
					Category: spdx.CategorySecurity,
					Locator:  cpe,
				})
			}
		}

		if pkg.Purl != "" {
			refs = append(refs, &spdx.PackageExternalReference{
				RefType:  spdx.PackageManagerPURL,
				Category: spdx.CategoryPackageManager,
				Locator:  pkg.Purl,
			})
		}

		id := NewSPDXPackageID(pkg)
		refToID[ref] = id

		// Integrity digests → SPDX checksums. Alg is CycloneDX spelling
		// ("SHA-512"); SPDX uses the dash-free form ("SHA512").
		var checksums []common.Checksum
		for _, h := range pkg.Hashes {
			checksums = append(checksums, common.Checksum{
				Algorithm: common.ChecksumAlgorithm(strings.ReplaceAll(h.Alg, "-", "")),
				Value:     h.Value,
			})
		}

		// The SPDX specification requires NOASSERTION rather than an empty value
		// for the license and copyright fields. An empty string is not valid
		// SPDX, and a strict consumer may reject the whole document over it.
		//
		// SPDX draws exactly the distinction the license model does: *declared*
		// is what the package says about itself, *concluded* is what this
		// document asserts after looking. They differ precisely when it matters
		// — a package declaring MIT while shipping AGPL-3.0-only — so where a
		// concluded license was determined it is emitted as itself rather than
		// echoing the declared one.
		declared := spdxNoAssertion(spdxLicense(declaredLicenses(pkg), pkg.License, extractedLicenses))
		concluded := spdxNoAssertion(spdxLicense(concludedLicenses(pkg), "", extractedLicenses))
		if concluded == spdxNoAssertionValue {
			// Nothing was concluded. Echoing the declared value is the honest
			// reading: it is what the document has to go on, and asserting more
			// than was determined is what NOASSERTION exists to avoid.
			concluded = declared
		}
		doc.Packages = append(doc.Packages, &spdx.Package{
			PackageSPDXIdentifier:     id,
			PackageName:               pkg.Name,
			PackageVersion:            pkg.Version,
			PackageLicenseDeclared:    declared,
			PackageLicenseConcluded:   concluded,
			PackageCopyrightText:      spdxNoAssertion(strings.Join(pkg.Copyright, "\n")),
			PackageLicenseComments:    spdxLicenseComments(concludedLicenses(pkg)),
			PackageSupplier:           spdxSupplier(pkg.Supplier),
			PackageDescription:        pkg.Description,
			PackageExternalReferences: refs,
			PackageFileName:           pkg.Location,
			PackageChecksums:          checksums,
		})
		// A LicenseRef-* identifier is not on the SPDX license list, so the
		// specification requires the document to define it before it may be
		// referenced.
		extractedLicenses.add(declared)
		extractedLicenses.add(concluded)
	}

	doc.OtherLicenses = extractedLicenses.render()

	// Emit the package→package dependency graph as SPDX DEPENDS_ON relationships,
	// skipping any edge whose endpoints are not in this document.
	for _, dep := range bom.Dependencies {
		fromID, ok := refToID[dep.Ref]
		if !ok {
			continue
		}
		for _, to := range dep.DependencyRefs {
			toID, ok := refToID[to]
			if !ok {
				continue
			}
			doc.Relationships = append(doc.Relationships, &spdx.Relationship{
				RefA:         common.MakeDocElementID("", string(fromID)),
				RefB:         common.MakeDocElementID("", string(toID)),
				Relationship: spdx.RelationshipDependsOn,
			})
		}
	}

	return doc
}

var expr = regexp.MustCompile("[^a-zA-Z0-9.-]")

// NewSPDXPackageID creates a new SPDX ID for a package
// see https://spdx.github.io/spdx-spec/v2.3/relationships-between-SPDX-elements/
func NewSPDXPackageID(pkg *Package) spdx.ElementID {
	hash, _ := pkg.Hash()

	id := fmt.Sprintf("Package-%s-%s-%s", pkg.Type, pkg.Name, hash)
	// Scrub anything outside the SPDX ID charset. This must assign the result
	// back: an unsanitized name (e.g. one containing a newline) would otherwise
	// inject tags into SPDX tag-value output.
	id = expr.ReplaceAllString(id, "-")
	return spdx.ElementID(id)
}

func (s *Spdx) Convert(bom *Sbom) (any, error) {
	spdxLatestBom := s.convertToSpdx(bom)

	var spdxBom any
	var err error
	switch s.Version {
	case "2.1":
		doc := v2_1.Document{}
		err = convert.Document(spdxLatestBom, &doc)
		spdxBom = doc
	case "2.2":
		doc := v2_2.Document{}
		err = convert.Document(spdxLatestBom, &doc)
		spdxBom = doc
	case "2.3":
		fallthrough
	case "":
		doc := v2_3.Document{}
		err = convert.Document(spdxLatestBom, &doc)
		spdxBom = doc
	default:
		return nil, fmt.Errorf("unsupported SPDX version %q", s.Version)
	}

	if err != nil {
		return nil, fmt.Errorf("unable to convertToCycloneDx SBOM to SPDX document: %w", err)
	}
	return spdxBom, nil
}

func (s *Spdx) Render(w io.Writer, bom *Sbom) error {
	spdxBom, err := s.Convert(bom)
	if err != nil {
		return err
	}

	switch s.Format {
	case FormatSpdxTagValue:
		err = tagvalue.Write(spdxBom, w)
		if err != nil {
			return fmt.Errorf("unable to write SPDX tag-value document: %w", err)
		}
		return nil
	case FormatSpdxJSON:
		fallthrough
	default:
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		return enc.Encode(spdxBom)
	}
}

func (s *Spdx) Parse(r io.ReadSeeker) (*Sbom, error) {
	// try to parse all supported SPDX format
	switch s.Format {
	case FormatSpdxTagValue:
		doc, err := tagvalue.Read(r)
		if err == nil && doc.SPDXVersion != "" {
			return s.convertToSbom(doc), nil
		}
	case FormatSpdxJSON:
		var doc spdx.Document
		err := json.NewDecoder(r).Decode(&doc)
		if err == nil && doc.SPDXVersion != "" {
			return s.convertToSbom(&doc), nil
		}
	}

	return nil, errors.New("unable to parse SPDX document")
}

func (s *Spdx) convertToSbom(doc *spdx.Document) *Sbom {
	bom := &Sbom{
		Generator: &Generator{
			Name: doc.CreationInfo.Creators[0].Creator,
		},
		Asset: &Asset{
			Name: doc.DocumentName,
			Platform: &Platform{
				Name:    "spdx",
				Version: doc.SPDXVersion,
				Title:   "SPDX",
			},
		},
		Packages: []*Package{},
	}

	name := ""
	var pf *Platform

	for i := range doc.Packages {
		pkg := doc.Packages[i]

		bomPkg := &Package{
			Name:        pkg.PackageName,
			Version:     pkg.PackageVersion,
			Description: pkg.PackageDescription,
			Location:    pkg.PackageFileName,
			Type:        "", // extract package type from purl, see below
			Purl:        "", // extract package type from purl, see below
			Cpes:        []string{},
		}

		for _, ref := range pkg.PackageExternalReferences {
			if ref.RefType == spdx.PackageManagerPURL {
				bomPkg.Purl = ref.Locator
				pkgUrl, err := packageurl.FromString(ref.Locator)
				if err == nil {
					bomPkg.Type = pkgUrl.Type

					// extract distro information
					m := pkgUrl.Qualifiers.Map()
					distroVal, ok := m["distro"]
					if ok {
						if pf == nil {
							pf = &Platform{}
						}
						name = distroVal
						pf.Title = distroVal
						// The qualifier is the platform name and version
						// joined with a dash (purl.NewPlatformPurl), and it is
						// the *name* that commonly carries dashes of its own:
						// opensuse-leap, opensuse-tumbleweed, opensuse-microos.
						// Splitting on the first dash reports
						// "opensuse-leap-15.6" as name "opensuse" version
						// "leap", so split on the last one. A qualifier with no
						// dash carries no version, e.g. "arch" rather than
						// "arch-rolling".
						pf.Name = distroVal
						if i := strings.LastIndex(distroVal, "-"); i > 0 {
							pf.Name = distroVal[:i]
							pf.Version = distroVal[i+1:]
						}
						pf.Family = familyMap[pf.Name]
					}
					arch, ok := m["arch"]
					if ok && pf != nil {
						pf.Arch = arch
					}
				}
			}
			if ref.RefType == spdx.SecurityCPE23Type {
				bomPkg.Cpes = append(bomPkg.Cpes, ref.Locator)
			}
		}

		if pkg.PackageFileName != "" && s.opts.IncludeEvidence {
			bomPkg.EvidenceList = append(bomPkg.EvidenceList, &Evidence{
				Type:  EvidenceType_EVIDENCE_TYPE_FILE,
				Value: pkg.PackageFileName,
			})
		}

		bom.Packages = append(bom.Packages, bomPkg)
	}

	if name != "" {
		bom.Asset.Name = name
	}
	if pf != nil {
		bom.Asset.Platform = pf
	}

	return bom
}

// spdxNoAssertion returns the SPDX-mandated NOASSERTION for an empty value.
func spdxNoAssertion(v string) string {
	if strings.TrimSpace(v) == "" {
		return spdxNoAssertionValue
	}
	return v
}

const spdxNoAssertionValue = "NOASSERTION"

// licenseRefPrefix marks a custom identifier that is not on the SPDX license
// list.
const licenseRefPrefix = "LicenseRef-"

// extractedLicenseSet collects the LicenseRef-* identifiers a document
// references, so each can be declared in hasExtractedLicensingInfos as the
// specification requires.
//
// The value is the human-readable name the identifier stands for, which matters
// for a reference synthesized from a free-form license string: "see LICENSE" is
// not a legal identifier, so it becomes LicenseRef-see-LICENSE, and the original
// text is the only place the reader learns what that means. Empty means the
// identifier is all that is known and names itself.
type extractedLicenseSet map[string]string

// add registers every LicenseRef-* identifier appearing in a rendered license
// expression. It never overwrites a name already recorded for an identifier, so
// a reference synthesized by addNamed keeps its original text when the rendered
// expression it landed in is scanned again.
func (s extractedLicenseSet) add(license string) {
	for _, tok := range strings.FieldsFunc(license, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '(' || r == ')'
	}) {
		if !strings.HasPrefix(tok, licenseRefPrefix) {
			continue
		}
		if _, ok := s[tok]; !ok {
			s[tok] = ""
		}
	}
}

// addNamed registers an identifier together with the text it was synthesized
// from.
func (s extractedLicenseSet) addNamed(id, name string) {
	if existing, ok := s[id]; ok && existing != "" {
		return
	}
	s[id] = name
}

// render returns the collected identifiers as SPDX other-license entries, sorted
// so a document is byte-stable across runs.
func (s extractedLicenseSet) render() []*spdx.OtherLicense {
	if len(s) == 0 {
		return nil
	}
	ids := make([]string, 0, len(s))
	for id := range s {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	out := make([]*spdx.OtherLicense, 0, len(ids))
	for _, id := range ids {
		name := s[id]
		if name == "" {
			name = id
		}
		out = append(out, &spdx.OtherLicense{
			LicenseIdentifier: id,
			// The text is not available here — the extractors carry an
			// identifier, not a license body — so the field states that rather
			// than inventing one.
			ExtractedText: spdxNoAssertionValue,
			LicenseName:   name,
		})
	}
	return out
}

// spdxLicense renders license entries as the single value an SPDX field holds,
// falling back to a legacy scalar when the structured list is empty.
//
// SPDX carries one expression per field, so several entries are joined with
// AND: a package under two licenses is under both, and OR would grant a choice
// the producer never reported.
//
// The field must hold a license *expression*, which constrains what each operand
// may be, so the three License shapes do not all reach it the same way. Refs
// synthesized for a free-form name are registered in extracted so the document
// declares them before referencing them.
//
// The fallback scalar is passed through unchanged. It has rendered verbatim for
// as long as the field has existed and producers set it to whatever their
// package manager reported; re-encoding it here would rewrite what every
// existing document says about a large share of OS packages, which is a
// migration to make deliberately rather than as a side effect of adding a list.
func spdxLicense(licenses []*License, fallback string, extracted extractedLicenseSet) string {
	parts := make([]string, 0, len(licenses))
	for _, l := range licenses {
		if v := strings.TrimSpace(l.GetExpression()); v != "" {
			parts = append(parts, v)
			continue
		}
		if v := strings.TrimSpace(l.GetSpdxId()); v != "" {
			parts = append(parts, v)
			continue
		}
		// A free-form name is not a legal operand: "see LICENSE" has a space in
		// it, so emitting it raw makes the field unparseable and joining it with
		// AND produces "MIT AND see LICENSE", which reads as an expression and
		// is not one. SPDX's encoding for a license that is not on its list is a
		// LicenseRef-* identifier defined in hasExtractedLicensingInfos, so that
		// is what the text becomes — the name survives, in the field that can
		// hold it.
		if v := strings.TrimSpace(l.GetName()); v != "" {
			ref := spdxLicenseRef(v)
			if ref == "" {
				// Nothing in the name survives sanitization, so there is no
				// identifier to reference and no way to say this in SPDX.
				continue
			}
			extracted.addNamed(ref, v)
			parts = append(parts, ref)
		}
	}
	if len(parts) == 0 {
		return strings.TrimSpace(fallback)
	}
	if len(parts) == 1 {
		return parts[0]
	}
	for i, p := range parts {
		if isSPDXExpression(p) {
			// Parenthesize so joining cannot reassociate an operand: "A OR B"
			// AND "C" is not "A OR (B AND C)". Decided on the rendered operands
			// rather than the input count, so an entry that carried no value
			// cannot add parentheses around the one that did.
			parts[i] = "(" + p + ")"
		}
	}
	return strings.Join(parts, " AND ")
}

// spdxLicenseRef builds the LicenseRef-* identifier that stands for a free-form
// license name.
//
// SPDX restricts the part after the prefix to letters, digits, "." and "-", so
// every other run of characters collapses to a single dash. Returns "" when the
// name has nothing an identifier can be built from, which is the caller's signal
// that there is no way to reference it.
func spdxLicenseRef(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-':
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			dash = false
			b.WriteRune(r)
		default:
			dash = true
		}
	}
	id := strings.Trim(b.String(), "-.")
	if id == "" {
		return ""
	}
	return licenseRefPrefix + id
}

// spdxLicenseComments records what a conclusion was based on: the file it was
// read from and how sure whoever read it was.
//
// SPDX has no field for either, and PackageLicenseComments is the one place the
// spec puts prose about how the license fields were arrived at. It is free text,
// so this is for a human reading the document rather than a consumer parsing it
// -- but the alternative is dropping the difference between a certainty and an
// inference entirely, which is what the renderer did before.
//
// Empty when nothing was concluded, or when a conclusion carries neither
// detail: a comment saying nothing is worse than none, since a reader takes its
// presence as a sign there was something to say.
func spdxLicenseComments(licenses []*License) string {
	notes := make([]string, 0, len(licenses))
	for _, l := range licenses {
		value := strings.TrimSpace(spdxLicense([]*License{l}, "", extractedLicenseSet{}))
		if value == "" {
			continue
		}

		var detail []string
		if loc := strings.TrimSpace(l.GetLocation()); loc != "" {
			detail = append(detail, "read from "+loc)
		}
		// A conclusion at full confidence is not worth a note: it says the same
		// thing as a conclusion with no score attached.
		if c := l.GetConfidence(); c > 0 && c < 1 {
			detail = append(detail, "confidence "+strconv.FormatFloat(c, 'g', -1, 64))
		}
		if len(detail) == 0 {
			continue
		}
		notes = append(notes, value+": "+strings.Join(detail, ", "))
	}
	if len(notes) == 0 {
		return ""
	}
	return "Concluded " + strings.Join(notes, "; ") + "."
}

// spdxSupplier renders a supplier as the spec's Originator/Supplier shape, or
// NOASSERTION when none is known.
//
// SPDX requires the value to be typed — "Organization: name" or "Person: name".
// An unqualified name is not valid, and this package has no way to tell which a
// supplier is, so it reports the type SPDX itself uses for a supplier of
// software: an organization.
func spdxSupplier(supplier string) *common.Supplier {
	supplier = strings.TrimSpace(supplier)
	if supplier == "" {
		return &common.Supplier{Supplier: spdxNoAssertionValue}
	}
	return &common.Supplier{Supplier: supplier, SupplierType: "Organization"}
}
