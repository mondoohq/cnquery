// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package luarocks

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLuaRocksList(t *testing.T) {
	f, err := os.Open("testdata/luarocks_list.txt")
	require.NoError(t, err)
	defer f.Close()

	pkgs, fps := ParseLuaRocksList(f, "luarocks list")
	require.Len(t, pkgs, 4)

	assert.Equal(t, "luasocket", pkgs[0].Name)
	assert.Equal(t, "3.1.0-1", pkgs[0].Version)
	assert.Equal(t, "pkg:lua/luasocket@3.1.0-1", pkgs[0].Purl)

	// File paths extracted from ROCKS_DIR column
	assert.Len(t, fps, 4)

	assert.Equal(t, "lua-cjson", pkgs[1].Name)
	assert.Equal(t, "2.1.0.14-1", pkgs[1].Version)

	assert.Equal(t, "lpeg", pkgs[2].Name)
	assert.Equal(t, "luafilesystem", pkgs[3].Name)
}

func TestParseRocksDir(t *testing.T) {
	afs := &afero.Afero{Fs: afero.NewOsFs()}
	pkgs, fps := ParseRocksDir(afs, "testdata/rocks")
	require.Len(t, pkgs, 2)
	require.NotEmpty(t, fps)

	byName := map[string]string{}
	for _, p := range pkgs {
		byName[p.Name] = p.Version
	}

	assert.Equal(t, "3.1.0-1", byName["luasocket"])
	assert.Equal(t, "2.1.0.14-1", byName["lua-cjson"])

	// Verify PURL
	for _, p := range pkgs {
		if p.Name == "luasocket" {
			assert.Equal(t, "pkg:lua/luasocket@3.1.0-1", p.Purl)
		}
	}
}
