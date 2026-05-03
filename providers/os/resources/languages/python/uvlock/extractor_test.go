// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package uvlock

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSimple(t *testing.T) {
	f, err := os.Open("testdata/simple.lock")
	require.NoError(t, err)
	defer f.Close()

	e := &Extractor{}
	bom, err := e.Parse(f, "testdata/simple.lock")
	require.NoError(t, err)

	root := bom.Root()
	require.NotNil(t, root)
	assert.Equal(t, "my-project", root.Name)
	assert.Equal(t, "0.1.0", root.Version)

	assert.Nil(t, bom.Direct())

	pkgs := bom.Transitive()
	assert.Len(t, pkgs, 4)

	django := pkgs.Find("django")
	require.NotNil(t, django)
	assert.Equal(t, "4.2.7", django.Version)
	assert.Equal(t, "pkg:pypi/django@4.2.7", django.Purl)
	assert.Len(t, django.EvidenceList, 1)
}

func TestName(t *testing.T) {
	e := &Extractor{}
	assert.Equal(t, "uvlock", e.Name())
}
