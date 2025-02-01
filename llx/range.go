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
func RangePrimitive(data RangeData) *Primitive {
	return &Primitive{
		Type:  string(types.Range),
		Value: data,
	}
}

type RangeData []byte

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

func NewRange() RangeData {
	return []byte{}
}

func (r RangeData) AddLine(line uint32) RangeData {
	r = append(r, rangeVersion1|0x01)
	bytes := int2bytes(int64(line))
	r = append(r, byte(len(bytes)<<4))
	r = append(r, bytes...)
	return r
}

func (r RangeData) AddLineRange(line1 uint32, line2 uint32) RangeData {
	r = append(r, rangeVersion1|0x02)
	bytes1 := int2bytes(int64(line1))
	bytes2 := int2bytes(int64(line2))
	r = append(r, byte(len(bytes1)<<4)|byte(len(bytes2)&0x0f))
	r = append(r, bytes1...)
	r = append(r, bytes2...)
	return r
}

func (r RangeData) AddColumnRange(line uint32, column1 uint32, column2 uint32) RangeData {
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

func (r RangeData) AddLineColumnRange(line1 uint32, line2 uint32, column1 uint32, column2 uint32) RangeData {
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

func (r RangeData) ExtractNext() ([]uint32, RangeData) {
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
			n := bytes2int(r[idx : idx+l1])
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
			n := bytes2int(r[idx : idx+l1])
			idx += l2
			res = append(res, uint32(n))
		}

	default:
		log.Error().Msg("failed to extract range, wrong number of entries")
		return nil, nil
	}

	return res, r[idx:]
}

func (r RangeData) ExtractAll() [][]uint32 {
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

func (r RangeData) String() string {
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

	return res.String()
}
