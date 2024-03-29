// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"encoding/json"
	"fmt"
	"github.com/spdx/tools-golang/convert"
	"github.com/spdx/tools-golang/spdx"
	"github.com/spdx/tools-golang/spdx/v2/v2_1"
	"github.com/spdx/tools-golang/spdx/v2/v2_2"
	"github.com/spdx/tools-golang/spdx/v2/v2_3"
	"github.com/spdx/tools-golang/tagvalue"
	"io"
	"regexp"
	"time"
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
					Creator:     bom.Generator.Name,
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
