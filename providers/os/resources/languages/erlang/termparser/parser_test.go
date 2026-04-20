// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package termparser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSimpleList(t *testing.T) {
	node, err := Parse(`[1, 2, 3]`)
	require.NoError(t, err)
	assert.Equal(t, NodeList, node.Type)
	assert.Equal(t, 3, node.Len())
	assert.Equal(t, "1", node.Get(0).Str())
	assert.Equal(t, "2", node.Get(1).Str())
	assert.Equal(t, "3", node.Get(2).Str())
}

func TestParseTuple(t *testing.T) {
	node, err := Parse(`{<<"cowboy">>, {pkg, <<"cowboy">>, <<"2.10.0">>}, 0}`)
	require.NoError(t, err)
	assert.Equal(t, NodeTuple, node.Type)
	assert.Equal(t, 3, node.Len())

	// First element: binary string <<"cowboy">>
	assert.Equal(t, "cowboy", node.Get(0).Str())

	// Second element: nested tuple {pkg, <<"cowboy">>, <<"2.10.0">>}
	inner := node.Get(1)
	assert.Equal(t, NodeTuple, inner.Type)
	assert.Equal(t, 3, inner.Len())
	assert.Equal(t, "pkg", inner.Get(0).Str())
	assert.Equal(t, "cowboy", inner.Get(1).Str())
	assert.Equal(t, "2.10.0", inner.Get(2).Str())

	// Third element: number
	assert.Equal(t, "0", node.Get(2).Str())
}

func TestParseRebarLock(t *testing.T) {
	input := `[{<<"cowboy">>,{pkg,<<"cowboy">>,<<"2.10.0">>,<<"ABC123">>},0},
 {<<"cowlib">>,{pkg,<<"cowlib">>,<<"2.12.1">>,<<"DEF456">>},0},
 {<<"ranch">>,{pkg,<<"ranch">>,<<"2.1.0">>,<<"GHI789">>},0}].`

	node, err := Parse(input)
	require.NoError(t, err)
	assert.Equal(t, NodeList, node.Type)
	assert.Equal(t, 3, node.Len())

	// First package
	pkg0 := node.Get(0)
	assert.Equal(t, NodeTuple, pkg0.Type)
	assert.Equal(t, "cowboy", pkg0.Get(0).Str())

	pkgInfo := pkg0.Get(1)
	assert.Equal(t, "cowboy", pkgInfo.Get(1).Str())
	assert.Equal(t, "2.10.0", pkgInfo.Get(2).Str())

	// Third package
	pkg2 := node.Get(2)
	assert.Equal(t, "ranch", pkg2.Get(0).Str())
	assert.Equal(t, "2.1.0", pkg2.Get(1).Get(2).Str())
}

func TestParseWithComments(t *testing.T) {
	input := `% This is a comment
[{<<"test">>, <<"1.0.0">>}].`

	node, err := Parse(input)
	require.NoError(t, err)
	assert.Equal(t, NodeList, node.Type)
	assert.Equal(t, 1, node.Len())
}

func TestParseQuotedAtom(t *testing.T) {
	node, err := Parse(`{'hello world', "test"}`)
	require.NoError(t, err)
	assert.Equal(t, NodeTuple, node.Type)
	assert.Equal(t, "hello world", node.Get(0).Str())
	assert.Equal(t, NodeAtom, node.Get(0).Type)
	assert.Equal(t, "test", node.Get(1).Str())
	assert.Equal(t, NodeString, node.Get(1).Type)
}

func TestParseEmpty(t *testing.T) {
	node, err := Parse(`[]`)
	require.NoError(t, err)
	assert.Equal(t, NodeList, node.Type)
	assert.Equal(t, 0, node.Len())
}

func TestGetOutOfRange(t *testing.T) {
	node, err := Parse(`[1]`)
	require.NoError(t, err)
	assert.Nil(t, node.Get(5))
	assert.Nil(t, node.Get(-1))

	var nilNode *Node
	assert.Nil(t, nilNode.Get(0))
	assert.Equal(t, "", nilNode.Str())
	assert.Equal(t, 0, nilNode.Len())
}

func TestParseVersionedRebarLock(t *testing.T) {
	// Newer rebar3 format with version header
	input := `{<<"1.2.0">>,
[{<<"cowboy">>,{pkg,<<"cowboy">>,<<"2.10.0">>,<<"HASH">>},0},
 {<<"jsx">>,{pkg,<<"jsx">>,<<"3.1.0">>,<<"HASH">>},1}]}.`

	node, err := Parse(input)
	require.NoError(t, err)
	// Top level is a tuple: {version, [deps]}
	assert.Equal(t, NodeTuple, node.Type)
	assert.Equal(t, "1.2.0", node.Get(0).Str())

	deps := node.Get(1)
	assert.Equal(t, NodeList, deps.Type)
	assert.Equal(t, 2, deps.Len())
	assert.Equal(t, "cowboy", deps.Get(0).Get(0).Str())
}
