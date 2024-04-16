// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin

import inventory "go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"

type Connection interface {
	ID() uint32

	// ParentID returns the ID of the parent connection. If this returns >0,
	// the connection with that ID will be used to store and get data.
	ParentID() uint32
}

type connection struct {
	id       uint32
	parentId uint32
}

func NewConnection(id uint32, asset *inventory.Asset) Connection {
	conn := &connection{
		id: id,
	}
	if len(asset.Connections) > 0 && asset.Connections[0].ParentConnectionId > 0 {
		conn.parentId = asset.Connections[0].ParentConnectionId
	}
	return conn
}

func (c *connection) ID() uint32 {
	return c.id
}

func (c *connection) ParentID() uint32 {
	return c.parentId
}
