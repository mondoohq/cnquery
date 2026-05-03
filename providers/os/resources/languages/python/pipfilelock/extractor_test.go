// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package pipfilelock

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSimple(t *testing.T) {
	f, err := os.Open("testdata/simple.json")
	require.NoError(t, err)
	defer f.Close()

	e := &Extractor{}
	bom, err := e.Parse(f, "testdata/simple.json")
	require.NoError(t, err)

	assert.Nil(t, bom.Root())

	direct := bom.Direct()
	assert.Len(t, direct, 3)

	django := direct.Find("django")
	require.NotNil(t, django)
	assert.Equal(t, "4.2.7", django.Version)
	assert.Equal(t, "pkg:pypi/django@4.2.7", django.Purl)

	// Transitive includes both default and develop, with deduplication.
	all := bom.Transitive()
	assert.Len(t, all, 4) // 3 default + 1 develop (django is deduplicated)

	pytest := all.Find("pytest")
	require.NotNil(t, pytest)
	assert.Equal(t, "7.4.3", pytest.Version)
	assert.Equal(t, "pkg:pypi/pytest@7.4.3", pytest.Purl)
}

func TestName(t *testing.T) {
	e := &Extractor{}
	assert.Equal(t, "pipfilelock", e.Name())
}
