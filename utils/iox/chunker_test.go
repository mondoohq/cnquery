// Using license identifier: BUSL-1.1
// Using copyright holder: Mondoo, Inc.

package iox

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/interop/grpc_testing"
)

func TestChunkerIgnoreTooLargeMessages(t *testing.T) {
	payloads := []*grpc_testing.Payload{
		{
			Body: bytes.Repeat([]byte{0x01}, maxMessageSize+1),
		},
		{
			Body: bytes.Repeat([]byte{0x02}, maxMessageSize/2),
		},
	}

	var chunks [][]*grpc_testing.Payload
	err := ChunkMessages(func(chunk []*grpc_testing.Payload) error {
		chunks = append(chunks, chunk)
		return nil
	}, func(*grpc_testing.Payload, int) {}, payloads...)
	require.NoError(t, err)
	require.Len(t, chunks, 1)
	require.Len(t, chunks[0], 1)
	require.Equal(t, payloads[1], chunks[0][0])
}

func TestChunker(t *testing.T) {
	maxMessageSize = 100
	payloads := []*grpc_testing.Payload{
		{
			Body: bytes.Repeat([]byte{0x01}, maxMessageSize-10),
		},
		{
			Body: bytes.Repeat([]byte{0x02}, maxMessageSize-10),
		},
		{
			Body: bytes.Repeat([]byte{0x03}, 10),
		},
		{
			Body: bytes.Repeat([]byte{0x04}, 10),
		},
	}

	var chunks [][]*grpc_testing.Payload
	err := ChunkMessages(func(chunk []*grpc_testing.Payload) error {
		chunks = append(chunks, chunk)
		return nil
	}, func(*grpc_testing.Payload, int) {}, payloads...)
	require.NoError(t, err)
	require.Len(t, chunks, 3)
	require.Len(t, chunks[0], 1)
	require.Equal(t, payloads[0], chunks[0][0])
	require.Len(t, chunks[1], 1)
	require.Equal(t, payloads[1], chunks[1][0])
	require.Len(t, chunks[2], 2)
	require.Equal(t, payloads[2], chunks[2][0])
}
