// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
)

// mockResource implements Resource for testing.
type mockResource struct {
	id   string
	name string
}

func (r *mockResource) MqlID() string   { return r.id }
func (r *mockResource) MqlName() string { return r.name }

// mockSerializableResource implements Resource + SerializableInternal for testing.
type mockSerializableResource struct {
	id       string
	name     string
	Internal map[string]string
}

func (r *mockSerializableResource) MqlID() string   { return r.id }
func (r *mockSerializableResource) MqlName() string { return r.name }

func (r *mockSerializableResource) MarshalInternal() ([]byte, error) {
	if len(r.Internal) == 0 {
		return nil, nil
	}
	return json.Marshal(r.Internal)
}

func (r *mockSerializableResource) UnmarshalInternal(data []byte) error {
	return json.Unmarshal(data, &r.Internal)
}

func newTestRuntime() *Runtime {
	return &Runtime{
		CreateResource: func(runtime *Runtime, name string, args map[string]*llx.RawData) (Resource, error) {
			id := ""
			if v, ok := args["__id"]; ok {
				id = v.Value.(string)
			}
			return &mockResource{id: id, name: name}, nil
		},
	}
}

// newSerializableTestRuntime creates resources that support SerializableInternal.
func newSerializableTestRuntime() *Runtime {
	return &Runtime{
		CreateResource: func(runtime *Runtime, name string, args map[string]*llx.RawData) (Resource, error) {
			id := ""
			if v, ok := args["__id"]; ok {
				id = v.Value.(string)
			}
			return &mockSerializableResource{id: id, name: name}, nil
		},
	}
}

func TestSqliteResources_BasicGetSet(t *testing.T) {
	runtime := newTestRuntime()
	sr, err := NewSqliteResources(100, runtime)
	require.NoError(t, err)
	defer sr.Close()

	res := &mockResource{id: "abc123", name: "test.resource"}

	// Get on empty returns false
	_, ok := sr.Get("test.resource\x00abc123")
	assert.False(t, ok)

	// Set and Get
	sr.Set("test.resource\x00abc123", res)
	got, ok := sr.Get("test.resource\x00abc123")
	assert.True(t, ok)
	assert.Equal(t, res, got)
}

func TestSqliteResources_LRUEvictionAndReconstruction(t *testing.T) {
	runtime := newTestRuntime()
	sr, err := NewSqliteResources(3, runtime)
	require.NoError(t, err)
	defer sr.Close()

	// Fill the LRU to capacity with SetWithArgs so data is in SQLite.
	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("test.resource\x00id-%d", i)
		res := &mockResource{id: fmt.Sprintf("id-%d", i), name: "test.resource"}
		sr.SetWithArgs(key, res, map[string]*llx.RawData{
			"__id": llx.StringData(fmt.Sprintf("id-%d", i)),
		})
	}

	// All 3 should be in LRU.
	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("test.resource\x00id-%d", i)
		_, ok := sr.Get(key)
		assert.True(t, ok, "expected key %s to be found", key)
	}

	// Add a 4th entry — evicts the LRU tail (id-0).
	sr.SetWithArgs("test.resource\x00id-3", &mockResource{id: "id-3", name: "test.resource"}, map[string]*llx.RawData{
		"__id": llx.StringData("id-3"),
	})

	// id-0 was evicted from LRU — Get should reconstruct it from SQLite.
	got, ok := sr.Get("test.resource\x00id-0")
	assert.True(t, ok, "evicted resource should be reconstructed from SQLite")
	if ok {
		assert.Equal(t, "id-0", got.MqlID())
		assert.Equal(t, "test.resource", got.MqlName())
	}

	// id-3 (newest) should be in LRU.
	got, ok = sr.Get("test.resource\x00id-3")
	assert.True(t, ok)
	assert.Equal(t, "id-3", got.MqlID())
}

func TestSqliteResources_SetWithArgsPersists(t *testing.T) {
	runtime := newTestRuntime()
	sr, err := NewSqliteResources(10, runtime)
	require.NoError(t, err)
	defer sr.Close()

	// SetWithArgs should not panic and should store in LRU
	args := map[string]*llx.RawData{
		"__id": llx.StringData("myid"),
		"name": llx.StringData("myname"),
	}
	sr.SetWithArgs("myres\x00myid", &mockResource{id: "myid", name: "myres"}, args)

	got, ok := sr.Get("myres\x00myid")
	assert.True(t, ok)
	assert.Equal(t, "myid", got.MqlID())
	assert.Equal(t, "myres", got.MqlName())
}

func TestSqliteResources_Close(t *testing.T) {
	runtime := newTestRuntime()
	sr, err := NewSqliteResources(10, runtime)
	require.NoError(t, err)

	dbPath := sr.dbPath
	assert.NotEmpty(t, dbPath)

	err = sr.Close()
	assert.NoError(t, err)
	assert.Empty(t, sr.dbPath)

	// Verify Close is idempotent
	err = sr.Close()
	assert.NoError(t, err)
}

