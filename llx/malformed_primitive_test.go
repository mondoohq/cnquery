// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/types"
)

func TestPrimitiveHasNoType(t *testing.T) {
	tests := []struct {
		name string
		prim *Primitive
		want bool
	}{
		{"nil primitive", nil, false},
		{"typed primitive", StringPrimitive("a"), false},
		{"genuine null keeps its 1-byte type", NilPrimitive, false},
		{"typeless primitive", &Primitive{}, true},
		{
			name: "typeless element inside a typed array",
			prim: &Primitive{Type: string(types.Array(types.String)), Array: []*Primitive{
				StringPrimitive("ok"),
				{}, // malformed element
			}},
			want: true,
		},
		{
			name: "typeless value inside a typed map",
			prim: &Primitive{Type: string(types.Map(types.String, types.String)), Map: map[string]*Primitive{
				"good": StringPrimitive("ok"),
				"bad":  {},
			}},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, primitiveHasNoType(tc.prim))
		})
	}
}

func TestReportMalformedPrimitives(t *testing.T) {
	// Capture the global logger so we can assert on the diagnostic.
	var buf bytes.Buffer
	orig := zlog.Logger
	zlog.Logger = zerolog.New(&buf)
	defer func() { zlog.Logger = orig }()

	// A code bundle whose block 1, chunk 2 carries a typeless primitive — the
	// shape the compiler bug produces.
	code := &CodeV2{
		Id: "//query/checksum/abc",
		Blocks: []*Block{
			{
				Chunks: []*Chunk{
					{Call: Chunk_PRIMITIVE, Id: "healthy", Primitive: StringPrimitive("ok")},
					{Call: Chunk_PRIMITIVE, Id: "brokenField", Primitive: &Primitive{}},
				},
			},
		},
		Checksums: map[uint64]string{
			(1 << 32) | 2: "chunk-checksum-xyz",
		},
	}

	reportMalformedPrimitives(code, "//assets/asset-42")

	out := buf.String()
	require.Contains(t, out, "malformed primitive")
	require.Contains(t, out, "//assets/asset-42", "asset MRN must be in the diagnostic")
	require.Contains(t, out, "//query/checksum/abc", "query id must be in the diagnostic")
	require.Contains(t, out, "brokenField", "offending field/chunk id must be in the diagnostic")
	require.Contains(t, out, "chunk-checksum-xyz", "chunk checksum must be in the diagnostic")
	// Only the malformed chunk is reported, not the healthy one.
	require.Equal(t, 1, strings.Count(out, "malformed primitive"))
}

func TestReportMalformedPrimitives_clean(t *testing.T) {
	var buf bytes.Buffer
	orig := zlog.Logger
	zlog.Logger = zerolog.New(&buf)
	defer func() { zlog.Logger = orig }()

	code := &CodeV2{
		Id: "//query/ok",
		Blocks: []*Block{
			{Chunks: []*Chunk{
				{Call: Chunk_PRIMITIVE, Id: "a", Primitive: StringPrimitive("ok")},
				{Call: Chunk_FUNCTION, Id: "==", Function: &Function{Args: []*Primitive{BoolPrimitive(true)}}},
			}},
		},
	}

	reportMalformedPrimitives(code, "//assets/asset-1")
	require.Empty(t, buf.String(), "well-typed code must not produce any diagnostic")
}
