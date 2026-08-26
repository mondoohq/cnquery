// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers-sdk/v1/resources"
	"go.mondoo.com/mql/types"
)

// updateTestProvider builds a minimal Provider for selectProvidersToUpdate
// tests. A non-empty path marks an installed (external) provider; an empty path
// marks a builtin one compiled into the binary.
func updateTestProvider(name, path string) *providers.Provider {
	return &providers.Provider{Provider: &plugin.Provider{Name: name}, Path: path}
}

func updateProviderNames(ps []*providers.Provider) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Name
	}
	return out
}

func TestSelectProvidersToUpdate_NoNamesReturnsAllInstalledSorted(t *testing.T) {
	all := []*providers.Provider{
		updateTestProvider("os", "/p/os"),
		updateTestProvider("aws", "/p/aws"),
		updateTestProvider("core", ""), // builtin, must be excluded
	}

	toUpdate, notInstalled := selectProvidersToUpdate(all, nil)

	assert.Equal(t, []string{"aws", "os"}, updateProviderNames(toUpdate), "all installed providers, sorted, builtin excluded")
	assert.Empty(t, notInstalled)
}

func TestSelectProvidersToUpdate_NamedSubset(t *testing.T) {
	all := []*providers.Provider{
		updateTestProvider("aws", "/p/aws"),
		updateTestProvider("gcp", "/p/gcp"),
		updateTestProvider("os", "/p/os"),
	}

	toUpdate, notInstalled := selectProvidersToUpdate(all, []string{"aws", "os"})

	assert.Equal(t, []string{"aws", "os"}, updateProviderNames(toUpdate))
	assert.Empty(t, notInstalled)
}

func TestSelectProvidersToUpdate_MissingNameReportedNotFatal(t *testing.T) {
	all := []*providers.Provider{updateTestProvider("aws", "/p/aws")}

	toUpdate, notInstalled := selectProvidersToUpdate(all, []string{"aws", "notreal"})

	assert.Equal(t, []string{"aws"}, updateProviderNames(toUpdate))
	assert.Equal(t, []string{"notreal"}, notInstalled, "unknown names are reported, not selected")
}

func TestSelectProvidersToUpdate_BuiltinNameIsNotUpdatable(t *testing.T) {
	all := []*providers.Provider{
		updateTestProvider("core", ""), // builtin
		updateTestProvider("aws", "/p/aws"),
	}

	// A builtin provider is compiled into the binary, so naming it is treated
	// the same as an uninstalled provider: reported and skipped, never selected.
	toUpdate, notInstalled := selectProvidersToUpdate(all, []string{"core"})

	assert.Empty(t, toUpdate)
	assert.Equal(t, []string{"core"}, notInstalled)
}

// --- schema/JSON output contract ---
//
// These tests lock the JSON shape emitted by the schema-browsing commands
// (providers list/info/resources). They exercise the pure builder functions
// with fabricated inputs so the contract is verified without a provider
// registry or installed providers on disk.

func TestShortProviderName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"go.mondoo.com/mql/providers/aws", "aws"},
		{"go.mondoo.com/mql/providers/core", "core"},
		{"go.mondoo.com/mql/providers/aws/resources", "aws"},
		{"aws", "aws"},
		{"some/other/path", "path"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, shortProviderName(c.in), "input %q", c.in)
	}
}

func TestBuildProviderListEntries(t *testing.T) {
	all := []*providers.Provider{
		{Provider: &plugin.Provider{
			Name:    "os",
			Version: "13.2.0",
			Connectors: []plugin.Connector{
				{Name: "local"},
				{Name: "internal", IsHidden: true}, // must be omitted
			},
		}, Path: "/p/os"},
		{Provider: &plugin.Provider{Name: "core", Version: "13.2.0"}}, // builtin (no path)
	}

	entries := buildProviderListEntries(all)

	// sorted by name
	require.Len(t, entries, 2)
	assert.Equal(t, "core", entries[0].Name)
	assert.Equal(t, "os", entries[1].Name)

	// status derives from on-disk path
	assert.Equal(t, "builtin", entries[0].Status)
	assert.Equal(t, "installed", entries[1].Status)

	// hidden connectors are dropped
	assert.Equal(t, []string{"local"}, entries[1].Connectors)

	// JSON contract: status is always present (not omitempty)
	blob, err := json.Marshal(entries[0])
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(blob, &m))
	assert.Contains(t, m, "status")
	assert.Equal(t, "builtin", m["status"])
}