func TestSqliteResources_ConcurrentAccess(t *testing.T) {
	runtime := newTestRuntime()
	sr, err := NewSqliteResources(100, runtime)
	require.NoError(t, err)
	defer sr.Close()

	var wg sync.WaitGroup
	n := 50

	// Concurrent writes
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("test.resource\x00id-%d", i)
			res := &mockResource{id: fmt.Sprintf("id-%d", i), name: "test.resource"}
			sr.SetWithArgs(key, res, map[string]*llx.RawData{
				"__id": llx.StringData(fmt.Sprintf("id-%d", i)),
			})
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("test.resource\x00id-%d", i)
			got, ok := sr.Get(key)
			assert.True(t, ok, "expected key %s to be found", key)
			if ok {
				assert.Equal(t, fmt.Sprintf("id-%d", i), got.MqlID())
			}
		}(i)
	}
	wg.Wait()
}

func TestSqliteResources_EmptyArgs(t *testing.T) {
	runtime := newTestRuntime()
	sr, err := NewSqliteResources(10, runtime)
	require.NoError(t, err)
	defer sr.Close()

	// SetWithArgs with nil args should not panic
	sr.SetWithArgs("test\x00id1", &mockResource{id: "id1", name: "test"}, nil)

	got, ok := sr.Get("test\x00id1")
	assert.True(t, ok)
	assert.Equal(t, "id1", got.MqlID())
}

func TestSqliteResources_SetWithArgsAfterClose(t *testing.T) {
	runtime := newTestRuntime()
	sr, err := NewSqliteResources(10, runtime)
	require.NoError(t, err)

	sr.Close()

	// Should not panic after Close
	sr.SetWithArgs("test\x00id1", &mockResource{id: "id1", name: "test"}, map[string]*llx.RawData{
		"__id": llx.StringData("id1"),
	})

	// LRU still works after Close (only SQLite is gone)
	got, ok := sr.Get("test\x00id1")
	assert.True(t, ok)
	assert.Equal(t, "id1", got.MqlID())
}

func TestSqliteResources_ImplementsResourcesWithArgs(t *testing.T) {
	runtime := newTestRuntime()
	sr, err := NewSqliteResources(10, runtime)
	require.NoError(t, err)
	defer sr.Close()

	// Should satisfy Resources[Resource]
	var _ Resources[Resource] = sr

	// Should satisfy ResourcesWithArgs
	var _ ResourcesWithArgs = sr
}

func TestSqliteResources_SerializableInternal(t *testing.T) {
	runtime := newSerializableTestRuntime()
	sr, err := NewSqliteResources(2, runtime)
	require.NoError(t, err)
	defer sr.Close()

	// Store a resource with internal state.
	res := &mockSerializableResource{
		id:   "s1",
		name: "test.serializable",
		Internal: map[string]string{
			"foo": "bar",
			"baz": "qux",
		},
	}
	sr.SetWithArgs("test.serializable\x00s1", res, map[string]*llx.RawData{
		"__id": llx.StringData("s1"),
	})

	// Evict by filling the LRU.
	sr.SetWithArgs("test.serializable\x00s2",
		&mockSerializableResource{id: "s2", name: "test.serializable"},
		map[string]*llx.RawData{"__id": llx.StringData("s2")})
	sr.SetWithArgs("test.serializable\x00s3",
		&mockSerializableResource{id: "s3", name: "test.serializable"},
		map[string]*llx.RawData{"__id": llx.StringData("s3")})

	// Reconstruct from SQLite — internal data should be restored.
	got, ok := sr.Get("test.serializable\x00s1")
	require.True(t, ok, "evicted resource should be reconstructed from SQLite")
	assert.Equal(t, "s1", got.MqlID())

	ser, ok := got.(SerializableInternal)
	require.True(t, ok, "reconstructed resource should implement SerializableInternal")
	// Verify internal state was restored via UnmarshalInternal.
	gotRes := got.(*mockSerializableResource)
	assert.Equal(t, "bar", gotRes.Internal["foo"])
	assert.Equal(t, "qux", gotRes.Internal["baz"])
	_ = ser
}

