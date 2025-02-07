// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/types"
)

// RangePrimitive creates a range primitive from the given
// range data. Use the helper functions to initialize and
// combine multiple sets of range data.
func RangePrimitive(data Range) *Primitive {
	return &Primitive{
		Type:  string(types.Range),
		Value: data,
	}
}

type Range []byte

const (
	// Byte indicators for ranges work like this:
	//
	// Byte1:    version + mode
	// xxxx xxxx
	// VVVV -------> version for the range
	//      MMMM --> 1 = single line
	//               2 = line range
	//               3 = line with column range
	//               4 = line + column range
	//
	// Byte2+:   length indicators
	// xxxx xxxx
	// NNNN -------> length of the first entry (up to 128bit)
	//      MMMM --> length of the second entry (up to 128bit)
	//               note: currently we only support up to 32bit
	//
	rangeVersion1 byte = 0x10
)

func NewRange() Range {
	return []byte{}
}

func (r Range) AddLine(line uint32) Range {
	r = append(r, rangeVersion1|0x01)
	bytes := int2bytes(int64(line))
	r = append(r, byte(len(bytes)<<4))
	r = append(r, bytes...)
	return r
}

func (r Range) AddLineRange(line1 uint32, line2 uint32) Range {
	r = append(r, rangeVersion1|0x02)
	bytes1 := int2bytes(int64(line1))
	bytes2 := int2bytes(int64(line2))
	r = append(r, byte(len(bytes1)<<4)|byte(len(bytes2)&0x0f))
	r = append(r, bytes1...)
	r = append(r, bytes2...)
	return r
}

func (r Range) AddColumnRange(line uint32, column1 uint32, column2 uint32) Range {
	r = append(r, rangeVersion1|0x03)
	bytes := int2bytes(int64(line))
	bytes1 := int2bytes(int64(column1))
	bytes2 := int2bytes(int64(column2))

	r = append(r, byte(len(bytes)<<4))
	r = append(r, bytes...)

	r = append(r, byte(len(bytes1)<<4)|byte(len(bytes2)&0xf))
	r = append(r, bytes1...)
	r = append(r, bytes2...)
	return r
}

func (r Range) AddLineColumnRange(line1 uint32, line2 uint32, column1 uint32, column2 uint32) Range {
	r = append(r, rangeVersion1|0x04)
	bytes1 := int2bytes(int64(line1))
	bytes2 := int2bytes(int64(line2))
	r = append(r, byte(len(bytes1)<<4)|byte(len(bytes2)&0xf))
	r = append(r, bytes1...)
	r = append(r, bytes2...)

	bytes1 = int2bytes(int64(column1))
	bytes2 = int2bytes(int64(column2))
	r = append(r, byte(len(bytes1)<<4)|byte(len(bytes2)&0xf))
	r = append(r, bytes1...)
	r = append(r, bytes2...)

	return r
}

func (r Range) ExtractNext() ([]uint32, Range) {
	if len(r) == 0 {
		return nil, nil
	}

	version := r[0] & 0xf0
	if version != rangeVersion1 {
		log.Error().Msg("failed to extract range, version is unsupported")
		return nil, nil
	}

	entries := r[0] & 0x0f
	res := []uint32{}
	idx := 1
	switch entries {
	case 3, 4:
		l1 := int((r[idx] & 0xf0) >> 4)
		l2 := int(r[idx] & 0x0f)

		idx++
		if l1 != 0 {
			n := bytes2int(r[idx : idx+l1])
			idx += l1
			res = append(res, uint32(n))
		}
		if l2 != 0 {
			n := bytes2int(r[idx : idx+l2])
			idx += l2
			res = append(res, uint32(n))
		}

		fallthrough

	case 1, 2:
		l1 := int((r[idx] & 0xf0) >> 4)
		l2 := int(r[idx] & 0x0f)

		idx++
		if l1 != 0 {
			n := bytes2int(r[idx : idx+l1])
			idx += l1
			res = append(res, uint32(n))
		}
		if l2 != 0 {
			n := bytes2int(r[idx : idx+l2])
			idx += l2
			res = append(res, uint32(n))
		}

	default:
		log.Error().Msg("failed to extract range, wrong number of entries")
		return nil, nil
	}

	return res, r[idx:]
}

