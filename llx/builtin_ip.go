// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"errors"
	"net"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v11/types"
)

type IP struct {
	net.IP
	Version int
	Mask    int
}

func NewIP(s string) IP {
	prefix := s
	suffix := ""
	if idx := strings.IndexByte(s, '/'); idx != -1 {
		prefix = s[0:idx]
		if len(s) > idx+1 {
			suffix = s[idx+1:]
		}
	}

	mask := -1
	if suffix != "" {
		mask64, _ := strconv.ParseInt(suffix, 10, 0)
		mask = int(mask64)
	}

	ip := net.ParseIP(prefix)

	version := 0
	if ip.To4() != nil {
		version = 4
		if mask == -1 {
			m := ip.DefaultMask()
			mask = countMaskBits(m)
		}
	} else if ip.To16() != nil {
		version = 6
		if mask == -1 {
			mask = 64
		}
	}

	return IP{
		IP:      ip,
		Version: version,
		Mask:    mask,
	}
}

var bitmasks = []byte{0x00, 0x80, 0xc0, 0xe0, 0xf0, 0xf8, 0xfc, 0xfe, 0xff}

func int2ip(i int64) string {
	cur := i

	d := byte(cur & 0xff)
	cur = cur >> 8

	c := byte(cur & 0xff)
	cur = cur >> 8

	b := byte(cur & 0xff)
	cur = cur >> 8

	a := byte(cur & 0xff)

	return net.IPv4(a, b, c, d).String()
}

func countMaskBits(b []byte) int {
	var res int
	for _, cur := range b {
		// optimization for speed
		if cur == 0xff {
			res += 8
			continue
		}
		if cur == 0 {
			break
		}
		// and the remaining bits
		for cur&0x80 != 0 {
			res++
			cur = cur << 1
		}
		break
	}
	return res
}

func makeBits(bits int, on bool) []byte {
	var res []byte
	var one byte
	if on {
		one = 0xff
	}
	for ; bits >= 8; bits -= 8 {
		res = append(res, one)
	}
	if bits > 0 {
		if on {
			res = append(res, bitmasks[bits])
		} else {
			res = append(res, ^bitmasks[bits])
		}
	}
	return res
}

func createMask(maskBits int, offsetBits int, maxBytes int) []byte {
	var res []byte

	// we need to see how many bits over max we are and remove them
	over := (offsetBits + maskBits) - maxBytes*8
	if over > 0 {
		maskBits -= over
		if maskBits < 0 {
			offsetBits += maskBits
		}
	}

	// create an offset of zero-bits first
	if offsetBits > 0 {
		res = makeBits(offsetBits, false)
		rem := offsetBits % 8
		if rem > 0 {
			maskBits -= 8 - rem // remaining bits in the byte, that were already part of the mask
		}
	}

	// then create the mask bits i.e. one-bits
	if maskBits > 0 {
		res = append(res, makeBits(maskBits, true)...)
	}

	for len(res) < maxBytes {
		res = append(res, 0x00)
	}

	return res
}

func mask2string(b []byte) string {
	var res strings.Builder
	for i, bi := range b {
		if i != 0 {
			res.WriteByte('.')
		}
		res.WriteString(strconv.Itoa(int(bi)))
	}
	return res.String()
}

func (i IP) Subnet() string {
	if i.Version == 4 {
		b := createMask(i.Mask, 0, 4)
		return mask2string(b)
	}

	// For IPv6 this is a bit tricky:
	// https://www.rfc-editor.org/rfc/rfc3587.txt
	// If the mask (i.e. prefix) is too large there is no subnet
	if i.Mask >= 64 {
		return ""
	}

	// for everything else we get the subnet from the remainder of
	// the first 64 bits that are not the prefix
	subnetBits := 64 - i.Mask
	b := createMask(subnetBits, i.Mask, 16)
	if len(b) == 0 {
		return ""
	}

	mask := net.IPMask(b)
	subnet := i.IP.Mask(mask)
	res := subnet.String()

	for hasMore := true; hasMore; {
		res, hasMore = strings.CutPrefix(res, "0:")
	}
	res, _ = strings.CutSuffix(res, "::")

	return res
}

