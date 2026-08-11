// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package digest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

func hex(d uint64) string { return fmt.Sprintf("%016x", d) }

func strResult(codeID, s string) *llx.Result {
	return &llx.Result{
		CodeId: codeID,
		Data:   llx.StringPrimitive(s),
	}
}

// TestHashDataRowGolden pins the algorithm's output bits. These vectors are
// the shared contract with every consumer that recomputes digests (the
// server's port pins the same values): if a refactor changes any of them, it
// changed the algorithm — bump AlgoVersion and update all consumers, or
// revert.
func TestHashDataRowGolden(t *testing.T) {
	d, err := HashDataRow("code-1", strResult("code-1", "hello"))
	require.NoError(t, err)
	assert.Equal(t, "5ef31dae55c5be30", hex(d))

	// The row key is a fold input: same payload under another key differs.
	d2, err := HashDataRow("code-2", strResult("code-2", "hello"))
	require.NoError(t, err)
	assert.NotEqual(t, d, d2)

	// The error field is a fold input.
	d3, err := HashDataRow("code-1", &llx.Result{CodeId: "code-1", Error: "boom"})
	require.NoError(t, err)
	assert.NotEqual(t, d, d3)

	// A nil result is well-defined.
	d4, err := HashDataRow("code-1", nil)
	require.NoError(t, err)
	assert.Equal(t, "e74573eb6b333daf", hex(d4))
}

// TestArraysAreMultisets is AlgoVersion 2's defining property: provider
// enumeration order is not content.
func TestArraysAreMultisets(t *testing.T) {
	arr := func(vals ...string) *llx.Result {
		prims := make([]*llx.Primitive, len(vals))
		for i, v := range vals {
			prims[i] = llx.StringPrimitive(v)
		}
		return &llx.Result{Data: &llx.Primitive{Type: string(types.Array(types.String)), Array: prims}}
	}

	ab, err := HashDataRow("k", arr("a", "b"))
	require.NoError(t, err)
	ba, err := HashDataRow("k", arr("b", "a"))
	require.NoError(t, err)
	assert.Equal(t, ab, ba, "same multiset, different order: must be equal")

	// Duplicates still count: {a,a,b} != {a,b,b}.
	aab, err := HashDataRow("k", arr("a", "a", "b"))
	require.NoError(t, err)
	abb, err := HashDataRow("k", arr("a", "b", "b"))
	require.NoError(t, err)
	assert.NotEqual(t, aab, abb, "multiset, not set: duplicate counts matter")
}

// TestDictPayloadIsDecoded: Dict payloads are nested encoded primitives and
// must be structurally re-walked, never hashed as raw bytes.
func TestDictPayloadIsDecoded(t *testing.T) {
	inner := llx.StringPrimitive("payload")
	raw, err := inner.MarshalVT()
	require.NoError(t, err)

	dict := &llx.Result{Data: &llx.Primitive{Type: string(types.Dict), Value: raw}}
	d1, err := HashDataRow("k", dict)
	require.NoError(t, err)

	// A dict wrapping the same inner value must hash deterministically.
	d2, err := HashDataRow("k", dict)
	require.NoError(t, err)
	assert.Equal(t, d1, d2)

	// Garbage that does not decode is a hard error — never a silent
	// passthrough of raw bytes.
	bad := &llx.Result{Data: &llx.Primitive{Type: string(types.Dict), Value: []byte{0xff, 0xff, 0xff}}}
	_, err = HashDataRow("k", bad)
	assert.Error(t, err)
}

// TestLengthPrefixing: length-prefixed writes mean concatenation ambiguity
// cannot collide ("ab"+"c" vs "a"+"bc").
func TestLengthPrefixing(t *testing.T) {
	h1 := NewHasher("t").Str("ab").Str("c").Sum64()
	h2 := NewHasher("t").Str("a").Str("bc").Sum64()
	assert.NotEqual(t, h1, h2)

	// The domain literal separates hash spaces.
	assert.NotEqual(t, NewHasher("x").Str("v").Sum64(), NewHasher("y").Str("v").Sum64())
}

func TestHashResourceRow(t *testing.T) {
	rec := &llx.ResourceRecording{Resource: "user", Id: "root"}
	d1, err := HashResourceRow(rec)
	require.NoError(t, err)
	d2, err := HashResourceRow(rec)
	require.NoError(t, err)
	assert.Equal(t, d1, d2, "deterministic")

	other := &llx.ResourceRecording{Resource: "user", Id: "admin"}
	d3, err := HashResourceRow(other)
	require.NoError(t, err)
	assert.NotEqual(t, d1, d3, "resource id is a fold input")
}
