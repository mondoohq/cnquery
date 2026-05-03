// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package poetrylock

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

	assert.Nil(t, bom.Root())
	assert.Nil(t, bom.Direct())

	pkgs := bom.Transitive()
	assert.Len(t, pkgs, 5)

	django := pkgs.Find("django")
	require.NotNil(t, django)
	assert.Equal(t, "4.2.7", django.Version)
	assert.Equal(t, "pkg:pypi/django@4.2.7", django.Purl)
	assert.Len(t, django.EvidenceList, 1)
	assert.Equal(t, "testdata/simple.lock", django.EvidenceList[0].Value)

	requests := pkgs.Find("requests")
	require.NotNil(t, requests)
	assert.Equal(t, "2.31.0", requests.Version)
	assert.Equal(t, "pkg:pypi/requests@2.31.0", requests.Purl)
}

func TestParseWithExtras(t *testing.T) {
	f, err := os.Open("testdata/with-extras.lock")
	require.NoError(t, err)
	defer f.Close()

	e := &Extractor{}
	bom, err := e.Parse(f, "testdata/with-extras.lock")
	require.NoError(t, err)

	pkgs := bom.Transitive()
	assert.Len(t, pkgs, 3)

	toolbar := pkgs.Find("django-debug-toolbar")
	require.NotNil(t, toolbar)
	assert.Equal(t, "4.2.0", toolbar.Version)
}

func TestName(t *testing.T) {
	e := &Extractor{}
	assert.Equal(t, "poetrylock", e.Name())
}
