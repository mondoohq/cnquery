// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package firewalld

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFirewalldParser(t *testing.T) {
	publicZone, err := os.ReadFile("./testdata/public.xml")
	require.NoError(t, err)
	zone, err := ParseZone(publicZone)
	require.NoError(t, err)
	require.Equal(t, "default", zone.Target)

	trustedZone, err := os.ReadFile("./testdata/trusted.xml")
	require.NoError(t, err)
	zone, err = ParseZone(trustedZone)
	require.NoError(t, err)
	require.Equal(t, "ACCEPT", zone.Target)
}
