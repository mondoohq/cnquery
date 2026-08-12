// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package checksum computes canonical, structural content checksums over llx
// values — the shared algorithm behind the scan-content manifests of the
// unchanged-scan short-circuit (ADR in the server repo). This package owns
// everything expressible in llx terms: the hash writer, the canonicalization
// of llx primitives/results, and the data/resource row checksums. Checksums
// over types that live outside mql (cnspec's scores and risk factors) build
// on this package from cnspec's policy/scandb/checksum. Consumers never
// re-implement any of this: two implementations of one algorithm drift, and
// checksum drift silently reports identical scans as changed (or worse).
//
// Canonicalization is structural and explicit: every hashed message has a
// hand-written canon function that names each field it folds in. There is
// deliberately no reflection walk (protoreflect) and no hashing of marshaled
// proto bytes. Marshaled bytes are out because proto serialization (even
// Deterministic) is not guaranteed stable across library versions; a
// reflection walk is out because it changes meaning silently the moment a
// message grows a field — an explicit canon makes "is this field content?"
// a reviewed decision, visible in a diff.
//
// Every variable-length write is length-prefixed, so concatenation ambiguity
// ("ab"+"c" vs "a"+"bc") cannot collide. Domain literals separate the hash
// spaces. Lists whose order is not content are hashed as multisets: sorted
// per-element sub-checksums, so the same set in a different enumeration
// order produces the same checksum while duplicates still count.
package checksum

import (
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"
	"math"
	"slices"
	"sort"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/protobuf/proto"
)

// AlgoVersion identifies the canonicalization algorithm. It is a fold input:
// checksums produced by different versions never compare equal. Bump on any
// change to the canonicalization rules below.
//
// v1: FNV-1a 64 (stdlib hash/fnv — no third-party dependency),
// length-prefixed writes, explicit canonicalization of the fixed message
// set, arrays hashed as multisets (sorted per-element sub-checksums —
// provider enumeration order is not content).
const AlgoVersion = "1"

// Hasher is a length-prefixed, domain-separated hash writer over FNV-1a 64.
// AlgoVersion covers a later switch to a stronger or wider hash.
type Hasher struct {
	h hash.Hash64
}

// NewHasher returns a Hasher domain-separated by the given literal.
func NewHasher(domain string) *Hasher {
	h := &Hasher{h: fnv.New64a()}
	h.Str(domain)
	return h
}

func (h *Hasher) writeLen(n int) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(n))
	h.h.Write(buf[:]) //nolint:errcheck // hash.Hash.Write never errors
}

// Str writes a length-prefixed string.
func (h *Hasher) Str(s string) *Hasher {
	h.writeLen(len(s))
	h.h.Write([]byte(s)) //nolint:errcheck
	return h
}

// Bytes writes a length-prefixed byte slice.
func (h *Hasher) Bytes(b []byte) *Hasher {
	h.writeLen(len(b))
	h.h.Write(b) //nolint:errcheck
	return h
}

// U64 writes a fixed-width uint64.
func (h *Hasher) U64(v uint64) *Hasher {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], v)
	h.h.Write(buf[:]) //nolint:errcheck
	return h
}

// U32 writes a fixed-width uint32 (widened).
func (h *Hasher) U32(v uint32) *Hasher { return h.U64(uint64(v)) }

// I64 writes a fixed-width int64.
func (h *Hasher) I64(v int64) *Hasher { return h.U64(uint64(v)) }

// F64 writes a float64 by bit pattern.
func (h *Hasher) F64(v float64) *Hasher { return h.U64(math.Float64bits(v)) }

// Bool writes a single canonical byte.
func (h *Hasher) Bool(v bool) *Hasher {
	if v {
		return h.U64(1)
	}
	return h.U64(0)
}

// Sum64 finalizes the hash.
func (h *Hasher) Sum64() uint64 { return h.h.Sum64() }

