// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"cloud.google.com/go/memorystore/apiv1/memorystorepb"
	"github.com/stretchr/testify/assert"
)

func TestMemorystorePscAutoConnectionPort(t *testing.T) {
	t.Run("nil connection", func(t *testing.T) {
		assert.Equal(t, int64(0), memorystorePscAutoConnectionPort(nil))
	})

	t.Run("ports oneof unset", func(t *testing.T) {
		c := &memorystorepb.PscAutoConnection{PscConnectionId: "pc-1"}
		assert.Equal(t, int64(0), memorystorePscAutoConnectionPort(c))
	})

	t.Run("port set via oneof", func(t *testing.T) {
		c := &memorystorepb.PscAutoConnection{
			PscConnectionId: "pc-1",
			Ports:           &memorystorepb.PscAutoConnection_Port{Port: 6379},
		}
		assert.Equal(t, int64(6379), memorystorePscAutoConnectionPort(c))
	})
}
