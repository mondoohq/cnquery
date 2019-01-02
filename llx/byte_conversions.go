package llx

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math"
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
