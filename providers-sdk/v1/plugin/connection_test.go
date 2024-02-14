// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	inventory "go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
)

func TestConnection_ID(t *testing.T) {
	c := NewConnection(1, &inventory.Asset{})
	require.NotNil(t, c)
	assert.Equal(t, uint32(1), c.ID())
}

func TestConnection_ParentID_Nil(t *testing.T) {
	c := NewConnection(1, &inventory.Asset{})
	require.NotNil(t, c)
	assert.Equal(t, 0, c.ParentID())
}

func TestConnection_ParentID(t *testing.T) {
	c := NewConnection(1, &inventory.Asset{
		Connections: []*inventory.Config{
			{
				ParentConnectionId: 2,
			},
		},
	})
	require.NotNil(t, c)
	assert.Equal(t, uint32(2), c.ParentID())
}

func TestConnection_ParentID_0(t *testing.T) {
	c := NewConnection(1, &inventory.Asset{
		Connections: []*inventory.Config{
			{
				ParentConnectionId: 0,
			},
		},
	})
	require.NotNil(t, c)
	assert.Nil(t, c.ParentID())
}
