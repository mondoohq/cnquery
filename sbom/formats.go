// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"errors"
	"io"
	"strings"

	"github.com/CycloneDX/cyclonedx-go"
)

const (
	FormatJson          string = "json"
	FormatCycloneDxJSON string = "cyclonedx-json"
	FormatCycloneDxXML  string = "cyclonedx-xml"
	FormatSpdxJSON      string = "spdx-json"
	FormatSpdxTagValue  string = "spdx-tag-value"
	FormatList          string = "table"
)

// unnamedSubject is what a BOM's subject is called when the asset does not name
// itself. Both document formats have to put a name somewhere -- SPDX's document
// name is mandatory, and a CycloneDX component's name is required by the schema
// -- so the choice is between this and a nameless entry a consumer has to
// recognise as junk. Shared so that the SPDX and CycloneDX renderings of one
// BOM do not disagree about what its subject is called.
const unnamedSubject = "sbom"

var (
	errConversionNotSupported = errors.New("conversion is not supported")
	errParsingNotSupported    = errors.New("parsing is not supported")
)

type FormatSpecificationHandler interface {
	// Convert converts cnquery sbom to the desired format
	Convert(bom *Sbom) (any, error)
	// Render writes the converted sbom to the writer in the desired format
	Render(w io.Writer, bom *Sbom) error
	// ApplyOptions applies render options to the handler
	ApplyOptions(opts ...renderOption)
	Decoder
}

func AllFormats() string {
	formats := []string{
		FormatJson, FormatCycloneDxJSON, FormatCycloneDxXML, FormatSpdxJSON, FormatSpdxTagValue, FormatList,
	}

	return strings.Join(formats, ", ")
}

func New(format string) FormatSpecificationHandler {
	switch format {
	case FormatJson, "cnquery-json":
		return &CnqueryBOM{}
	case FormatCycloneDxJSON:
		return &CycloneDX{
			Format: cyclonedx.BOMFileFormatJSON,
		}
	case FormatCycloneDxXML:
		return &CycloneDX{
			Format: cyclonedx.BOMFileFormatXML,
		}
	case FormatSpdxJSON:
		return &Spdx{
			Version: "2.3",
			Format:  FormatSpdxJSON,
		}
	case FormatSpdxTagValue:
		return &Spdx{
			Version: "2.3",
			Format:  FormatSpdxTagValue,
		}
	case FormatList:
		fallthrough
	default:
		return &TextList{}
	}
}
