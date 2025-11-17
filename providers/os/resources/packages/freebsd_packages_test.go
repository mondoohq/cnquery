// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFreeBSDPackages(t *testing.T) {
	f, err := os.Open("testdata/freebsd-package-info-streaming.txt")
	require.NoError(t, err)

	m, err := ParseFreeBSDPackages(f)
	require.Nil(t, err)
	assert.Equal(t, 32, len(m), "detected the right amount of packages")

	p := Package{
		Name:        "brotli",
		Version:     "1.1.0,1",
		Arch:        "FreeBSD:14:amd64",
		Format:      "freebsd",
		Description: "Generic-purpose lossless compression algorithm",
		Origin:      "archivers/brotli",
	}
	assert.Contains(t, m, p)
}