// Multiset hashes a list as a multiset: each element hashed independently
// under domain, the sorted element checksums folded in. Element order is not
// content — the same set in a different order must produce the same
// checksum. Duplicates still count (multiset, not set). Exported so
// downstream canon functions (cnspec's score checksums) fold their lists
// with the exact same encoding.
func Multiset[T any](h *Hasher, domain string, items []T, canon func(*Hasher, T) error) error {
	h.writeLen(len(items))
	if len(items) == 0 {
		return nil
	}
	sums := make([]uint64, len(items))
	for i, el := range items {
		eh := NewHasher(domain)
		if err := canon(eh, el); err != nil {
			return err
		}
		sums[i] = eh.Sum64()
	}
	slices.Sort(sums)
	for _, s := range sums {
		h.U64(s)
	}
	return nil
}

// CanonResultMap hashes a key/result map key-sorted. Exported for the same
// reason as Multiset: downstream canon functions carry result maps too.
func CanonResultMap(h *Hasher, m map[string]*llx.Result) error {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h.writeLen(len(keys))
	for _, k := range keys {
		h.Str(k)
		if err := CanonResult(h, m[k]); err != nil {
			return err
		}
	}
	return nil
}

// CanonPrimitive hashes an llx.Primitive structurally. Scalar Value bytes
// are hashed as-is (they are llx's own canonical scalar encodings, not
// proto-marshaled messages). Dict and Asset primitives carry nested encoded
// messages in Value; those are decoded and re-walked, never hashed raw.
// Arrays are hashed as multisets; maps key-sorted.
func CanonPrimitive(h *Hasher, p *llx.Primitive) error {
	if p == nil {
		h.Str("<nil>")
		return nil
	}
	h.Str(p.Type)

	switch types.Type(p.Type) {
	case types.Dict:
		if len(p.Value) > 0 {
			inner := &llx.Primitive{}
			if err := proto.Unmarshal(p.Value, inner); err != nil {
				return fmt.Errorf("canon: failed to decode dict payload: %w", err)
			}
			h.Str("dict")
			if err := CanonPrimitive(h, inner); err != nil {
				return err
			}
		} else {
			h.Str("dict:empty")
		}
	case types.Asset:
		if len(p.Value) > 0 {
			av := &llx.AssetValue{}
			if err := proto.Unmarshal(p.Value, av); err != nil {
				return fmt.Errorf("canon: failed to decode asset payload: %w", err)
			}
			h.Str("asset")
			h.Str(av.ResourceType).Str(av.ResourceId)
		} else {
			h.Str("asset:empty")
		}
	default:
		h.Bytes(p.Value)
	}

	if err := Multiset(h, "array_element", p.Array, CanonPrimitive); err != nil {
		return err
	}

	if len(p.Map) > 0 {
		keys := make([]string, 0, len(p.Map))
		for k := range p.Map {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		h.writeLen(len(keys))
		for _, k := range keys {
			h.Str(k)
			if err := CanonPrimitive(h, p.Map[k]); err != nil {
				return err
			}
		}
	} else {
		h.writeLen(0)
	}

	return nil
}

// CanonResult hashes an llx.Result's error and data. The CodeId is NOT
// hashed here — callers hash the row key themselves (it is the map key /
// primary key at every call site).
func CanonResult(h *Hasher, r *llx.Result) error {
	if r == nil {
		h.Str("<nil>")
		return nil
	}
	h.Str(r.Error)
	return CanonPrimitive(h, r.Data)
}

// HashDataRow computes the row checksum for a data row:
// H("row:data" ‖ code_id ‖ error ‖ canon(data)).
func HashDataRow(codeID string, r *llx.Result) (uint64, error) {
	h := NewHasher("row:data")
	h.Str(codeID)
	if r == nil {
		h.Str("<nil>")
		return h.Sum64(), nil
	}
	h.Str(r.Error)
	if err := CanonPrimitive(h, r.Data); err != nil {
		return 0, err
	}
	return h.Sum64(), nil
}

// HashResourceRow computes the row checksum for a resource row. Created and
// Updated are wall-clock noise and deliberately excluded.
func HashResourceRow(rec *llx.ResourceRecording) (uint64, error) {
	h := NewHasher("row:resource")
	if rec == nil {
		h.Str("<nil>")
		return h.Sum64(), nil
	}
	h.Str(rec.Resource).Str(rec.Id)
	if err := CanonResultMap(h, rec.Fields); err != nil {
		return 0, err
	}
	return h.Sum64(), nil
}
