// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
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
	assert.Equal(t, 17, len(m), "detected the right amount of packages")

	p := Package{
		Name:        "X11.apps.msmit",
		Arch:        "powerpc",
		Version:     "7.3.0.0",
		Description: "AIXwindows msmit Application",
		PUrl:        "pkg:generic/aix/X11.apps.msmit@7.3.0.0?arch=powerpc",
		CPEs: []string{
			"cpe:2.3:a:x11.apps.msmit:x11.apps.msmit:7.3.0.0:*:*:*:*:*:powerpc:*",
			"cpe:2.3:a:x11.apps.msmit:x11.apps.msmit:7.3.0:*:*:*:*:*:powerpc:*",
		},
		Format: "bff",
		Status: "COMMITTED",
	}
	assert.Contains(t, m, p)

	p = Package{
		Name:        "bos.sysmgt.nim.client",
		Arch:        "powerpc",
		Version:     "7.3.3.0",
		Description: "Network Install Manager - Client Tools",
		PUrl:        "pkg:generic/aix/bos.sysmgt.nim.client@7.3.3.0?arch=powerpc&efix=locked",
		CPEs: []string{
			"cpe:2.3:a:bos.sysmgt.nim.client:bos.sysmgt.nim.client:7.3.3.0:*:*:*:*:*:powerpc:*",
			"cpe:2.3:a:bos.sysmgt.nim.client:bos.sysmgt.nim.client:7.3.3:*:*:*:*:*:powerpc:*",
		},
		Format: "bff",
		Status: "COMMITTED|EFIXLOCKED",
	}
	assert.Contains(t, m, p)
}
