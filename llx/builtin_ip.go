// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
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
	mask := 0
	if idx := strings.IndexByte(s, '/'); idx != -1 {
		prefix = s[0:idx]
		if len(s) > idx+1 {
			suffix = s[idx+1:]
		}
	}

	if suffix != "" {
		mask64, _ := strconv.ParseInt(suffix, 10, 0)
		mask = int(mask64)
	}

	ip := net.ParseIP(prefix)

	version := 0
	if ip.To4() != nil {
		version = 4
	} else if ip.To16() != nil {
		version = 6
		if mask == 0 {
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

func (i IP) subnet() string {
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

func (i IP) prefix() string {
	var b []byte
	if i.Version == 4 {
		b = createMask(i.Mask, 0, 4)
	} else if i.Version == 6 {
		b = createMask(i.Mask, 0, 16)
	}
	if len(b) == 0 {
		return ""
	}
	mask := net.IPMask(b)
	prefix := i.IP.Mask(mask)
	return prefix.String()
}

func flipMask(b []byte) []byte {
	for i := range b {
		b[i] = ^b[i]
	}
	return b
}

func (i IP) suffix() string {
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

func ipCmpIP(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left, right interface{}) *RawData {
		return BoolData(left.(string) == right.(string))
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
	return StringData(v.subnet()), 0, nil
}

func ipPrefix(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}

	v := NewIP(bind.Value.(string))
	return StringData(v.prefix()), 0, nil
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
	return StringData(v.suffix()), 0, nil
}

func ipUnspecified(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}

	v := NewIP(bind.Value.(string))
	return BoolData(v.IP.IsUnspecified()), 0, nil
}
