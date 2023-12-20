// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCosPackages(t *testing.T) {
	f, err := os.Open("testdata/cos-package-info.json")
	require.NoError(t, err)

	m, err := ParseCosPackages(f)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(m), "detected the right amount of packages")

	p := Package{
		Name:    "zlib",
		Version: "1.2.11-r4",
		Arch:    "",
		Format:  "cos",
	}
	assert.Contains(t, m, p)
}
