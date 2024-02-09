// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestConnection struct {
	id uint32
}

func newTestConnection(id uint32) *TestConnection {
	return &TestConnection{id: id}
}

func (c *TestConnection) ID() uint32 {
	return c.id
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

func TestAddRuntime(t *testing.T) {
	s := NewService()
	wg := sync.WaitGroup{}
	wg.Add(4)
	addRuntimes := func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_, err := s.AddRuntime(func(connId uint32) (*Runtime, error) {
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

func TestGetRuntime(t *testing.T) {
	s := NewService()

	runtime, err := s.AddRuntime(func(connId uint32) (*Runtime, error) {
		return &Runtime{
			Connection: newTestConnection(connId),
		}, nil
	})
	require.NoError(t, err)

	// Add some more runtimes
	for i := 0; i < 5; i++ {
		_, err := s.AddRuntime(func(connId uint32) (*Runtime, error) {
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

	_, err := s.AddRuntime(func(connId uint32) (*Runtime, error) {
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

	runtime, err := s.AddRuntime(func(connId uint32) (*Runtime, error) {
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

	runtime, err := s.AddRuntime(func(connId uint32) (*Runtime, error) {
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
		_, err := s.AddRuntime(func(connId uint32) (*Runtime, error) {
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
		runtime, err := s.AddRuntime(func(connId uint32) (*Runtime, error) {
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
