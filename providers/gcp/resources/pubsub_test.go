// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"cloud.google.com/go/pubsub"
	"github.com/stretchr/testify/assert"
)

func TestPubsubSchemaTypeString(t *testing.T) {
	t.Run("protocol buffer", func(t *testing.T) {
		assert.Equal(t, "PROTOCOL_BUFFER", pubsubSchemaTypeString(pubsub.SchemaProtocolBuffer))
	})

	t.Run("avro", func(t *testing.T) {
		assert.Equal(t, "AVRO", pubsubSchemaTypeString(pubsub.SchemaAvro))
	})

	t.Run("unspecified", func(t *testing.T) {
		assert.Equal(t, "TYPE_UNSPECIFIED", pubsubSchemaTypeString(pubsub.SchemaTypeUnspecified))
	})

	t.Run("unknown value", func(t *testing.T) {
		assert.Equal(t, "TYPE_UNSPECIFIED", pubsubSchemaTypeString(pubsub.SchemaType(99)))
	})
}
