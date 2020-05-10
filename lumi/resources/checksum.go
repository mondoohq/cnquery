package resources

import (
	"encoding/base64"
	"encoding/binary"

	"github.com/segmentio/fasthash/fnv1a"
)

func checksum2string(checksum uint64) string {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, checksum)
	return base64.StdEncoding.EncodeToString(b)
}
func checksumStrings(strings ...string) string {
	checksum := fnv1a.Init64
	for i := range strings {
		checksum = fnv1a.AddString64(checksum, strings[i])
	}
	return checksum2string(checksum)
}
