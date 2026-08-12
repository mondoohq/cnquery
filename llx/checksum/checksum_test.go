// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package checksum

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/protobuf/proto"
)

func hex(d uint64) string { return fmt.Sprintf("%016x", d) }

func strResult(codeID, s string) *llx.Result {
	return &llx.Result{
		CodeId: codeID,
		Data:   llx.StringPrimitive(s),
	}
}

// TestHashDataRowGolden pins the algorithm's output bits. These vectors are
// the shared contract with every consumer that recomputes checksums (cnspec's
// scandb writer and the server pin the same package): if a refactor changes
// any of them, it changed the algorithm — bump AlgoVersion and update all
// consumers, or revert.
func TestHashDataRowGolden(t *testing.T) {
	d, err := HashDataRow("code-1", strResult("code-1", "hello"))
	require.NoError(t, err)
	assert.Equal(t, "bc7e2a9cf30d5d03", hex(d))

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
	assert.Equal(t, "211d667e209d5ed6", hex(d4))
}

// TestArraysAreMultisets is the algorithm's defining property: provider
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

// TestAssetValueIsExplicit: Asset payloads are decoded and hashed
// field-by-field — both identity fields are fold inputs.
func TestAssetValueIsExplicit(t *testing.T) {
	asset := func(rt, id string) *llx.Result {
		raw, err := (&llx.AssetValue{ResourceType: rt, ResourceId: id}).MarshalVT()
		require.NoError(t, err)
		return &llx.Result{Data: &llx.Primitive{Type: string(types.Asset), Value: raw}}
	}

	d1, err := HashDataRow("k", asset("aws.instance", "i-1"))
	require.NoError(t, err)
	d2, err := HashDataRow("k", asset("aws.instance", "i-2"))
	require.NoError(t, err)
	d3, err := HashDataRow("k", asset("gcp.instance", "i-1"))
	require.NoError(t, err)
	assert.NotEqual(t, d1, d2)
	assert.NotEqual(t, d1, d3)

	same, err := HashDataRow("k", asset("aws.instance", "i-1"))
	require.NoError(t, err)
	assert.Equal(t, d1, same)
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

func hashPrim(t *testing.T, p *llx.Primitive) uint64 {
	t.Helper()
	h := NewHasher("test")
	require.NoError(t, CanonPrimitive(h, p))
	return h.Sum64()
}

// TestDeterminismPerType covers every value shape CanonPrimitive
// canonicalizes. Each case constructs the primitive FRESH per call — two
// independent constructions must checksum identically (structural equality,
// not pointer identity), and the mutated variant must differ.
func TestDeterminismPerType(t *testing.T) {
	t0 := time.Unix(1700000000, 0).UTC()
	t1 := time.Unix(1700000001, 0).UTC()
	mustAsset := func(av *llx.AssetValue) *llx.Primitive {
		raw, err := proto.Marshal(av)
		require.NoError(t, err)
		return &llx.Primitive{Type: string(types.Asset), Value: raw}
	}
	mustDict := func(inner *llx.Primitive) *llx.Primitive {
		raw, err := inner.MarshalVT()
		require.NoError(t, err)
		return &llx.Primitive{Type: string(types.Dict), Value: raw}
	}

	cases := []struct {
		name    string
		make    func() *llx.Primitive
		mutated func() *llx.Primitive
	}{
		{"bool", func() *llx.Primitive { return llx.BoolPrimitive(true) },
			func() *llx.Primitive { return llx.BoolPrimitive(false) }},
		{"int", func() *llx.Primitive { return llx.IntPrimitive(42) },
			func() *llx.Primitive { return llx.IntPrimitive(43) }},
		{"float", func() *llx.Primitive { return llx.FloatPrimitive(3.14) },
			func() *llx.Primitive { return llx.FloatPrimitive(3.15) }},
		{"string", func() *llx.Primitive { return llx.StringPrimitive("hello") },
			func() *llx.Primitive { return llx.StringPrimitive("hellO") }},
		{"regex", func() *llx.Primitive { return llx.RegexPrimitive(".*") },
			func() *llx.Primitive { return llx.RegexPrimitive(".+") }},
		{"time", func() *llx.Primitive { return llx.TimePrimitive(&t0) },
			func() *llx.Primitive { return llx.TimePrimitive(&t1) }},
		{"nil-typed", func() *llx.Primitive { return &llx.Primitive{Type: string(types.Nil)} },
			func() *llx.Primitive { return llx.StringPrimitive("") }},
		{"ref", func() *llx.Primitive { return llx.RefPrimitiveV2(7) },
			func() *llx.Primitive { return llx.RefPrimitiveV2(8) }},
		{"score", func() *llx.Primitive { return llx.ScorePrimitive(80) },
			func() *llx.Primitive { return llx.ScorePrimitive(81) }},
		{"array", func() *llx.Primitive {
			return llx.ArrayPrimitive([]*llx.Primitive{llx.StringPrimitive("a"), llx.IntPrimitive(1)}, types.Any)
		}, func() *llx.Primitive {
			return llx.ArrayPrimitive([]*llx.Primitive{llx.StringPrimitive("a"), llx.IntPrimitive(2)}, types.Any)
		}},
		{"map", func() *llx.Primitive {
			return llx.MapPrimitive(map[string]*llx.Primitive{"k1": llx.StringPrimitive("v1"), "k2": llx.IntPrimitive(2)}, types.Any)
		}, func() *llx.Primitive {
			return llx.MapPrimitive(map[string]*llx.Primitive{"k1": llx.StringPrimitive("v1"), "k2": llx.IntPrimitive(3)}, types.Any)
		}},
		{"dict", func() *llx.Primitive { return mustDict(llx.StringPrimitive("payload")) },
			func() *llx.Primitive { return mustDict(llx.StringPrimitive("payloaD")) }},
		{"asset", func() *llx.Primitive {
			return mustAsset(&llx.AssetValue{ResourceType: "aws.ec2.instance", ResourceId: "i-1"})
		}, func() *llx.Primitive {
			return mustAsset(&llx.AssetValue{ResourceType: "aws.ec2.instance", ResourceId: "i-2"})
		}},
	}

	seen := map[uint64]string{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d1 := hashPrim(t, tc.make())
			d2 := hashPrim(t, tc.make())
			assert.Equal(t, d1, d2, "two independent constructions must checksum identically")
			assert.NotEqual(t, d1, hashPrim(t, tc.mutated()), "mutated value must differ")
			if prev, dup := seen[d1]; dup {
				t.Fatalf("checksum collides with case %q", prev)
			}
			seen[d1] = tc.name
		})
	}
}

