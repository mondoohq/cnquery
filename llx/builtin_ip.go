// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"errors"
	"net"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v11/types"
	"google.golang.org/protobuf/proto"
)

type RawIP struct {
	net.IP
	Version         uint8 // 4 and 6, 0 == unset
	HasPrefixLength bool
	PrefixLength    int // -1 = unset
}

type constraintOp byte

const (
	UnknownOp constraintOp = 0
	LessThan  constraintOp = 1 << iota
	Equal
	MoreThan
)

type ipConstraint struct {
	operand constraintOp
	address RawIP
}

func ParseIP(s string) RawIP {
	prefix := s
	suffix := ""
	prefixProvided := false
	if idx := strings.IndexByte(s, '/'); idx != -1 {
		prefix = s[0:idx]
		if len(s) > idx+1 {
			suffix = s[idx+1:]
		}
	}

	var explicitMask int = -1
	if suffix != "" {
		mask64, _ := strconv.ParseInt(suffix, 10, 0)
		explicitMask = int(mask64)
		prefixProvided = true
	}

	ip := net.ParseIP(prefix)
	version, prefixLength := ipVersionPrefix(ip, explicitMask)
	return RawIP{
		IP:              ip,
		Version:         version,
		PrefixLength:    prefixLength,
		HasPrefixLength: prefixProvided,
	}
}

func ipVersionPrefix(ip net.IP, mask int) (uint8, int) {
	var version uint8
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
	return version, mask
}

func int2ip[T int | int64](i T) net.IP {
	cur := i

	d := byte(cur & 0xff)
	cur = cur >> 8

	c := byte(cur & 0xff)
	cur = cur >> 8

	b := byte(cur & 0xff)
	cur = cur >> 8

	a := byte(cur & 0xff)

	return net.IPv4(a, b, c, d)
}

