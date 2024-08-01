// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMrnBasenameOrMrn(t *testing.T) {
	mrn := "//captain.api.mondoo.app/spaces/magical-nobel-298952"
	base := MrnBasenameOrMrn(mrn)
	require.Equal(t, "magical-nobel-298952", base)

	mrn = "magical-nobel-298952"
	base = MrnBasenameOrMrn(mrn)
	require.Equal(t, "magical-nobel-298952", base)
}

func TestDetermineConnTyp(t *testing.T) {
	t.Run("organization type", func(t *testing.T) {
		mrn := "//captain.api.mondoo.app/organizations/magical-nobel-298952"
		typ, err := determineConnType(mrn)
		require.NoError(t, err)
		require.Equal(t, ConnTypeOrganization, typ)
	})

	t.Run("space type", func(t *testing.T) {
		mrn := "//captain.api.mondoo.app/spaces/magical-nobel-298952"
		typ, err := determineConnType(mrn)
		require.NoError(t, err)
		require.Equal(t, ConnTypeSpace, typ)
	})

	t.Run("unknown type", func(t *testing.T) {
		mrn := "//captain.api.mondoo.app/serviceaccounts/magical-nobel-298952"
		_, err := determineConnType(mrn)
		require.Error(t, err)
	})
}
