// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAixPackages(t *testing.T) {
	f, err := os.Open("testdata/packages_aix.txt")
	require.NoError(t, err)

	m, err := parseAixPackages(f)
	require.Nil(t, err)
	assert.Equal(t, 16, len(m), "detected the right amount of packages")

	var p Package
	p = Package{
		Name:        "X11.apps.msmit",
		Version:     "7.3.0.0",
		Description: "AIXwindows msmit Application",
		Format:      "bff",
	}
	assert.Contains(t, m, p)
}
