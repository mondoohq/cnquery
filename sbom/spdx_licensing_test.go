// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The SPDX decoder is an importer: the document it reads was produced by
// somebody else, and what it says about licensing cannot be recovered from
// anywhere else in the file. It read each package's name, version, identifiers
// and file evidence and dropped the rest.
//
// #10597 fixed this for the Protobom decoder. This asserts through
// DefaultMultiDecoder, which is what consumers call and which routes SPDX to
// *this* decoder -- so the earlier fix reached none of them, and a test calling
// only the Protobom decoder would not have noticed.
func TestSpdxDecoderReadsLicensing(t *testing.T) {
	f, err := os.Open("testdata/licensing.spdx.json")
	require.NoError(t, err)
	defer f.Close()

	bom, err := DefaultMultiDecoder().Parse(f)
	require.NoError(t, err)

	byName := map[string]*Package{}
	for _, p := range bom.Packages {
		byName[p.Name] = p
	}

	t.Run("declared and concluded are told apart by the field they came from", func(t *testing.T) {
		p := byName["declared-and-concluded"]
		require.NotNil(t, p)
		require.Len(t, p.Licenses, 2)

		assert.Equal(t, LicenseAcquisition_LICENSE_ACQUISITION_DECLARED, p.Licenses[0].GetAcquisition())
		assert.Equal(t, "MIT", p.Licenses[0].GetSpdxId())

		assert.Equal(t, LicenseAcquisition_LICENSE_ACQUISITION_CONCLUDED, p.Licenses[1].GetAcquisition())
		assert.Equal(t, "AGPL-3.0-only", p.Licenses[1].GetSpdxId())

		// Zero, not 1.0: SPDX carries no score, and the model spells "nobody
		// measured this" as 0. Reporting full confidence would rank an imported
		// conclusion nobody scored alongside one that scored perfectly.
		assert.Zero(t, p.Licenses[1].GetConfidence())

		// The declared entry wins the legacy scalar, so a consumer that has not
		// migrated to the list still sees a license.
		assert.Equal(t, "MIT", p.License)

		assert.Equal(t, []string{"Copyright (c) 2019 Example Corp"}, p.Copyright)
		assert.Equal(t, "Example Corp", p.Supplier)
	})

	t.Run("a declaration alone is enough", func(t *testing.T) {
		p := byName["declared-only"]
		require.NotNil(t, p)
		require.Len(t, p.Licenses, 1)
		assert.Equal(t, LicenseAcquisition_LICENSE_ACQUISITION_DECLARED, p.Licenses[0].GetAcquisition())
		assert.Equal(t, "Apache-2.0", p.Licenses[0].GetSpdxId())
		assert.Equal(t, "Apache-2.0", p.License)
	})

	// NOASSERTION and NONE are SPDX's two ways of saying it is not telling you.
	// Carrying either through as a license name would report a package as
	// licensed under the string "NOASSERTION", which matches nothing and reads
	// as a statement where the document made none.
	for _, name := range []string{"states-nothing", "states-none"} {
		t.Run(name+" reports no license", func(t *testing.T) {
			p := byName[name]
			require.NotNil(t, p)
			assert.Empty(t, p.Licenses)
			assert.Empty(t, p.License)
		})
	}
}
