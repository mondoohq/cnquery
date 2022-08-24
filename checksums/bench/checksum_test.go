package bench

import (
	"encoding/base64"
	"encoding/binary"
	"testing"

	"github.com/segmentio/fasthash/fnv1a"
	"go.mondoo.com/cnquery/checksums"
)

var result string

func BenchmarkChecksum_fnv1a(b *testing.B) {
	var res string
	for n := 0; n < b.N; n++ {
		checksum := fnv1a.Init64
		for i := 0; i < 1000; i++ {
			checksum = fnv1a.AddString64(checksum, "hello")
		}

		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(checksum))
		res = base64.StdEncoding.EncodeToString(b)
	}
	result = res
}

func BenchmarkChecksum_fast(b *testing.B) {
	var res string
	for n := 0; n < b.N; n++ {
		checksum := checksums.New
		for i := 0; i < 1000; i++ {
			checksum = checksum.Add("hello")
		}

		res = checksum.String()
	}
	result = res
}
