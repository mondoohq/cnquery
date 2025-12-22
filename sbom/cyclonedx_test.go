// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/sbom"
	"go.mondoo.com/cnquery/v12/sbom/generator"
)

func TestCycloneDxOutput(t *testing.T) {
	report, err := generator.LoadReport("./testdata/alpine.json")
	require.NoError(t, err)

	sboms := generator.GenerateBom(report)

	// store bom in different formats
	selectedBom := sboms[0]

	exporter := sbom.New(sbom.FormatCycloneDxJSON)
	exporter.ApplyOptions(sbom.WithCPE(), sbom.WithEvidence())

	output := bytes.Buffer{}
	err = exporter.Render(&output, selectedBom)
	require.NoError(t, err)
	data := output.String()

	// os.WriteFile("./testdata/bom_cyclone.json", output.Bytes(), 0700)
	assert.Contains(t, data, "cyclonedx")

	// ensure os package is included
	assert.Contains(t, data, "alpine-baselayout")
	assert.Contains(t, data, "cpe:2.3:a:alpine-baselayout:alpine-baselayout:1695795276:aarch64:*:*:*:*:*:*")
	// check that package files are included
	assert.Contains(t, data, "etc/profile.d/color_prompt.sh.disabled")

	// ensure python package is included
	assert.Contains(t, data, "pip")
	assert.Contains(t, data, "cpe:2.3:a:pip_project:pip:21.2.4:*:*:*:*:*:*:*")
	assert.Contains(t, data, "pkg:pypi/pip@21.2.4")

	// ensure npm package is included
	assert.Contains(t, data, "npm")
	assert.Contains(t, data, "cpe:2.3:a:npm:npm:10.2.4:*:*:*:*:*:*:*")
	assert.Contains(t, data, "pkg:npm/npm@10.2.4")
}

func TestCycloneDxJsonDecoding(t *testing.T) {
	t.Run("alpine 3.19", func(t *testing.T) {
		f, err := os.Open("./testdata/alpine-319.cyclone.json")
		require.NoError(t, err)

		formatHandler := &sbom.CycloneDX{
			Format: cyclonedx.BOMFileFormatJSON,
		}

		bom, err := formatHandler.Parse(f)
		require.NoError(t, err)
		assert.NotNil(t, bom)
		assert.Equal(t, "alpine:3.19", bom.Asset.Name)
		assert.Equal(t, "alpine", bom.Asset.Platform.Name)
		assert.Equal(t, "3.19.1", bom.Asset.Platform.Version)
		// FIXME: support the bomRef property
		// assert.Equal(t, "//platformid.api.mondoo.app/runtime/docker/images/alpine:3.19", bom.Asset.PlatformIds[0])
	})

	t.Run("ubuntu 20.04 container", func(t *testing.T) {
		f, err := os.Open("./testdata/ubuntu-20.04-cyclonedx.json")
		require.NoError(t, err)

		formatHandler := &sbom.CycloneDX{
			Format: cyclonedx.BOMFileFormatJSON,
		}

		bom, err := formatHandler.Parse(f)
		require.NoError(t, err)
		assert.NotNil(t, bom)

		// verify we have the right asset and platform information.
		assert.Equal(t, "ubuntu", bom.Asset.Platform.Name)
		assert.Equal(t, "20.04", bom.Asset.Platform.Version)
		assert.Equal(t, []string{"linux", "unix", "os"}, bom.Asset.Platform.Family)
		assert.Equal(t, "Ubuntu 20.04.6 LTS", bom.Asset.Platform.Title)
		// this is the bom-ref
		assert.Equal(t, []string{"//platformid.api.mondoo.app/runtime/docker/images/e3cf4bf83104fade"}, bom.Asset.PlatformIds)
		// 1 library components + 1 os component
		assert.Len(t, bom.Packages, 2)

		// verify the generator is correct
		assert.Equal(t, "syft", bom.Generator.Name)
		assert.Equal(t, "1.38.2", bom.Generator.Version)
		assert.Equal(t, "anchore", bom.Generator.Vendor)
	})

	t.Run("ubuntu 22.04 container", func(t *testing.T) {
		f, err := os.Open("./testdata/ubuntu-22.04-cyclonedx.json")
		require.NoError(t, err)

		formatHandler := &sbom.CycloneDX{
			Format: cyclonedx.BOMFileFormatJSON,
		}

		bom, err := formatHandler.Parse(f)
		require.NoError(t, err)
		assert.NotNil(t, bom)

		// verify we have the right asset and platform information.
		assert.Equal(t, "ubuntu", bom.Asset.Platform.Name)
		assert.Equal(t, "22.04", bom.Asset.Platform.Version)
		assert.Equal(t, []string{"linux", "unix", "os"}, bom.Asset.Platform.Family)
		assert.Equal(t, "Ubuntu 22.04.5 LTS", bom.Asset.Platform.Title)
		// this is the bom-ref
		assert.Equal(t, []string{"//platformid.api.mondoo.app/runtime/docker/images/2e194621f3c81dfe"}, bom.Asset.PlatformIds)
		// 1 library components + 1 os component
		assert.Len(t, bom.Packages, 2)

		// verify the generator is correct
		assert.Equal(t, "syft", bom.Generator.Name)
		assert.Equal(t, "1.38.2", bom.Generator.Version)
		assert.Equal(t, "anchore", bom.Generator.Vendor)
	})
}

func TestCycloneDxXmlDecoding(t *testing.T) {
	f, err := os.Open("./testdata/alpine-319.cyclone.xml")
	require.NoError(t, err)

	formatHandler := &sbom.CycloneDX{
		Format: cyclonedx.BOMFileFormatXML,
	}

	bom, err := formatHandler.Parse(f)
	require.NoError(t, err)
	assert.NotNil(t, bom)
}

// syft dir:./next.js --source-name next.js_v15.4.1 -o cyclonedx-json > nextjs_v15_4_1.cyclonedx.json
func TestCycloneDxJsonDecoding_repo(t *testing.T) {
	f, err := os.Open("./testdata/nextjs_v15_4_1.cyclonedx.json")
	require.NoError(t, err)

	formatHandler := &sbom.CycloneDX{
		Format: cyclonedx.BOMFileFormatJSON,
	}

	bom, err := formatHandler.Parse(f)
	require.NoError(t, err)
	assert.NotNil(t, bom)
	assert.Equal(t, "next.js_v15.4.1", bom.Asset.Name)
	assert.Equal(t, "cyclonedx", bom.Asset.Platform.Name)
	assert.Equal(t, "1.6", bom.Asset.Platform.Version)
	assert.Equal(t, "CycloneDX", bom.Asset.Platform.Title)
}

func TestCycloneDxJsonDecoding_Alpine_syft(t *testing.T) {
	f, err := os.Open("./testdata/alpine-3.19.cyclonedx.syft.json")
	require.NoError(t, err)

	formatHandler := &sbom.CycloneDX{
		Format: cyclonedx.BOMFileFormatJSON,
	}

	bom, err := formatHandler.Parse(f)
	require.NoError(t, err)
	assert.NotNil(t, bom)
	assert.Equal(t, "alpine", bom.Asset.Name)
	assert.Equal(t, "alpine", bom.Asset.Platform.Name)
	assert.Equal(t, "3.19.9", bom.Asset.Platform.Version)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/docker/images/cd03a8ea6f29f815", bom.Asset.PlatformIds[0])
}
