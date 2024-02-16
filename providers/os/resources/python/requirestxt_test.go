// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package python

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiresTxt(t *testing.T) {
	data := `
nose>=1.2
Mock>=1.0
pycryptodome

[crypto]
pycryptopp>=0.5.12

[cryptography]
cryptography
`

	dependencies, err := ParseRequiresTxtDependencies(strings.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, []string{"nose", "Mock", "pycryptodome"}, dependencies)
}
