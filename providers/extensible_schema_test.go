// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/resources"
)

func TestExtensibleSchema(t *testing.T) {
	s := newExtensibleSchema()
	s.coordinator = newCoordinator()

	s.Add("first", &resources.Schema{
		Resources: map[string]*resources.ResourceInfo{
			"eternity": {
				Fields: map[string]*resources.Field{
					"iii": {Provider: "first"},
					"v":   {Provider: "first"},
				},
				Provider: "first",
			},
		},
	})

	s.Add("second", &resources.Schema{
		Resources: map[string]*resources.ResourceInfo{
			"eternity": {
				Fields: map[string]*resources.Field{
					"iii": {Provider: "second"},
				},
				Provider: "second",
			},
		},
	})

	info := s.Lookup("eternity")
	require.NotNil(t, info)
	require.Len(t, info.Others, 1)

	// Check that both providers are present for resource "eternity"
	providers := []string{info.Provider, info.Others[0].Provider}
	assert.ElementsMatch(t, []string{"first", "second"}, providers)

	info, finfo := s.LookupField("eternity", "iii")
	require.NotNil(t, info)
	require.Len(t, info.Others, 1)

	// Check that both providers are present for field "iii"
	providers = []string{finfo.Provider, info.Others[0].Fields["iii"].Provider}
	assert.ElementsMatch(t, []string{"first", "second"}, providers)

	info, finfo = s.LookupField("eternity", "v")
	require.NotNil(t, info)
	assert.Equal(t, "first", finfo.Provider)

	// Find field from resource
	filePath, fieldinfos, found := s.FindField(info, "v")
	require.True(t, found)
	require.Equal(t, resources.FieldPath{"v"}, filePath)
	require.Len(t, fieldinfos, 1)
	require.Equal(t, "first", fieldinfos[0].Provider)
}
