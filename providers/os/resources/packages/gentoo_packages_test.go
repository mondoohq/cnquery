// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGentooPackages(t *testing.T) {
	f, err := os.Open("testdata/gentoo_qlist.txt")
	require.NoError(t, err)

	m, err := ParseGentooPackages(f)
	require.Nil(t, err)
	assert.Equal(t, 13, len(m), "detected the right amount of packages")

	p := Package{
		Name:    "net-misc/curl",
		Version: "8.4.0",
		Format:  "gentoo",
	}
	assert.Contains(t, m, p)
}
