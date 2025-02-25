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
	Mask int
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
	return IP{
		IP:   ip,
		Mask: mask,
	}
}

var bitmasks = []byte{0x00, 0x80, 0xc0, 0xe0, 0xf0, 0xf8, 0xfc, 0xfe, 0xff}

func createMask(bits int, maxBytes int) []byte {
	i := bits
	var res []byte
	for ; i >= 8 && len(res) < maxBytes; i -= 8 {
		res = append(res, 0xff)
	}
	if i > 0 && len(res) < maxBytes {
		res = append(res, bitmasks[i])
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
	b := createMask(i.Mask, 4)
	return mask2string(b)
}

func (i IP) prefix() string {
	b := createMask(i.Mask, 4)
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
	b := createMask(i.Mask, 4)
	mask := flipMask(net.IPMask(b))
	suffix := i.IP.Mask(mask)
	return suffix.String()
}

func ipVersion(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}

	v := NewIP(bind.Value.(string))
	if v.IP.To4() != nil {
		return IntData(4), 0, nil
	}
	if v.IP.To16() != nil {
		return IntData(6), 0, nil
	}
	return IntData(0), 0, nil
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
