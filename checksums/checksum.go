package checksums

import (
	"encoding/base64"
	"encoding/binary"

	"github.com/segmentio/fasthash/fnv1a"
)

// Fast checksums
type Fast uint64

// New is the default starting checksum
const New = Fast(fnv1a.Init64)

// Add a string to a fast checksum
func (f Fast) Add(s string) Fast {
	return Fast(fnv1a.AddString64(uint64(f), s))
}

// Add an integer to a fast checksum
func (f Fast) AddUint(u uint64) Fast {
	return Fast(fnv1a.AddUint64(uint64(f), u))
}

// String returns a safe string representation of the checksum
func (f Fast) String() string {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(f))
	return base64.StdEncoding.EncodeToString(b)
}

// FastList returns the fast checksum as a string of a list of input strings
func FastList(strings ...string) string {
	checksum := New
	for i := range strings {
		checksum = checksum.Add(strings[i])
	}
	return checksum.String()
}
