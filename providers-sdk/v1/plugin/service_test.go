// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/utils/syncx"
)

type TestConnection struct {
	id       uint32
	parentId uint32
}

func newTestConnection(id uint32) *TestConnection {
	return &TestConnection{id: id}
}

func (c *TestConnection) ID() uint32 {
	return c.id
}

func (c *TestConnection) ParentID() uint32 {
	return c.parentId
}

type TestConnectionWithClose struct {
	*TestConnection
	closed bool
}

func newTestConnectionWithClose(id uint32) *TestConnectionWithClose {
	return &TestConnectionWithClose{TestConnection: newTestConnection(id)}
}

func (c *TestConnectionWithClose) Close() {
	c.closed = true
}

type TestResource struct{}

func (r *TestResource) MqlID() string {
	return "test.resource"
}

func (r *TestResource) MqlName() string {
	return "Test Resource"
}

func TestAddRuntime(t *testing.T) {
	s := NewService()
	wg := sync.WaitGroup{}
	wg.Add(4)
	addRuntimes := func(j int) {
		defer wg.Done()
		for i := 1; i < 51; i++ {
			idStr := fmt.Sprintf("%d%d", i, j)
			id, err := strconv.Atoi(idStr)
			require.NoError(t, err)
			_, err = s.AddRuntime(&inventory.Config{Id: uint32(id)}, func(connId uint32) (*Runtime, error) {
				return &Runtime{}, nil
			})
			require.NoError(t, err)
		}
	}

	// Add runtimes concurrently
	for i := 1; i < 5; i++ {
		go addRuntimes(i)
	}

	// Wait until all runtimes are added
	wg.Wait()

	// Vertify that all runtimes are added and the last connectiod ID is correct
	assert.Len(t, s.runtimes, 200)
	assert.Equal(t, s.lastConnectionID, uint32(0))
}

func TestAddRuntime_Existing(t *testing.T) {
	s := NewService()

	inv := &inventory.Config{Id: 1}
	createRuntime := func(connId uint32) (*Runtime, error) {
		resMap := &syncx.Map[Resource]{}
		resMap.Set("test.resource", &TestResource{})

		return &Runtime{
			Resources:  resMap,
			Connection: newTestConnection(connId),
		}, nil
	}
	runtime1, err := s.AddRuntime(inv, createRuntime)
	require.NoError(t, err)

	runtime2, err := s.AddRuntime(inv, createRuntime)
	require.NoError(t, err)
	assert.Equal(t, runtime1, runtime2)
}

func TestDeprecatedAddRuntime(t *testing.T) {
	s := NewService()
	wg := sync.WaitGroup{}
	wg.Add(4)
	addRuntimes := func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_, err := s.AddRuntime(&inventory.Config{}, func(connId uint32) (*Runtime, error) {
				return &Runtime{}, nil
			})
			require.NoError(t, err)
		}
	}

	// Add runtimes concurrently
	for i := 0; i < 4; i++ {
		go addRuntimes()
	}

	// Wait until all runtimes are added
	wg.Wait()

	// Vertify that all runtimes are added and the last connectiod ID is correct
	assert.Len(t, s.runtimes, 200)
	assert.Equal(t, s.lastConnectionID, uint32(200))
}

func TestAddRuntime_ParentNotExist(t *testing.T) {
	s := NewService()
	parentId := uint32(10)
	_, err := s.AddRuntime(&inventory.Config{Id: 1}, func(connId uint32) (*Runtime, error) {
		c := newTestConnection(connId)
		c.parentId = parentId
		return &Runtime{
			Connection: c,
		}, nil
	})
	require.Error(t, err)
	assert.Equal(t, "parent connection 10 not found", err.Error())
}

func TestDeprecatedAddRuntime_ParentNotExist(t *testing.T) {
	s := NewService()
	parentId := uint32(10)
	_, err := s.AddRuntime(&inventory.Config{}, func(connId uint32) (*Runtime, error) {
		c := newTestConnection(connId)
		c.parentId = parentId
		return &Runtime{
			Connection: c,
		}, nil
	})
	require.Error(t, err)
	assert.Equal(t, "parent connection 10 not found", err.Error())
}

func TestAddRuntime_Parent(t *testing.T) {
	s := NewService()

	parent, err := s.AddRuntime(&inventory.Config{Id: 1}, func(connId uint32) (*Runtime, error) {
		resMap := &syncx.Map[Resource]{}
		resMap.Set("test.resource", &TestResource{})

		return &Runtime{
			Resources:  resMap,
			Connection: newTestConnection(connId),
		}, nil
	})
	require.NoError(t, err)

	parentId := parent.Connection.ID()
	child, err := s.AddRuntime(&inventory.Config{Id: 2}, func(connId uint32) (*Runtime, error) {
		c := newTestConnection(connId)
		c.parentId = parentId
		return &Runtime{
			Connection: c,
		}, nil
	})
	require.NoError(t, err)

	// Check that the resources for the parent and the child are the same
	assert.Equal(t, parent.Resources, child.Resources)

	// Add another resource and check that it appears in the child runtime
	parent.Resources.Set("test.resource2", &TestResource{})
	assert.Equal(t, parent.Resources, child.Resources)
}