func (i IP) prefix() net.IP {
	var b []byte
	if i.Version == 4 {
		b = createMask(i.Mask, 0, 4)
	} else if i.Version == 6 {
		b = createMask(i.Mask, 0, 16)
	}
	if len(b) == 0 {
		return []byte{}
	}
	mask := net.IPMask(b)
	return i.IP.Mask(mask)
}

func (i IP) Prefix() string {
	return i.prefix().String()
}

func flipMask(b []byte) []byte {
	for i := range b {
		b[i] = ^b[i]
	}
	return b
}

func (i IP) Suffix() string {
	var b []byte
	if i.Version == 4 {
		b = createMask(i.Mask, 0, 4)
	} else if i.Version == 6 {
		if i.Mask < 64 {
			b = createMask(65, 0, 16)
		} else {
			b = createMask(i.Mask, 0, 16)
		}
	}
	if len(b) == 0 {
		return ""
	}
	mask := flipMask(net.IPMask(b))
	suffix := i.IP.Mask(mask)
	return suffix.String()
}

func (i IP) inRange(other IP) bool {
	prefix := i.prefix()
	otherPrefix := other.prefix()
	return prefix.Equal(otherPrefix)
}

func (i IP) Cmp(other IP) int {
	for idx, b := range i.IP {
		if len(other.IP) <= idx {
			return 1
		}
		o := other.IP[idx]
		if b == o {
			continue
		}
		if b < o {
			return -1
		} else {
			return 1
		}
	}
	if len(other.IP) > len(i.IP) {
		return -1
	}
	return 0
}

func ipCmpIP(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left, right interface{}) *RawData {
		lip := NewIP(left.(string))
		rip := NewIP(right.(string))
		return BoolData(lip.Equal(rip.IP) && lip.Mask == rip.Mask)
	})
}

func ipNotIP(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left, right interface{}) *RawData {
		lip := NewIP(left.(string))
		rip := NewIP(right.(string))
		return BoolData(!lip.Equal(rip.IP) || lip.Mask != rip.Mask)
	})
}

func ipVersion(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}

	v := NewIP(bind.Value.(string))
	return IntData(v.Version), 0, nil
}

func ipSubnet(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}

	v := NewIP(bind.Value.(string))
	return StringData(v.Subnet()), 0, nil
}

func ipPrefix(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}

	v := NewIP(bind.Value.(string))
	return StringData(v.Prefix()), 0, nil
}

func ipPrefixLength(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}

	v := NewIP(bind.Value.(string))
	return IntData(v.Mask), 0, nil
}

func ipSuffix(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}

	v := NewIP(bind.Value.(string))
	return StringData(v.Suffix()), 0, nil
}

func ipUnspecified(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}

	v := NewIP(bind.Value.(string))
	return BoolData(v.IP.IsUnspecified()), 0, nil
}

func ipInRange(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}

	conditions := []string{}
	for i := range chunk.Function.Args {
		argRef := chunk.Function.Args[i]

		arg, rref, err := e.resolveValue(argRef, ref)
		if err != nil || rref > 0 {
			return nil, rref, err
		}

		s, ok := arg.Value.(string)
		if !ok {
			return nil, 0, errors.New("incorrect type for argument in `inRange` call (expected string)")
		}
		conditions = append(conditions, s)
	}

	v := NewIP(bind.Value.(string))
	if len(conditions) == 1 {
		i := NewIP(conditions[0])
		return BoolData(v.inRange(i)), 0, nil
	}

	min := NewIP(conditions[0])
	max := NewIP(conditions[1])

	mincmp := min.Cmp(v)
	if mincmp == 1 {
		return BoolFalse, 0, nil
	}
	maxcmp := v.Cmp(max)
	if maxcmp == 1 {
		return BoolFalse, 0, nil
	}

	return BoolTrue, 0, nil
}
