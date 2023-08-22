// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cat

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGetItemFile(t *testing.T) {
	data, err := os.Open("./testdata/getitem_file.json")
	require.NoError(t, err)

	m, err := ParseGetItem(data)
	assert.Nil(t, err)

	assert.Equal(t, "test.txt", m.Name)
	assert.Equal(t, uint32(32), m.Attributes)
}

func TestParseGetItemDir(t *testing.T) {
	data, err := os.Open("./testdata/getitem_dir.json")
	require.NoError(t, err)

	m, err := ParseGetItem(data)
	assert.Nil(t, err)

	assert.Equal(t, "Windows", m.Name)
	assert.Equal(t, uint32(16), m.Attributes)
}