func TestSqliteResources_InFlightPreventsLoop(t *testing.T) {
	// Simulate CreateResource calling Get for the same key (deduplication).
	// The inFlight mechanism should return not-found for the re-entrant call.
	var sr *SqliteResources
	runtime := &Runtime{
		CreateResource: func(runtime *Runtime, name string, args map[string]*llx.RawData) (Resource, error) {
			id := ""
			if v, ok := args["__id"]; ok {
				id = v.Value.(string)
			}
			// Re-entrant call — should return false (in-flight).
			_, reOk := sr.Get(name + "\x00" + id)
			assert.False(t, reOk, "re-entrant Get during reconstruction should return false")
			return &mockResource{id: id, name: name}, nil
		},
	}

	var err error
	sr, err = NewSqliteResources(2, runtime)
	require.NoError(t, err)
	defer sr.Close()

	// Store and evict.
	sr.SetWithArgs("test\x00re-entrant", &mockResource{id: "re-entrant", name: "test"}, map[string]*llx.RawData{
		"__id": llx.StringData("re-entrant"),
	})
	sr.SetWithArgs("test\x00filler1", &mockResource{id: "filler1", name: "test"}, map[string]*llx.RawData{
		"__id": llx.StringData("filler1"),
	})
	sr.SetWithArgs("test\x00filler2", &mockResource{id: "filler2", name: "test"}, map[string]*llx.RawData{
		"__id": llx.StringData("filler2"),
	})

	// This triggers reconstruction which calls CreateResource which calls Get again.
	got, ok := sr.Get("test\x00re-entrant")
	assert.True(t, ok, "reconstruction should succeed despite re-entrant Get")
	assert.Equal(t, "re-entrant", got.MqlID())
}

func TestSqliteResources_SetWithoutArgs_NoReconstruction(t *testing.T) {
	runtime := newTestRuntime()
	sr, err := NewSqliteResources(2, runtime)
	require.NoError(t, err)
	defer sr.Close()

	// Set without args — nothing persisted to SQLite.
	sr.Set("test\x00plain", &mockResource{id: "plain", name: "test"})

	// Evict by filling LRU.
	sr.Set("test\x00a", &mockResource{id: "a", name: "test"})
	sr.Set("test\x00b", &mockResource{id: "b", name: "test"})

	// Should not be found — no SQLite data to reconstruct from.
	_, ok := sr.Get("test\x00plain")
	assert.False(t, ok, "resource stored with Set (no args) should not be reconstructable")
}

func TestSqliteResources_FieldCache_BasicRoundTrip(t *testing.T) {
	runtime := newTestRuntime()
	sr, err := NewSqliteResources(10, runtime)
	require.NoError(t, err)
	defer sr.Close()

	cacheKey := "test.resource\x00abc123"

	// SetField with data
	sr.SetField(cacheKey, "name", &DataRes{
		Data: llx.StringPrimitive("hello"),
	})

	// GetField should return the same DataRes
	got := sr.GetField(cacheKey, "name")
	require.NotNil(t, got)
	assert.Equal(t, "hello", string(got.Data.Value))
	assert.Empty(t, got.Error)

	// SetField with error
	sr.SetField(cacheKey, "broken", &DataRes{
		Error: "something went wrong",
	})

	got = sr.GetField(cacheKey, "broken")
	require.NotNil(t, got)
	assert.Nil(t, got.Data)
	assert.Equal(t, "something went wrong", got.Error)
}

func TestSqliteResources_FieldCache_Miss(t *testing.T) {
	runtime := newTestRuntime()
	sr, err := NewSqliteResources(10, runtime)
	require.NoError(t, err)
	defer sr.Close()

	// GetField on non-existent key returns nil
	got := sr.GetField("nonexistent\x00key", "field")
	assert.Nil(t, got)
}

func TestSqliteResources_FieldCache_AfterClose(t *testing.T) {
	runtime := newTestRuntime()
	sr, err := NewSqliteResources(10, runtime)
	require.NoError(t, err)

	sr.Close()

	// SetField after Close should be a no-op (no panic)
	sr.SetField("test\x00id", "field", &DataRes{
		Data: llx.StringPrimitive("value"),
	})

	// GetField after Close should return nil (no panic)
	got := sr.GetField("test\x00id", "field")
	assert.Nil(t, got)
}

func TestSqliteResources_ImplementsResourcesWithFieldCache(t *testing.T) {
	runtime := newTestRuntime()
	sr, err := NewSqliteResources(10, runtime)
	require.NoError(t, err)
	defer sr.Close()

	var _ ResourcesWithFieldCache = sr
}

func TestMarshalUnmarshalArgs(t *testing.T) {
	args := map[string]*llx.RawData{
		"__id":  llx.StringData("test-id"),
		"name":  llx.StringData("test-name"),
		"count": llx.IntData(42),
		"flag":  llx.BoolData(true),
	}

	bytes, err := marshalArgs(args)
	require.NoError(t, err)
	require.NotNil(t, bytes)

	pargs, err := unmarshalArgs(bytes)
	require.NoError(t, err)

	// Verify the primitives round-trip correctly
	assert.Equal(t, "test-id", string(pargs["__id"].Value))
	assert.Equal(t, "test-name", string(pargs["name"].Value))
}

func TestMarshalUnmarshalArgs_Empty(t *testing.T) {
	bytes, err := marshalArgs(nil)
	require.NoError(t, err)
	assert.Nil(t, bytes)

	pargs, err := unmarshalArgs(nil)
	require.NoError(t, err)
	assert.Empty(t, pargs)
}
