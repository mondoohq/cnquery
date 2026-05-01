// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/stretchr/testify/assert"
)

func TestPubsubSchemaTypeString(t *testing.T) {
	t.Run("protocol buffer", func(t *testing.T) {
		assert.Equal(t, "PROTOCOL_BUFFER", pubsubSchemaTypeString(pubsubpb.Schema_PROTOCOL_BUFFER))
	})

	t.Run("avro", func(t *testing.T) {
		assert.Equal(t, "AVRO", pubsubSchemaTypeString(pubsubpb.Schema_AVRO))
	})

	t.Run("unspecified", func(t *testing.T) {
		assert.Equal(t, "TYPE_UNSPECIFIED", pubsubSchemaTypeString(pubsubpb.Schema_TYPE_UNSPECIFIED))
	})

	t.Run("unknown value", func(t *testing.T) {
		assert.Equal(t, "TYPE_UNSPECIFIED", pubsubSchemaTypeString(pubsubpb.Schema_Type(99)))
	})
}

func TestPubsubSchemaEncodingString(t *testing.T) {
	t.Run("JSON encoding", func(t *testing.T) {
		assert.Equal(t, "JSON", pubsubSchemaEncodingString(pubsubpb.Encoding_JSON))
	})

	t.Run("binary encoding", func(t *testing.T) {
		assert.Equal(t, "BINARY", pubsubSchemaEncodingString(pubsubpb.Encoding_BINARY))
	})

	t.Run("unspecified encoding", func(t *testing.T) {
		assert.Equal(t, "ENCODING_UNSPECIFIED", pubsubSchemaEncodingString(pubsubpb.Encoding(99)))
	})
}

func TestBuildTopicSchemaSettings(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result, err := buildTopicSchemaSettings(nil, "parent", nil)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}