func ParseIntIP[T int | int64](i T) RawIP {
	ip := int2ip(i)
	version, mask := ipVersionPrefix(ip, -1)
	return RawIP{
		IP:           ip,
		Version:      version,
		PrefixLength: mask,
	}
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
			res = append(res, ^(1<<(8-bits) - 1))
		} else {
			res = append(res, (1<<(8-bits) - 1))
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

func (i RawIP) Subnet() string {
	if i.Version == 4 {
		b := createMask(i.PrefixLength, 0, 4)
		return mask2string(b)
	}

	// For IPv6 this is a bit tricky:
	// https://www.rfc-editor.org/rfc/rfc3587.txt
	// If the mask (i.e. prefix) is too large there is no subnet
	if i.PrefixLength >= 64 {
		return ""
	}

	// for everything else we get the subnet from the remainder of
	// the first 64 bits that are not the prefix
	subnetBits := 64 - i.PrefixLength
	b := createMask(subnetBits, i.PrefixLength, 16)
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

func (i RawIP) prefix() net.IP {
	var b []byte
	if i.Version == 4 {
		b = createMask(i.PrefixLength, 0, 4)
	} else if i.Version == 6 {
		b = createMask(i.PrefixLength, 0, 16)
	}
	if len(b) == 0 {
		return []byte{}
	}
	mask := net.IPMask(b)
	return i.IP.Mask(mask)
}

func (i RawIP) Prefix() string {
	return i.prefix().String()
}

func flipMask(b []byte) []byte {
	for i := range b {
		b[i] = ^b[i]
	}
	return b
}

func (i RawIP) Suffix() string {
	var b []byte
	if i.Version == 4 {
		b = createMask(i.PrefixLength, 0, 4)
	} else if i.Version == 6 {
		if i.PrefixLength < 64 {
			b = createMask(65, 0, 16)
		} else {
			b = createMask(i.PrefixLength, 0, 16)
		}
	}
	if len(b) == 0 {
		return ""
	}
	mask := flipMask(net.IPMask(b))
	suffix := i.IP.Mask(mask)
	return suffix.String()
}

func (i RawIP) Ipv4Broadcast() net.IP {
	res := i.prefix()
	m := createMask(i.PrefixLength, 0, 4)
	for i := range m {
		res[i] = res[i] | (^m[i])
	}
	return res
}

// compare this IP to another IP, byte-wise step by step
// returns:
// - -1 if this IP is smaller than the other IP
// -  0 if this IP is equal to the other IP
// - +1 if this IP is larger than the other IP
func (i RawIP) CmpIP(other net.IP) int {
	for idx, b := range i.IP {
		if len(other) <= idx {
			return 1
		}
		o := other[idx]
		if b == o {
			continue
		}
		if b < o {
			return -1
		} else {
			return 1
		}
	}
	if len(other) > len(i.IP) {
		return -1
	}
	return 0
}

// similar to IP comparison, but optimizezd for subnets,
// compares every byte-segment in the IP with the other IP
// returns:
// - -1 if this IP is below the subnet
// -  0 if this IP is inside of the subnet (but not its any or broadcast IP)
// - +1 if this IP is above the subnet
// - -2 if this IP is the subnet's any IP (zero bits, eg 192.168.0.0/24)
// - +2 if this IP is the subnet's broadcast IP (one bits, eg 192.168.0.255)
// assummptions:
// - both RawIPs are stored at length 16, as IPv6
func (ip RawIP) CmpSubnet(subnet RawIP) int {
	// the number of bits inside of a byte that the subnet is offset by
	// eg: 20 => 16 + 4 i.e. 4 bits
	offBits := subnet.PrefixLength % 8
	// the prefix length is adjusted for IPv6, ie we assume the same IPv6 prefix,
	// which allows us to compare ipv4 and ipv6
	prefixLen := subnet.PrefixLength
	if subnet.Version == 4 {
		prefixLen += 12 * 8
	}

	bytePos := 0
	for ; bytePos*8 < prefixLen && bytePos < len(ip.IP); bytePos++ {
		// fast comparison for all prefix bits
		if bytePos*8+8 <= prefixLen {
			if ip.IP[bytePos] == subnet.IP[bytePos] {
				continue
			}
			if ip.IP[bytePos] < subnet.IP[bytePos] {
				return -1
			}
			return 1
		}

		// masked comparison for all remaining bits
		var mask byte = ^(1<<(8-offBits) - 1)
		ipM := ip.IP[bytePos] & mask
		subnetM := subnet.IP[bytePos] & mask
		if ipM == subnetM {
			break
		}
		if ipM < subnetM {
			return -1
		}
		return 1
	}

	// At this point we know the subnets are the same,
	// time to test the rest

	// Partial comparison of bits
	allOnes := true
	allZeros := true
	if offBits != 0 {
		var mask byte = 1<<(8-offBits) - 1
		rem := ip.IP[bytePos] & mask
		if rem == 0 {
			allOnes = false
		} else if rem == mask {
			allZeros = false
		} else {
			return 0
		}
		bytePos++
	}

	for ; bytePos < len(ip.IP) && (allOnes || allZeros); bytePos++ {
		cur := ip.IP[bytePos]
		if cur == 0 {
			allOnes = false
		} else if cur == 255 {
			allZeros = false
		} else {
			return 0
		}
	}

	if allZeros {
		return -2
	}
	if allOnes {
		return 2
	}
	return 0
}

func (i RawIP) Cmp(other RawIP) int {
	return i.CmpIP(other.IP)
}

func (i RawIP) Address() string {
	return i.IP.String()
}

func (i RawIP) CIDR() string {
	return i.IP.String() + "/" + strconv.Itoa(int(i.PrefixLength))
}

func (i RawIP) String() string {
	if i.HasPrefixLength {
		return i.IP.String() + "/" + strconv.Itoa(int(i.PrefixLength))
	}
	return i.IP.String()
}

func (i RawIP) Marshal() ([]byte, error) {
	data := &IP{
		Address:      i.IP,
		HasPrefix:    i.HasPrefixLength,
		PrefixLength: int32(i.PrefixLength),
	}

	return proto.Marshal(data)
}

func UnmarshalIP(bytes []byte) (*RawIP, error) {
	var data IP
	if err := proto.Unmarshal(bytes, &data); err != nil {
		return nil, err
	}

	var addr net.IP = data.Address
	var version uint8
	if addr.To4() != nil {
		version = 4
	} else if addr.To16() != nil {
		version = 6
	}

	return &RawIP{
		IP:              data.Address,
		HasPrefixLength: data.HasPrefix,
		PrefixLength:    int(data.PrefixLength),
		Version:         version,
	}, nil
}

func ipCmpIP(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpT(e, bind, chunk, ref, types.Bool, func(left, right RawIP) *RawData {
		return BoolData(left.Equal(right.IP) && left.PrefixLength == right.PrefixLength)
	})
}

func ipNotIP(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpT(e, bind, chunk, ref, types.Bool, func(left, right RawIP) *RawData {
		return BoolData(!left.Equal(right.IP) || left.PrefixLength != right.PrefixLength)
	})
}

func ipVersion(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}
	v, ok := bind.Value.(RawIP)
	if !ok {
		return nil, 0, errors.New("incorrect internal data for IP type")
	}
	return IntData(int(v.Version)), 0, nil
}

func ipSubnet(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}
	v, ok := bind.Value.(RawIP)
	if !ok {
		return nil, 0, errors.New("incorrect internal data for IP type")
	}
	return StringData(v.Subnet()), 0, nil
}

