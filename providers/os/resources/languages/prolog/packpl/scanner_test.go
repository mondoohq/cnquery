// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packpl

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanPackDir(t *testing.T) {
	afs := &afero.Afero{Fs: afero.NewOsFs()}
	packs, err := ScanPackDir(afs, "./testdata")
	require.NoError(t, err)

	// pack1 + pack2 = 2 packs; not-a-pack is skipped
	assert.Equal(t, 2, len(packs))

	var mavis, listUtil *PrologPack
	for i := range packs {
		switch packs[i].Name {
		case "mavis":
			mavis = &packs[i]
		case "list_util":
			listUtil = &packs[i]
		}
	}

	require.NotNil(t, mavis)
	assert.Equal(t, "mavis", mavis.Name)
	assert.Equal(t, "1.0.5", mavis.Version)
	assert.Equal(t, "Example Prolog pack", mavis.Title)

	require.NotNil(t, listUtil)
	assert.Equal(t, "list_util", listUtil.Name)
	assert.Equal(t, "0.15.0", listUtil.Version)
}

func TestToPackages(t *testing.T) {
	packs := []PrologPack{
		{Name: "mavis", Version: "1.0.5", FilePath: "pack.pl"},
	}
	pkgs := ToPackages(packs)
	assert.Equal(t, 1, len(pkgs))
	assert.Equal(t, "mavis", pkgs[0].Name)
	assert.Equal(t, "1.0.5", pkgs[0].Version)
	assert.Equal(t, "pkg:swi-prolog/mavis@1.0.5", pkgs[0].Purl)
}
