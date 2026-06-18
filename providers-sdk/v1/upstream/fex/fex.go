// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package fex

//go:generate protoc --plugin=protoc-gen-go=../../../../scripts/protoc/protoc-gen-go --plugin=protoc-gen-go-vtproto=../../../../scripts/protoc/protoc-gen-go-vtproto --go_out=. --go_opt=paths=source_relative --go-vtproto_out=. --go-vtproto_opt=paths=source_relative --go-vtproto_opt=features=marshal+unmarshal+size fex.proto

// FexToDocument wraps a FindingExchange in a FindingDocument.
func FexToDocument(f *FindingExchange) *FindingDocument {
	return &FindingDocument{
		Finding: &FindingDocument_Fex{Fex: f},
	}
}

// FexToDocuments wraps each FindingExchange in a FindingDocument.
func FexToDocuments(fexes []*FindingExchange) []*FindingDocument {
	docs := make([]*FindingDocument, 0, len(fexes))
	for _, f := range fexes {
		docs = append(docs, FexToDocument(f))
	}
	return docs
}

// VexToDocument wraps a VulnerabilityExchange in a FindingDocument.
func VexToDocument(v *VulnerabilityExchange) *FindingDocument {
	return &FindingDocument{
		Finding: &FindingDocument_Vex{Vex: v},
	}
}

// VexToDocuments wraps each VulnerabilityExchange in a FindingDocument.
func VexToDocuments(vexes []*VulnerabilityExchange) []*FindingDocument {
	docs := make([]*FindingDocument, 0, len(vexes))
	for _, v := range vexes {
		docs = append(docs, VexToDocument(v))
	}
	return docs
}

// NewRemediation builds a Remediation with the Fix category and the given
// summary, details, and optional URL.
func NewRemediation(summary, details, url string) *Remediation {
	return &Remediation{
		Category: Remediation_Fix,
		Summary:  summary,
		Details:  details,
		Url:      url,
	}
}
