// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
)

func TestMrnBasenameOrMrn(t *testing.T) {
	mrn := "//captain.api.mondoo.app/organizations/romantic-hopper-662653"
	base := mrnBasenameOrMrn(mrn)
	require.Equal(t, "romantic-hopper-662653", base)

	mrn = "romantic-hopper-662653"
	base = mrnBasenameOrMrn(mrn)
	require.Equal(t, "romantic-hopper-662653", base)
}

func TestFillAsset(t *testing.T) {
	t.Run("mondoo organization asset", func(t *testing.T) {
		a := &inventory.Asset{}
		mrn := "//captain.api.mondoo.app/organizations/romantic-hopper-662653"
		err := fillAsset(mrn, a)
		require.NoError(t, err)
		require.Equal(t, "Mondoo Organization romantic-hopper-662653", a.Name)

		expectedPlatform := &inventory.Platform{
			Name:    "mondoo-organization",
			Title:   "Mondoo Organization",
			Family:  []string{},
			Kind:    "api",
			Runtime: "mondoo",
			Labels:  map[string]string{},
		}
		require.Equal(t, expectedPlatform, a.Platform)
	})

	t.Run("mondoo space asset", func(t *testing.T) {
		a := &inventory.Asset{}
		mrn := "//captain.api.mondoo.app/spaces/romantic-hopper-662653"
		err := fillAsset(mrn, a)
		require.NoError(t, err)
		require.Equal(t, "Mondoo Space romantic-hopper-662653", a.Name)

		expectedPlatform := &inventory.Platform{
			Name:    "mondoo-space",
			Title:   "Mondoo Space",
			Family:  []string{},
			Kind:    "api",
			Runtime: "mondoo",
			Labels:  map[string]string{},
		}
		require.Equal(t, expectedPlatform, a.Platform)
	})

	t.Run("no asset (invalid MRN)", func(t *testing.T) {
		a := &inventory.Asset{}
		mrn := "//captain.api.mondoo.app/serviceaccounts/romantic-hopper-662653"
		err := fillAsset(mrn, a)
		require.Error(t, err)
	})
}
