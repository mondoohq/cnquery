// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package requirements

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSetupPy_QuotedDeps(t *testing.T) {
	data := `
from setuptools import setup

setup(
    name="myproject",
    version="1.0.0",
    install_requires=[
        'requests==2.31.0',
        "openai==1.30.0",
        'transformers==4.40.0;python_version>="3.8"',
        "numpy>=1.21",
    ],
)
`
	reqs, err := ParseSetupPy(strings.NewReader(data))
	require.NoError(t, err)

	// Should find 3 pinned deps, not numpy (not pinned with ==)
	require.Len(t, reqs, 3)

	byName := map[string]string{}
	for _, r := range reqs {
		byName[r.Name] = r.Version
	}
	assert.Equal(t, "2.31.0", byName["requests"])
	assert.Equal(t, "1.30.0", byName["openai"])
	assert.Equal(t, "4.40.0", byName["transformers"])
}

func TestParseSetupPy_UnquotedDeps(t *testing.T) {
	data := `
mypy == v0.770
black == 23.3.0
`
	reqs, err := ParseSetupPy(strings.NewReader(data))
	require.NoError(t, err)
	require.Len(t, reqs, 2)
	assert.Equal(t, "mypy", reqs[0].Name)
	assert.Equal(t, "v0.770", reqs[0].Version)
	assert.Equal(t, "black", reqs[1].Name)
	assert.Equal(t, "23.3.0", reqs[1].Version)
}

func TestParseSetupPy_MixedQuotedAndUnquoted(t *testing.T) {
	data := `
setup(
    install_requires=['requests==2.31.0'],
)
mypy == v0.770
`
	reqs, err := ParseSetupPy(strings.NewReader(data))
	require.NoError(t, err)
	require.Len(t, reqs, 2)
}

func TestParseSetupPy_Templates(t *testing.T) {
	data := `
setup(
    install_requires=[
        'requests==%s' % req_version,
        '${package}==1.0.0',
        'openai==1.30.0',
    ],
)
`
	reqs, err := ParseSetupPy(strings.NewReader(data))
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Equal(t, "openai", reqs[0].Name)
}

func TestParseSetupPy_Deduplication(t *testing.T) {
	data := `
    'requests==2.31.0',
    "requests==2.31.0",
`
	reqs, err := ParseSetupPy(strings.NewReader(data))
	require.NoError(t, err)
	require.Len(t, reqs, 1)
}

func TestParseSetupPy_Empty(t *testing.T) {
	reqs, err := ParseSetupPy(strings.NewReader(""))
	require.NoError(t, err)
	assert.Empty(t, reqs)
}

func TestParseSetupPy_SetupCfg(t *testing.T) {
	data := `
[options]
install_requires =
    requests==2.31.0
    openai==1.30.0
`
	reqs, err := ParseSetupPy(strings.NewReader(data))
	require.NoError(t, err)
	require.Len(t, reqs, 2)
	assert.Equal(t, "requests", reqs[0].Name)
	assert.Equal(t, "openai", reqs[1].Name)
}
