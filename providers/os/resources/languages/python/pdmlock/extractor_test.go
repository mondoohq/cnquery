// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package pdmlock

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSimple(t *testing.T) {
	f, err := os.Open("testdata/simple.toml")
	require.NoError(t, err)
	defer f.Close()

	e := &Extractor{}
	bom, err := e.Parse(f, "testdata/simple.toml")
	require.NoError(t, err)

	assert.Nil(t, bom.Root())
	assert.Nil(t, bom.Direct())

	pkgs := bom.Transitive()
	assert.Len(t, pkgs, 4)

	django := pkgs.Find("django")
	require.NotNil(t, django)
	assert.Equal(t, "4.2.7", django.Version)
	assert.Equal(t, "pkg:pypi/django@4.2.7", django.Purl)

	pytest := pkgs.Find("pytest")
	require.NotNil(t, pytest)
	assert.Equal(t, "7.4.3", pytest.Version)
}

func TestName(t *testing.T) {
	e := &Extractor{}
	assert.Equal(t, "pdmlock", e.Name())
}
