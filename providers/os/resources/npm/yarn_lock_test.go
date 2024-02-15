// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package npm

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestYarnParser(t *testing.T) {
	f, err := os.Open("./testdata/yarn-lock/d3-yarn.lock")
	require.NoError(t, err)
	defer f.Close()

	_, pkgs, err := (&YarnLockParser{}).Parse(f)
	assert.Nil(t, err)
	assert.Equal(t, 99, len(pkgs))

	p := findPkg(pkgs, "has")
	assert.Equal(t, &Package{
		Name:    "has",
		Version: "1.0.3",
	}, p)

	p = findPkg(pkgs, "iconv-lite")
	assert.Equal(t, &Package{
		Name:    "iconv-lite",
		Version: "0.4.24",
	}, p)
}

func TestParsePackagename(t *testing.T) {
	var name string
	var version string
	var err error

	name, version, err = parseYarnPackageName("source-map-support@~0.5.10")
	assert.Nil(t, err)
	assert.Equal(t, "source-map-support", name)
	assert.Equal(t, "~0.5.10", version)

	name, version, err = parseYarnPackageName("@types/node@*")
	assert.Nil(t, err)
	assert.Equal(t, "@types/node", name)
	assert.Equal(t, "*", version)

	name, version, err = parseYarnPackageName("@babel/code-frame@^7.0.0-beta.47")
	assert.Nil(t, err)
	assert.Equal(t, "@babel/code-frame", name)
	assert.Equal(t, "^7.0.0-beta.47", version)

	name, version, err = parseYarnPackageName("has@^1.0.1, has@^1.0.3, has@~1.0.3")
	assert.Nil(t, err)
	assert.Equal(t, "has", name)
	assert.Equal(t, "^1.0.1", version)
}
