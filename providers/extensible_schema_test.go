// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/resources"
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

	s.prioritizeIDs("second")

	info := s.Lookup("eternity")
	require.NotNil(t, info)
	require.Equal(t, "second", info.Provider)

	_, finfo := s.LookupField("eternity", "iii")
	require.NotNil(t, info)
	require.Equal(t, "second", finfo.Provider)

	_, finfo = s.LookupField("eternity", "v")
	require.NotNil(t, info)
	require.Equal(t, "first", finfo.Provider)

	s.prioritizeIDs("first")

	_, finfo = s.LookupField("eternity", "iii")
	require.NotNil(t, info)
	require.Equal(t, "first", finfo.Provider)
}
