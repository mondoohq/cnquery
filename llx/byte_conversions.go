// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math"
	"time"
)

func bool2bytes(b bool) []byte {
	if b {
		return []byte{1}
	}
	return []byte{0}
}

func bytes2bool(b []byte) bool {
	return len(b) > 0 && b[0] > 0
}

func int2bytes(i int64) []byte {
	v := make([]byte, binary.MaxVarintLen64)
	n := binary.PutVarint(v, i)
	return v[:n]
}

func bytes2int(b []byte) int64 {
	r := bytes.NewReader(b)
	res, err := binary.ReadVarint(r)
	if err != nil {
		panic("Failed to read bytes into integer: '" + hex.EncodeToString(b) + "'\n")
	}
	return res
}

func float2bytes(f float64) []byte {
	var v [8]byte
	binary.LittleEndian.PutUint64(v[:], math.Float64bits(f))
	return v[:]
}

func bytes2float(b []byte) float64 {
	return math.Float64frombits(binary.LittleEndian.Uint64(b))
}

func bytes2time(b []byte) time.Time {
	// A valid encoding is 12 bytes: 8 for seconds, 4 for nanoseconds.
	// Fall back to the zero time on a short buffer instead of slicing
	// out of range, matching ptime2raw's handling of an empty value.
	if len(b) < 12 {
		return time.Unix(0, 0)
	}
	secs := int64(binary.LittleEndian.Uint64(b[0:8]))
	nanos := int64(binary.LittleEndian.Uint32(b[8:12]))
	return time.Unix(secs, nanos)
}
