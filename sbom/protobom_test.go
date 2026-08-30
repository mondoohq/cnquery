// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtobomSpdxJsonDecoder(t *testing.T) {
	f, err := os.Open("testdata/alpine-319.spdx.json")
	require.NoError(t, err)

	decoder := NewProtobom()

	sbomReport, err := decoder.Parse(f)
	require.NoError(t, err)
	assert.NotNil(t, sbomReport)

}

func TestProtobomCycloneDxJsonDecoder(t *testing.T) {
	f, err := os.Open("testdata/alpine-319.cyclone.json")
	require.NoError(t, err)

	decoder := NewProtobom()

	sbomReport, err := decoder.Parse(f)
	require.NoError(t, err)
	assert.NotNil(t, sbomReport)
}

// The Protobom decoder is an importer: the document it reads was produced by
// somebody else, and what it says about a package's licensing cannot be
// recovered from anywhere else in the file. Every field asserted here was
// already being parsed and then discarded.
func TestProtobomReadsLicensing(t *testing.T) {
	f, err := os.Open("testdata/licensing.spdx.json")
	require.NoError(t, err)
	defer f.Close()

	bom, err := NewProtobom().Parse(f)
	require.NoError(t, err)

	byName := map[string]*Package{}
	for _, p := range bom.Packages {
		byName[p.Name] = p
	}

	t.Run("declared and concluded stay distinct", func(t *testing.T) {
		p := byName["declared-and-concluded"]
		require.NotNil(t, p)
		require.Len(t, p.Licenses, 2)

		declared := p.Licenses[0]
		assert.Equal(t, LicenseAcquisition_LICENSE_ACQUISITION_DECLARED, declared.GetAcquisition())
		assert.Equal(t, "MIT", declared.GetSpdxId())

		// The whole point of the split: what the package says about itself and
		// what the document's producer determined disagree, and flattening them
		// would report a grant the shipped code does not make.
		concluded := p.Licenses[1]
		assert.Equal(t, LicenseAcquisition_LICENSE_ACQUISITION_CONCLUDED, concluded.GetAcquisition())
		assert.Equal(t, "AGPL-3.0-only", concluded.GetSpdxId())

		// No location. The model documents it as the file a license was read
		// from, and protobom has no field holding one: license_comments is
		// free-form prose ("concluded from LICENSE" is a sentence, not a path),
		// so putting it here would make the field mean two different things
		// depending on who wrote the document.
		assert.Empty(t, concluded.GetLocation())

		// And no confidence. The document attached no score, and 0 is how the
		// model spells that; 1.0 would report a conclusion nobody measured as
		// one that matched exactly. This is the assertion that fails if the
		// constructor ever goes back to promoting an absent score, which is a
		// change no other test here would notice.
		assert.Zero(t, concluded.GetConfidence())

		// The declared entry is the contrast: a package stating its own license
		// is not a measurement, so it is certain by construction.
		assert.Equal(t, 1.0, declared.GetConfidence())
	})

	t.Run("the legacy scalar takes the first declared entry", func(t *testing.T) {
		assert.Equal(t, "MIT", byName["declared-and-concluded"].License)
		assert.Equal(t, "Apache-2.0", byName["declared-only"].License)
	})

	t.Run("copyright and supplier are carried", func(t *testing.T) {
		p := byName["declared-and-concluded"]
		require.NotNil(t, p)
		assert.Equal(t, []string{"Copyright (c) 2019 Example Corp"}, p.Copyright)
		assert.Equal(t, "Example Corp", p.Supplier)
	})

	// NOASSERTION is SPDX for "this document does not say", and it is
	// identifier-shaped, so nothing about its spelling would stop it becoming a
	// license named NOASSERTION. protobom drops it before it reaches the model;
	// this pins that, because if a future version stopped, every package in
	// every imported document would gain a license that does not exist.
	t.Run("NOASSERTION is not a license", func(t *testing.T) {
		p := byName["states-nothing"]
		require.NotNil(t, p)
		assert.Empty(t, p.Licenses)
		assert.Empty(t, p.License)
	})
}

// Every renderer reads Asset.Platform unguarded, so an imported document that
// names no operating system -- which is most of them -- crashed whatever it was
// handed to, including this package's own renderers. Parsing a document and
// then rendering it is the obvious thing to do with an importer, and it panicked.
func TestProtobomOutputCanBeRendered(t *testing.T) {
	f, err := os.Open("testdata/licensing.spdx.json")
	require.NoError(t, err)
	defer f.Close()

	bom, err := NewProtobom().Parse(f)
	require.NoError(t, err)
	require.NotNil(t, bom.Asset.Platform, "a nil platform is what every renderer dereferences")

	var b strings.Builder
	require.NotPanics(t, func() {
		require.NoError(t, New(FormatCycloneDxJSON).Render(&b, bom))
	})
	assert.Contains(t, b.String(), "declared-and-concluded")

	// SPDX is deliberately not asserted here. It fails on the same input for an
	// unrelated reason this change does not address: the renderer writes a
	// Creator from Generator.Vendor, and common.Creator refuses to marshal an
	// empty one, so any BOM whose generator has no vendor -- an imported one
	// included -- errors out. That is a renderer bug rather than an importer
	// one, and fixing it means choosing what a document with no stated vendor
	// says it was created by.
}

// The document name was read one line before the guard that checks whether the
// document has any metadata, which made the guard unreachable.
func TestProtobomHandlesADocumentWithoutMetadata(t *testing.T) {
	s := &Protobom{}
	require.NotPanics(t, func() {
		bom := s.convertToSbom(nil)
		require.NotNil(t, bom)
		assert.Empty(t, bom.Packages)
	})
}
