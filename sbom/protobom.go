// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"io"
	"strings"

	"github.com/package-url/packageurl-go"
	"github.com/protobom/protobom/pkg/reader"
	protobom_sbom "github.com/protobom/protobom/pkg/sbom"
)

func NewProtobom() *Protobom {
	return &Protobom{}
}

type Protobom struct {
	opts renderOpts
}

func (s *Protobom) Parse(f io.ReadSeeker) (*Sbom, error) {
	reader := reader.New()

	document, err := reader.ParseStream(f)
	if err != nil {
		return nil, err
	}

	return s.convertToSbom(document), nil
}

// readNodeLicensing carries a node's licensing across into the package.
//
// This is an importer: the document being read was produced by somebody else,
// and dropping what it stated about a package's licensing loses information
// that cannot be recovered from anywhere else in the file. Every field below
// was already parsed and then discarded.
//
// protobom keeps the same declared/concluded distinction the model does.
// `licenses` is what the package says about itself and `license_concluded` is
// what the document's producer determined, so they map onto the two
// acquisitions rather than being merged into one list.
func readNodeLicensing(pkg *Package, node *protobom_sbom.Node) {
	for _, l := range node.GetLicenses() {
		if entry := DeclaredLicense(l); entry != nil {
			pkg.Licenses = append(pkg.Licenses, entry)
		}
	}

	// The location is the document's own license comment when it has one: a
	// concluded license read out of a file is the case that field exists to
	// explain, and it is the only place protobom records where the conclusion
	// came from. The confidence is left to default, because the format carries
	// no score for a consumer to have measured.
	if entry := ConcludedLicense(node.GetLicenseConcluded(), node.GetLicenseComments(), 0); entry != nil {
		pkg.Licenses = append(pkg.Licenses, entry)
	}

	// The legacy scalar keeps being written, as every other producer in this
	// package does, so a consumer that has not migrated to the list still sees
	// a license. It takes the first DECLARED entry, which is the convention the
	// field's own documentation asks producers to follow, falling back to the
	// concluded value when the document declared none: a scalar that stayed
	// empty while a license was in fact stated is the failure the fallback
	// exists to prevent.
	for _, l := range pkg.Licenses {
		if l.GetAcquisition() == LicenseAcquisition_LICENSE_ACQUISITION_DECLARED {
			pkg.License = licenseValueOf(l)
			break
		}
	}
	if pkg.License == "" {
		pkg.License = strings.TrimSpace(node.GetLicenseConcluded())
	}

	if c := strings.TrimSpace(node.GetCopyright()); c != "" {
		pkg.Copyright = []string{c}
	}

	for _, person := range node.GetSuppliers() {
		if name := strings.TrimSpace(person.GetName()); name != "" {
			pkg.Supplier = name
			break
		}
	}
}

// licenseValueOf returns whichever of the three mutually exclusive value fields
// an entry carries, which is what the scalar holds.
func licenseValueOf(l *License) string {
	switch {
	case l.GetExpression() != "":
		return l.GetExpression()
	case l.GetSpdxId() != "":
		return l.GetSpdxId()
	default:
		return l.GetName()
	}
}

func (s *Protobom) convertToSbom(doc *protobom_sbom.Document) *Sbom {
	bom := &Sbom{
		Asset: &Asset{
			Name: doc.Metadata.Name,
		},
		Packages: make([]*Package, 0),
	}

	if doc.Metadata != nil && len(doc.Metadata.Tools) > 0 {
		bom.Generator = &Generator{
			Name:    doc.Metadata.Tools[0].Name,
			Version: doc.Metadata.Tools[0].Version,
			Vendor:  doc.Metadata.Tools[0].Vendor,
		}
	}

	if doc.GetNodeList() == nil || len(doc.GetNodeList().GetNodes()) == 0 {
		return bom // no nodes, return empty SBOM
	}

	for _, node := range doc.GetNodeList().GetNodes() {
		pkg := &Package{
			Name:    node.Name,
			Version: node.Version,
		}
		readNodeLicensing(pkg, node)

		for key, identifier := range node.GetIdentifiers() {
			if key == int32(protobom_sbom.SoftwareIdentifierType_PURL) {
				pkg.Purl = identifier
				if purl, err := packageurl.FromString(identifier); err == nil {
					pkg.Type = purl.Type
				}
			} else if key == int32(protobom_sbom.SoftwareIdentifierType_CPE23) {
				pkg.Cpes = append(pkg.Cpes, identifier)
			} else if key == int32(protobom_sbom.SoftwareIdentifierType_CPE22) {
				pkg.Cpes = append(pkg.Cpes, identifier)
			}
		}

		if !s.opts.IncludeCPE {
			// if CPEs are not included, clear them
			pkg.Cpes = nil
		}

		purposes := node.GetPrimaryPurpose()
		if len(purposes) > 0 {
			switch purposes[0] {
			case protobom_sbom.Purpose_OPERATING_SYSTEM:
				bom.Asset.Platform = &Platform{
					Name:    pkg.Name,
					Version: pkg.Version,
					Title:   pkg.Description,
				}
				bom.Asset.Platform.Family = familyMap[pkg.Name]
				bom.Asset.Platform.Cpes = pkg.Cpes
			case protobom_sbom.Purpose_APPLICATION:
				bom.Packages = append(bom.Packages, pkg)
			}
		}
	}

	return bom
}
