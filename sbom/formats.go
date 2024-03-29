// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"errors"
	"github.com/CycloneDX/cyclonedx-go"
	"io"
	"strings"
)

const (
	FormatJson          string = "json"
	FormatCycloneDxJSON string = "cyclonedx-json"
	FormatCycloneDxXML  string = "cyclonedx-xml"
	FormatSpdxJSON      string = "spdx-json"
	FormatSpdxTagValue  string = "spdx-tag-value"
	FormatList          string = "table"
)

var conversionNotSupportedError = errors.New("conversion not supported")

type FormatSpecificationHandler interface {
	// Convert converts cnquery sbom to the desired format
	Convert(bom *Sbom) (interface{}, error)
	// Render writes the converted sbom to the writer in the desired format
	Render(w io.Writer, bom *Sbom) error
}

func AllFormats() string {
	formats := []string{
		FormatJson, FormatCycloneDxJSON, FormatCycloneDxXML, FormatSpdxJSON, FormatSpdxTagValue, FormatList,
	}

	return strings.Join(formats, ", ")
}

func New(fomat string) FormatSpecificationHandler {
	switch fomat {
	case FormatJson:
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