func TestBuildProviderInfoEntry(t *testing.T) {
	p := &providers.Provider{Provider: &plugin.Provider{
		Name:    "aws",
		ID:      "go.mondoo.com/mql/providers/aws",
		Version: "13.2.0",
		Connectors: []plugin.Connector{
			{
				Name:      "aws",
				Short:     "Amazon Web Services",
				Aliases:   []string{"amazon"},
				Discovery: []string{"accounts"},
				Flags: []plugin.Flag{
					{Long: "profile", Type: plugin.FlagType_String, Desc: "AWS profile"},
					{Long: "secret", Type: plugin.FlagType_String, Option: plugin.FlagOption_Hidden}, // omitted
				},
			},
			{Name: "hidden-conn", IsHidden: true}, // omitted
		},
	}, Path: "/p/aws"}

	entry := buildProviderInfoEntry(p)

	assert.Equal(t, "aws", entry.Name)
	assert.Equal(t, "go.mondoo.com/mql/providers/aws", entry.ID)
	assert.Equal(t, "/p/aws", entry.Path)

	require.Len(t, entry.Connectors, 1, "hidden connector omitted")
	conn := entry.Connectors[0]
	assert.Equal(t, "aws", conn.Name)
	assert.Equal(t, []string{"amazon"}, conn.Aliases)
	assert.Equal(t, []string{"accounts"}, conn.Discovery)

	require.Len(t, conn.Flags, 1, "hidden flag omitted")
	assert.Equal(t, "profile", conn.Flags[0].Long)
	assert.Equal(t, "string", conn.Flags[0].Type)
}

func TestBuildResourceList(t *testing.T) {
	schema := &resources.Schema{Resources: map[string]*resources.ResourceInfo{
		"aws.s3.bucket": {
			Name:     "aws.s3.bucket",
			Title:    "Amazon S3 Bucket",
			Provider: "go.mondoo.com/mql/providers/aws",
			Fields: map[string]*resources.Field{
				"arn":  {Name: "arn", Type: string(types.String)},
				"name": {Name: "name", Type: string(types.String)},
			},
		},
		"aws.deprecated.thing": {
			Name:     "aws.deprecated.thing",
			Provider: "go.mondoo.com/mql/providers/aws",
			Maturity: resources.MaturityDeprecated,
		},
		"aws.internal.secret": {
			Name:    "aws.internal.secret",
			Private: true, // must be omitted
		},
	}}

	list := buildResourceList(schema, "aws")

	assert.Equal(t, "aws", list.Provider)
	require.Equal(t, 2, list.TotalResources, "private resource omitted")
	require.Len(t, list.Resources, 2)

	// sorted by name
	assert.Equal(t, "aws.deprecated.thing", list.Resources[0].Name)
	assert.Equal(t, "aws.s3.bucket", list.Resources[1].Name)

	// origin shortened, maturity surfaced, field count
	assert.Equal(t, "aws", list.Resources[1].Provider)
	assert.Equal(t, "deprecated", list.Resources[0].Maturity)
	assert.Equal(t, 2, list.Resources[1].FieldCount)
}

func TestBuildResourceDetail(t *testing.T) {
	ri := &resources.ResourceInfo{
		Name:     "aws.s3.bucket",
		Title:    "Amazon S3 Bucket",
		Provider: "go.mondoo.com/mql/providers/aws",
		Fields: map[string]*resources.Field{
			"name":   {Name: "name", Type: string(types.String), IsMandatory: true},
			"arn":    {Name: "arn", Type: string(types.String), Maturity: resources.MaturityDeprecated},
			"hidden": {Name: "hidden", Type: string(types.String), IsPrivate: true}, // omitted
		},
	}

	detail := buildResourceDetail(ri)

	assert.Equal(t, "aws.s3.bucket", detail.Name)
	assert.Equal(t, "aws", detail.Provider)
	require.Equal(t, 2, detail.FieldCount, "private field omitted")
	require.Len(t, detail.Fields, 2)

	// sorted by name: arn, name
	assert.Equal(t, "arn", detail.Fields[0].Name)
	assert.Equal(t, "name", detail.Fields[1].Name)

	// type label, maturity, mandatory
	assert.Equal(t, "string", detail.Fields[0].Type)
	assert.Equal(t, "deprecated", detail.Fields[0].Maturity)
	assert.True(t, detail.Fields[1].IsMandatory)

	// JSON contract: is_mandatory omitted when false, present when true
	blob, err := json.Marshal(detail.Fields[1])
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(blob, &m))
	assert.Contains(t, m, "is_mandatory")
	blob, err = json.Marshal(detail.Fields[0])
	require.NoError(t, err)
	m = map[string]any{}
	require.NoError(t, json.Unmarshal(blob, &m))
	assert.NotContains(t, m, "is_mandatory")
}
