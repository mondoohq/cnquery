// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bunlock

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
	assert.Nil(t, bom.Direct())

	pkgs := bom.Transitive()
	assert.Len(t, pkgs, 3)

	express := pkgs.Find("express")
	require.NotNil(t, express)
	assert.Equal(t, "4.18.2", express.Version)
	assert.Equal(t, "pkg:npm/express@4.18.2", express.Purl)
}

func TestParseScoped(t *testing.T) {
	f, err := os.Open("testdata/scoped.json")
	require.NoError(t, err)
	defer f.Close()

	e := &Extractor{}
	bom, err := e.Parse(f, "testdata/scoped.json")
	require.NoError(t, err)

	pkgs := bom.Transitive()
	assert.Len(t, pkgs, 3)

	typesNode := pkgs.Find("@types/node")
	require.NotNil(t, typesNode)
	assert.Equal(t, "20.10.0", typesNode.Version)
	assert.Equal(t, "pkg:npm/%40types/node@20.10.0", typesNode.Purl)
}

func TestParseBunNameVersion(t *testing.T) {
	tests := []struct {
		input   string
		name    string
		version string
	}{
		{"express@4.18.2", "express", "4.18.2"},
		{"@types/node@20.10.0", "@types/node", "20.10.0"},
		{"@babel/core@7.23.5", "@babel/core", "7.23.5"},
		{"file:./local-pkg", "", ""},
		{"noversion", "", ""},
	}

	for _, tt := range tests {
		name, version := parseBunNameVersion(tt.input)
		assert.Equal(t, tt.name, name, "input: %s", tt.input)
		assert.Equal(t, tt.version, version, "input: %s", tt.input)
	}
}

func TestStripJSONC(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"trailing comma in object", `{"a": 1,}`, `{"a": 1}`},
		{"trailing comma in array", `[1, 2,]`, `[1, 2]`},
		{"no trailing comma", `{"a": 1}`, `{"a": 1}`},
		{"comma in string preserved", `{"a": "b,}"}`, `{"a": "b,}"}`},
		{"nested trailing commas", `{"a": [1,], "b": 2,}`, `{"a": [1], "b": 2}`},
		{"empty input", ``, ``},
		{"escaped quote in string", `{"a": "b\"c,}"}`, `{"a": "b\"c,}"}`},
	}

	for _, tt := range tests {
		got := string(stripJSONC([]byte(tt.input)))
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestName(t *testing.T) {
	e := &Extractor{}
	assert.Equal(t, "bunlock", e.Name())
}