// TestMapOrderInsensitivity: map canonicalization sorts keys. Go randomizes
// map iteration per range, so hashing the same map repeatedly would already
// diverge if the sort were missing; two maps built in reverse insertion
// orders must checksum identically too.
func TestMapOrderInsensitivity(t *testing.T) {
	build := func(reverse bool) *llx.Primitive {
		m := map[string]*llx.Primitive{}
		for i := 0; i < 20; i++ {
			idx := i
			if reverse {
				idx = 19 - i
			}
			m[fmt.Sprintf("key-%02d", idx)] = llx.IntPrimitive(int64(idx))
		}
		return llx.MapPrimitive(m, types.Int)
	}

	want := hashPrim(t, build(false))
	for i := 0; i < 25; i++ {
		assert.Equal(t, want, hashPrim(t, build(false)), "iteration %d", i)
	}
	assert.Equal(t, want, hashPrim(t, build(true)), "insertion order is not content")

	changed := build(false)
	changed.Map["key-07"] = llx.IntPrimitive(99)
	assert.NotEqual(t, want, hashPrim(t, changed), "a changed value must change the checksum")
}

// TestArrayOrderInsensitivityMixed: the multiset property holds for arrays of
// mixed element types, including nested arrays and maps, under arbitrary
// permutations (fixed-seed shuffles keep the test deterministic).
func TestArrayOrderInsensitivityMixed(t *testing.T) {
	t0 := time.Unix(1700000000, 0).UTC()
	elements := []*llx.Primitive{
		llx.BoolPrimitive(true),
		llx.IntPrimitive(7),
		llx.FloatPrimitive(2.5),
		llx.StringPrimitive("mixed"),
		llx.TimePrimitive(&t0),
		llx.ArrayPrimitive([]*llx.Primitive{llx.StringPrimitive("nested"), llx.IntPrimitive(1)}, types.Any),
		llx.MapPrimitive(map[string]*llx.Primitive{"k": llx.StringPrimitive("v")}, types.String),
		llx.StringPrimitive("mixed"), // duplicate on purpose: multiset, not set
	}

	arr := func(els []*llx.Primitive) *llx.Primitive { return llx.ArrayPrimitive(els, types.Any) }
	want := hashPrim(t, arr(elements))

	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 10; i++ {
		shuffled := make([]*llx.Primitive, len(elements))
		copy(shuffled, elements)
		rng.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })
		assert.Equal(t, want, hashPrim(t, arr(shuffled)), "shuffle %d", i)
	}

	// Dropping the duplicate changes the multiset.
	assert.NotEqual(t, want, hashPrim(t, arr(elements[:len(elements)-1])))
}

// BenchmarkHashDataRow_Array500 measures a realistic big row: one data row
// whose payload is a 500-element array (the package-list shape that motivates
// the multiset fold).
func BenchmarkHashDataRow_Array500(b *testing.B) {
	strEls := make([]*llx.Primitive, 500)
	for i := range strEls {
		strEls[i] = llx.StringPrimitive(fmt.Sprintf("package-%03d 1.2.%d-r0 x86_64 registry.example.com/library", i, i))
	}
	strRow := &llx.Result{CodeId: "bench", Data: llx.ArrayPrimitive(strEls, types.String)}

	mapEls := make([]*llx.Primitive, 500)
	for i := range mapEls {
		mapEls[i] = llx.MapPrimitive(map[string]*llx.Primitive{
			"name":    llx.StringPrimitive(fmt.Sprintf("package-%03d", i)),
			"version": llx.StringPrimitive(fmt.Sprintf("1.2.%d-r0", i)),
			"arch":    llx.StringPrimitive("x86_64"),
			"epoch":   llx.IntPrimitive(int64(i)),
		}, types.Any)
	}
	mapRow := &llx.Result{CodeId: "bench", Data: llx.ArrayPrimitive(mapEls, types.Any)}

	b.Run("strings", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := HashDataRow("bench", strRow); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("maps", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := HashDataRow("bench", mapRow); err != nil {
				b.Fatal(err)
			}
		}
	})
}
