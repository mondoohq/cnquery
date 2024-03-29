// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/package-url/packageurl-go"
	"github.com/spdx/tools-golang/convert"
	"github.com/spdx/tools-golang/spdx"
	"github.com/spdx/tools-golang/spdx/v2/v2_1"
	"github.com/spdx/tools-golang/spdx/v2/v2_2"
	"github.com/spdx/tools-golang/spdx/v2/v2_3"
	"github.com/spdx/tools-golang/tagvalue"
)

type Spdx struct {
	Version string
	Format  string
}

func (s *Spdx) convertToSpdx(bom *Sbom) *spdx.Document {
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

	for i := range bom.Packages {
		pkg := bom.Packages[i]

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
				Category: spdx.CategorySecurity,
				Locator:  pkg.Purl,
			})
		}

		doc.Packages = append(doc.Packages, &spdx.Package{
			PackageSPDXIdentifier:     NewSPDXPackageID(pkg),
			PackageName:               pkg.Name,
			PackageVersion:            pkg.Version,
			PackageLicenseDeclared:    pkg.Version,
			PackageDescription:        pkg.Description,
			PackageExternalReferences: refs,
			PackageFileName:           pkg.Location,
		})
	}

	return doc
}

var expr = regexp.MustCompile("[^a-zA-Z0-9.-]")

// NewSPDXPackageID creates a new SPDX ID for a package
// see https://spdx.github.io/spdx-spec/v2.3/relationships-between-SPDX-elements/
func NewSPDXPackageID(pkg *Package) spdx.ElementID {
	hash, _ := pkg.Hash()

	id := fmt.Sprintf("Package-%s-%s-%s", pkg.Type, pkg.Name, hash)
	expr.ReplaceAllString(id, "-")
	return spdx.ElementID(id)
}

func (s *Spdx) Convert(bom *Sbom) (interface{}, error) {
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

func (s *Spdx) Parse(r io.Reader) (*Sbom, error) {

	type reader func(r io.Reader) (*spdx.Document, error)

	// try to parse all supported SPDX format
	var readers = []reader{
		func(r io.Reader) (*spdx.Document, error) {
			return tagvalue.Read(r)
		},
		func(r io.Reader) (*spdx.Document, error) {
			var doc spdx.Document
			err := json.NewDecoder(r).Decode(&s)
			return &doc, err
		},
	}

	for _, reader := range readers {
		doc, err := reader(r)
		if err == nil {
			return s.convertToSbom(doc), nil
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
		},
		Packages: []*Package{},
	}

	for i := range doc.Packages {
		pkg := doc.Packages[i]

		if pkg.PrimaryPackagePurpose == "CONTAINER" {
			bom.Asset.Platform = &Platform{
				Name:    pkg.PackageName,
				Version: pkg.PackageVersion,
				Title:   fmt.Sprintf("%s %s", pkg.PackageName, pkg.PackageVersion),
			}
			bom.Asset.Platform.Family = familyMap[pkg.PackageName]
			continue
		}

		bomPkg := &Package{
			Name:        pkg.PackageName,
			Version:     pkg.PackageVersion,
			Description: pkg.PackageDescription,
			Location:    pkg.PackageFileName,
			Type:        "", // extract package type from purl
			Purl:        "",
			Cpes:        []string{}, // TODO: extract CPEs from external references
		}

		for _, ref := range pkg.PackageExternalReferences {
			if ref.RefType == spdx.PackageManagerPURL {
				bomPkg.Purl = ref.Locator
				pkgUrl, err := packageurl.FromString(ref.Locator)
				if err == nil {
					bomPkg.Type = pkgUrl.Type
				}
			}
			if ref.RefType == spdx.SecurityCPE23Type {
				bomPkg.Cpes = append(bomPkg.Cpes, ref.Locator)
			}
		}

		if pkg.PackageFileName != "" {
			bomPkg.EvidenceList = append(bomPkg.EvidenceList, &Evidence{
				Type:  EvidenceType_EVIDENCE_TYPE_FILE,
				Value: pkg.PackageFileName,
			})
		}

		bom.Packages = append(bom.Packages, bomPkg)
	}

	return bom
}
