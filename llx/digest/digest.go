// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package digest computes canonical, structural content digests for llx
// values. It is the single source of truth for the scan-content digest
// algorithm shared by cnspec (computing digests while writing the scandb)
// and the server (stamping/verifying uploaded files) — see the
// unchanged-scan-short-circuit ADR in the server repo.
//
// Canonicalization is structural, never over marshaled proto bytes: proto
// serialization (even Deterministic) is not guaranteed stable across
// library versions, so nested encoded messages (Dict payloads, AssetValue)
// are decoded and re-walked. Every variable-length write is
// length-prefixed, so concatenation ambiguity ("ab"+"c" vs "a"+"bc")
// cannot collide. Kind literals domain-separate the hash spaces.
package digest

import (
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"
	"math"
	"sort"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// AlgoVersion identifies the canonicalization algorithm. It is a fold input:
// digests produced by different versions never compare equal. Bump on any
// change to the canonicalization rules below.
//
// v1: FNV-1a 64, length-prefixed writes, arrays hashed as multisets
// (per-element sub-digests, sorted) — provider enumeration order is not
// content. Hashing arrays in order produces false "changed" verdicts when
// e.g. macOS app enumeration returns the same package set in a different
// order.
const AlgoVersion = "1"

// Hasher is a length-prefixed, domain-separated hash writer over FNV-1a 64
// (stdlib hash/fnv — no third-party dependency). AlgoVersion covers a later
// switch to a stronger or wider hash.
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

// CanonPrimitive hashes an llx.Primitive structurally. Scalar Value bytes
// are hashed as-is (they are llx's own canonical scalar encodings, not
// proto-marshaled messages). Dict and Asset primitives carry nested encoded
// messages in Value; those are decoded and re-walked, never hashed raw.
// Arrays are hashed in order; maps key-sorted.
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
			if err := CanonProto(h, av); err != nil {
				return err
			}
		} else {
			h.Str("asset:empty")
		}
	default:
		h.Bytes(p.Value)
	}

	// Arrays are hashed as multisets: each element is hashed independently
	// and the sorted element digests are folded in. Provider enumeration
	// order is not content — the same set in a different order must produce
	// the same digest. Duplicates still count (multiset, not set).
	h.writeLen(len(p.Array))
	if len(p.Array) > 0 {
		sums := make([]uint64, len(p.Array))
		for i, el := range p.Array {
			eh := NewHasher("array_element")
			if err := CanonPrimitive(eh, el); err != nil {
				return err
			}
			sums[i] = eh.Sum64()
		}
		sort.Slice(sums, func(i, j int) bool { return sums[i] < sums[j] })
		for _, s := range sums {
			h.U64(s)
		}
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

// CanonProto hashes any proto message structurally via reflection: fields
// in field-number order, messages recursed, lists in order, maps key-sorted.
// Unknown fields are ignored (messages hashed here were decoded by this
// binary from its own schema). This exists so callers never hash marshaled
// proto bytes.
func CanonProto(h *Hasher, m proto.Message) error {
	if m == nil {
		h.Str("<nil>")
		return nil
	}
	return canonReflect(h, m.ProtoReflect())
}

func canonReflect(h *Hasher, m protoreflect.Message) error {
	if !m.IsValid() {
		h.Str("<nil>")
		return nil
	}
	type fv struct {
		fd protoreflect.FieldDescriptor
		v  protoreflect.Value
	}
	var fields []fv
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		fields = append(fields, fv{fd, v})
		return true
	})
	sort.Slice(fields, func(i, j int) bool { return fields[i].fd.Number() < fields[j].fd.Number() })

	h.writeLen(len(fields))
	for _, f := range fields {
		h.U32(uint32(f.fd.Number()))
		if err := canonValue(h, f.fd, f.v); err != nil {
			return err
		}
	}
	return nil
}

func canonValue(h *Hasher, fd protoreflect.FieldDescriptor, v protoreflect.Value) error {
	switch {
	case fd.IsMap():
		type kv struct {
			key string
			mk  protoreflect.MapKey
		}
		var keys []kv
		v.Map().Range(func(mk protoreflect.MapKey, _ protoreflect.Value) bool {
			keys = append(keys, kv{mk.String(), mk})
			return true
		})
		sort.Slice(keys, func(i, j int) bool { return keys[i].key < keys[j].key })
		h.writeLen(len(keys))
		for _, k := range keys {
			h.Str(k.key)
			if err := canonSingular(h, fd.MapValue(), v.Map().Get(k.mk)); err != nil {
				return err
			}
		}
		return nil
	case fd.IsList():
		l := v.List()
		h.writeLen(l.Len())
		for i := 0; i < l.Len(); i++ {
			if err := canonSingular(h, fd, l.Get(i)); err != nil {
				return err
			}
		}
		return nil
	default:
		return canonSingular(h, fd, v)
	}
}

func canonSingular(h *Hasher, fd protoreflect.FieldDescriptor, v protoreflect.Value) error {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		h.Bool(v.Bool())
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		h.I64(v.Int())
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		h.U64(v.Uint())
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		h.F64(v.Float())
	case protoreflect.EnumKind:
		h.I64(int64(v.Enum()))
	case protoreflect.StringKind:
		h.Str(v.String())
	case protoreflect.BytesKind:
		h.Bytes(v.Bytes())
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return canonReflect(h, v.Message())
	default:
		return fmt.Errorf("canon: unsupported proto kind %s", fd.Kind())
	}
	return nil
}

// HashDataRow computes the row digest for a data row:
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

// HashResourceRow computes the row digest for a resource row. Created and
// Updated are wall-clock noise and deliberately excluded.
func HashResourceRow(rec *llx.ResourceRecording) (uint64, error) {
	h := NewHasher("row:resource")
	if rec == nil {
		h.Str("<nil>")
		return h.Sum64(), nil
	}
	h.Str(rec.Resource).Str(rec.Id)

	keys := make([]string, 0, len(rec.Fields))
	for k := range rec.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h.writeLen(len(keys))
	for _, k := range keys {
		h.Str(k)
		if err := CanonResult(h, rec.Fields[k]); err != nil {
			return 0, err
		}
	}
	return h.Sum64(), nil
}