func (r Range) ExtractAll() [][]uint32 {
	res := [][]uint32{}
	for {
		cur, rest := r.ExtractNext()
		if len(cur) != 0 {
			res = append(res, cur)
		}
		if len(rest) == 0 {
			break
		}
		r = rest
	}

	return res
}

func (r Range) String() string {
	var res strings.Builder

	items := r.ExtractAll()
	for i := range items {
		x := items[i]
		switch len(x) {
		case 1:
			res.WriteString(strconv.Itoa(int(x[0])))
		case 2:
			res.WriteString(strconv.Itoa(int(x[0])))
			res.WriteString("-")
			res.WriteString(strconv.Itoa(int(x[1])))
		case 3:
			res.WriteString(strconv.Itoa(int(x[0])))
			res.WriteString(":")
			res.WriteString(strconv.Itoa(int(x[1])))
			res.WriteString("-")
			res.WriteString(strconv.Itoa(int(x[2])))
		case 4:
			res.WriteString(strconv.Itoa(int(x[0])))
			res.WriteString(":")
			res.WriteString(strconv.Itoa(int(x[2])))
			res.WriteString("-")
			res.WriteString(strconv.Itoa(int(x[1])))
			res.WriteString(":")
			res.WriteString(strconv.Itoa(int(x[3])))
		}

		if i != len(items)-1 {
			res.WriteString(",")
		}
	}

	ret := res.String()
	if ret == "" {
		return "undefined"
	}
	return ret
}

func (r Range) ExtractString(src string) string {
	lines := strings.Split(src, "\n")
	maxLines := uint32(len(lines))
	items := r.ExtractAll()
	var res strings.Builder
	for i := range items {
		x := items[i]
		switch len(x) {
		case 1:
			idx := x[0]
			if idx < uint32(len(lines)) {
				res.WriteString(lines[idx])
				res.WriteByte('\n')
			}

		case 2:
			end := uint32(len(lines) - 1)
			if x[1] < end {
				end = x[1]
			}
			for line := x[0]; line <= end; line++ {
				res.WriteString(lines[line])
				res.WriteByte('\n')
			}

		case 3:
			idx := x[0]
			if idx > maxLines {
				break
			}
			line := lines[idx]

			end := uint32(len(lines) - 1)
			if x[2] < end {
				end = x[2]
			}

			if x[1] < uint32(len(line)) {
				res.WriteString(line[x[1]:])
				res.WriteByte('\n')
			}
			idx++

			for ; idx <= end; idx++ {
				res.WriteString(lines[idx])
				res.WriteByte('\n')
			}

		case 4:
			idx := x[0]
			if idx > maxLines {
				break
			}
			line := lines[idx]

			end := uint32(len(lines) - 1)
			if x[2] < end {
				end = x[2]
			}

			c1 := x[1]
			c2 := x[3] + 1
			if idx == end {
				cMax := uint32(len(line))
				addNewline := false
				if c2 > cMax {
					c2 = cMax
					addNewline = idx < maxLines
				}
				if c1 < cMax {
					res.WriteString(line[c1:c2])
				}
				if addNewline {
					res.WriteByte('\n')
				}
				continue
			}

			if c1 < uint32(len(line)) {
				res.WriteString(line[c1:])
				res.WriteByte('\n')
			}
			idx++

			for ; idx < end; idx++ {
				res.WriteString(lines[idx])
				res.WriteByte('\n')
			}

			line = lines[end]
			addNewline := false
			if c2 > uint32(len(line)) {
				c2 = uint32(len(line))
				addNewline = end < maxLines
			}
			res.WriteString(line[:c2])
			if addNewline {
				res.WriteByte('\n')
			}
		}
	}

	return res.String()
}
