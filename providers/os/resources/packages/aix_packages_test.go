// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"os"
	"testing"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAixPackages(t *testing.T) {
	f, err := os.Open("testdata/packages_aix.txt")
	require.NoError(t, err)

	pf := &inventory.Platform{
		Name:    "aix",
		Version: "7.2",
		Arch:    "powerpc",
	}

	m, err := parseAixPackages(pf, f)
	require.Nil(t, err)
	assert.Equal(t, 16, len(m), "detected the right amount of packages")

	p := Package{
		Name:        "X11.apps.msmit",
		Version:     "7.3.0.0",
		Description: "AIXwindows msmit Application",
		PUrl:        "pkg:generic/aix/X11.apps.msmit@7.3.0.0?arch=powerpc",
		CPEs: []string{
			"cpe:2.3:a:x11.apps.msmit:x11.apps.msmit:7.3.0.0:*:*:*:*:*:powerpc:*",
			"cpe:2.3:a:x11.apps.msmit:x11.apps.msmit:7.3.0:*:*:*:*:*:powerpc:*",
		},
		Format: "bff",
	}
	assert.Contains(t, m, p)
}