func ipPrefix(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}
	v, ok := bind.Value.(RawIP)
	if !ok {
		return nil, 0, errors.New("incorrect internal data for IP type")
	}
	return StringData(v.Prefix()), 0, nil
}

func ipPrefixLength(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}
	v, ok := bind.Value.(RawIP)
	if !ok {
		return nil, 0, errors.New("incorrect internal data for IP type")
	}
	return IntData(v.PrefixLength), 0, nil
}

func ipSuffix(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}
	v, ok := bind.Value.(RawIP)
	if !ok {
		return nil, 0, errors.New("incorrect internal data for IP type")
	}
	return StringData(v.Suffix()), 0, nil
}

func ipUnspecified(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}
	v, ok := bind.Value.(RawIP)
	if !ok {
		return nil, 0, errors.New("incorrect internal data for IP type")
	}
	return BoolData(v.IP.IsUnspecified()), 0, nil
}

func ipAddress(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.String, Error: bind.Error}, 0, nil
	}
	v, ok := bind.Value.(RawIP)
	if !ok {
		return nil, 0, errors.New("incorrect internal data for IP type")
	}
	return StringData(v.Address()), 0, nil
}

func ipCIDR(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.String, Error: bind.Error}, 0, nil
	}
	v, ok := bind.Value.(RawIP)
	if !ok {
		return nil, 0, errors.New("incorrect internal data for IP type")
	}
	return StringData(v.CIDR()), 0, nil
}

func any2ip(v any) (RawIP, bool) {
	switch t := v.(type) {
	case RawIP:
		return t, true
	case string:
		return ParseIP(t), true
	case int64:
		return ParseIntIP(t), true
	default:
		return RawIP{}, false
	}
}

func parseIpConstraints(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) ([]ipConstraint, uint64, error) {
	res := []ipConstraint{}
	for i := range chunk.Function.Args {
		argRef := chunk.Function.Args[i]

		arg, rref, err := e.resolveValue(argRef, ref)
		if err != nil || rref > 0 {
			return res, rref, err
		}

		ip, ok := any2ip(arg.Value)
		if !ok {
			return res, 0, errors.New("incorrect type for argument in `inRange` call (expected string, int, or IP)")
		}
		res = append(res, ipConstraint{
			address: ip,
		})
	}
	return res, 0, nil
}

func ipInRange(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}
	bindIP, ok := bind.Value.(RawIP)
	if !ok {
		return nil, 0, errors.New("incorrect internal data for IP type")
	}

	constraints, ref, err := parseIpConstraints(e, bind, chunk, ref)
	if err != nil || ref != 0 {
		return nil, ref, err
	}

	if len(constraints) == 1 {
		c := constraints[0]
		if c.operand == 0 {
			res := bindIP.CmpSubnet(c.address)
			return BoolData(res == 0 || res == -2 || res == 2), 0, nil
		}

		return nil, 0, errors.New("no support for other comparisons on IP address yet")
	}

	min := constraints[0]
	max := constraints[1]

	mincmp := min.address.Cmp(bindIP)
	if mincmp == 1 {
		return BoolFalse, 0, nil
	}
	maxcmp := bindIP.Cmp(max.address)
	if maxcmp == 1 {
		return BoolFalse, 0, nil
	}

	return BoolTrue, 0, nil
}

func ipInSubnet(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}
	bindIP, ok := bind.Value.(RawIP)
	if !ok {
		return nil, 0, errors.New("incorrect internal data for IP type")
	}

	constraints, ref, err := parseIpConstraints(e, bind, chunk, ref)
	if err != nil || ref != 0 {
		return nil, ref, err
	}

	if len(constraints) == 1 {
		c := constraints[0]
		if c.operand == 0 {
			res := bindIP.CmpSubnet(c.address)
			return BoolData(res == 0), 0, nil
		}

		return nil, 0, errors.New("no support for other comparisons on IP address yet")
	}

	min := constraints[0]
	max := constraints[1]

	mincmp := min.address.Cmp(bindIP)
	if mincmp != -1 {
		return BoolFalse, 0, nil
	}
	maxcmp := bindIP.Cmp(max.address)
	if maxcmp != -1 {
		return BoolFalse, 0, nil
	}

	return BoolTrue, 0, nil
}