func TestDeprecatedAddRuntime_Parent(t *testing.T) {
	s := NewService()

	parent, err := s.AddRuntime(&inventory.Config{}, func(connId uint32) (*Runtime, error) {
		resMap := &syncx.Map[Resource]{}
		resMap.Set("test.resource", &TestResource{})

		return &Runtime{
			Resources:  resMap,
			Connection: newTestConnection(connId),
		}, nil
	})
	require.NoError(t, err)

	parentId := parent.Connection.ID()
	child, err := s.AddRuntime(&inventory.Config{}, func(connId uint32) (*Runtime, error) {
		c := newTestConnection(connId)
		c.parentId = parentId
		return &Runtime{
			Connection: c,
		}, nil
	})
	require.NoError(t, err)

	// Check that the resources for the parent and the child are the same
	assert.Equal(t, parent.Resources, child.Resources)

	// Add another resource and check that it appears in the child runtime
	parent.Resources.Set("test.resource2", &TestResource{})
	assert.Equal(t, parent.Resources, child.Resources)
}

func TestGetRuntime(t *testing.T) {
	s := NewService()

	runtime, err := s.AddRuntime(&inventory.Config{}, func(connId uint32) (*Runtime, error) {
		return &Runtime{
			Connection: newTestConnection(connId),
		}, nil
	})
	require.NoError(t, err)

	// Add some more runtimes
	for i := 0; i < 5; i++ {
		_, err := s.AddRuntime(&inventory.Config{}, func(connId uint32) (*Runtime, error) {
			return &Runtime{
				Connection: newTestConnection(connId),
			}, nil
		})
		require.NoError(t, err)
	}

	// Retrieve the first runtime
	retrievedRuntime, err := s.GetRuntime(runtime.Connection.ID())
	require.NoError(t, err)
	assert.Equal(t, runtime, retrievedRuntime)
}

func TestGetRuntime_DoesNotExist(t *testing.T) {
	s := NewService()

	_, err := s.AddRuntime(&inventory.Config{}, func(connId uint32) (*Runtime, error) {
		return &Runtime{
			Connection: newTestConnection(connId),
		}, nil
	})
	require.NoError(t, err)

	_, err = s.GetRuntime(10)
	assert.Error(t, err)
	assert.Equal(t, "connection 10 not found", err.Error())
}

func TestDisconnect(t *testing.T) {
	s := NewService()

	runtime, err := s.AddRuntime(&inventory.Config{}, func(connId uint32) (*Runtime, error) {
		return &Runtime{
			Connection: newTestConnection(connId),
		}, nil
	})
	require.NoError(t, err)

	assert.Len(t, s.runtimes, 1)

	_, err = s.Disconnect(&DisconnectReq{Connection: runtime.Connection.ID()})
	require.NoError(t, err)
	assert.Empty(t, s.runtimes)
}

func TestDisconnect_Closer(t *testing.T) {
	s := NewService()

	runtime, err := s.AddRuntime(&inventory.Config{}, func(connId uint32) (*Runtime, error) {
		return &Runtime{
			Connection: newTestConnectionWithClose(connId),
		}, nil
	})
	require.NoError(t, err)

	assert.False(t, runtime.Connection.(*TestConnectionWithClose).closed)
	assert.Len(t, s.runtimes, 1)

	_, err = s.Disconnect(&DisconnectReq{Connection: runtime.Connection.ID()})
	require.NoError(t, err)
	assert.Empty(t, s.runtimes)

	assert.True(t, runtime.Connection.(*TestConnectionWithClose).closed)
}

func TestShutdown(t *testing.T) {
	s := NewService()

	// Add some more runtimes
	for i := 0; i < 50; i++ {
		_, err := s.AddRuntime(&inventory.Config{}, func(connId uint32) (*Runtime, error) {
			return &Runtime{
				Connection: newTestConnection(connId),
			}, nil
		})
		require.NoError(t, err)
	}

	// Shutdown and verify all runtimes are gone
	_, err := s.Shutdown(&ShutdownReq{})
	require.NoError(t, err)
	assert.Empty(t, s.runtimes)
}

func TestShutdown_Closer(t *testing.T) {
	s := NewService()

	// Add some more runtimes
	runtimes := []*Runtime{}
	for i := 0; i < 50; i++ {
		runtime, err := s.AddRuntime(&inventory.Config{}, func(connId uint32) (*Runtime, error) {
			return &Runtime{
				Connection: newTestConnectionWithClose(connId),
			}, nil
		})
		require.NoError(t, err)
		runtimes = append(runtimes, runtime)
	}

	// Shutdown and verify all runtimes are gone
	_, err := s.Shutdown(&ShutdownReq{})
	require.NoError(t, err)
	assert.Empty(t, s.runtimes)

	// Verify that all runtimes are closed
	for _, runtime := range runtimes {
		assert.True(t, runtime.Connection.(*TestConnectionWithClose).closed)
	}
}
