// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"os"
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
		assert.Equal(t, "concluded from LICENSE", concluded.GetLocation())
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
