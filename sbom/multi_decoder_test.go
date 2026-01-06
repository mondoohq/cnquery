// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiDecoder(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		f, err := os.Open("./testdata/alpine-319.cyclone.json")
		require.NoError(t, err)
		cnquery := New(FormatJson)
		cycloneDx := NewCycloneDX(FormatCycloneDxJSON)
		spdxJson := NewSPDX(FormatSpdxJSON)
		spdxTag := NewSPDX(FormatSpdxTagValue)
		decoders := []Decoder{
			cnquery,
			spdxJson,
			spdxTag,
			cycloneDx,
		}
		mh := NewMultiDecoder(decoders...)
		bom2, err := mh.Parse(f)
		require.NoError(t, err)
		assert.NotNil(t, bom2)
	})

	t.Run("missing the correct decoder", func(t *testing.T) {
		f, err := os.Open("./testdata/alpine-319.cyclone.json")
		require.NoError(t, err)
		cnquery := New(FormatJson)
		spdxJson := NewSPDX(FormatSpdxJSON)
		spdxTag := NewSPDX(FormatSpdxTagValue)
		decoders := []Decoder{
			cnquery,
			spdxJson,
			spdxTag,
		}
		mh := NewMultiDecoder(decoders...)
		bom2, err := mh.Parse(f)
		require.Error(t, err)
		assert.Nil(t, bom2)
	})
}
