// Using license identifier: BUSL-1.1
// Using copyright holder: Mondoo, Inc.

package iox

import (
	"google.golang.org/protobuf/proto"
)

var maxMessageSize = 6 * (1 << 20)

func ChunkMessages[T proto.Message](sendFunc func([]T) error, onTooLarge func(T, int), items ...T) error {
	idx := 0
	for {
		buffer := make([]T, 0, len(items))

		if idx >= len(items) {
			break
		}
		size := 0
		for i := idx; i < len(items); i++ {
			msgSize := proto.Size(items[i])
			if msgSize > maxMessageSize {
				onTooLarge(items[i], msgSize)
				idx++
				continue
			}
			size += proto.Size(items[i])
			if size > maxMessageSize {
				break
			}
			buffer = append(buffer, items[i])
			idx++
		}
		if err := sendFunc(buffer); err != nil {
			return err
		}
	}

	return nil
}
