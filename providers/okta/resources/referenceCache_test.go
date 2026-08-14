// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/utils/syncx"
)

func testOktaUser(id string) *mqlOktaUser {
	return &mqlOktaUser{
		Id: plugin.TValue[string]{Data: id, State: plugin.StateIsSet},
	}
}

func testOktaGroup(id string) *mqlOktaGroup {
	return &mqlOktaGroup{
		Id: plugin.TValue[string]{Data: id, State: plugin.StateIsSet},
	}
}

// A readable list is the only state a reference may be answered from. Every
// other state has to read as a miss, because a miss falls back to the direct
// lookup while an empty answer would report the object as absent.
func TestReadableOktaList(t *testing.T) {
	entries := []any{testOktaUser("00u1")}

	t.Run("fetched list", func(t *testing.T) {
		list := &plugin.TValue[[]any]{Data: entries, State: plugin.StateIsSet}
		assert.Equal(t, entries, readableOktaList(list))
	})

	t.Run("errored list", func(t *testing.T) {
		list := &plugin.TValue[[]any]{
			Data:  entries,
			State: plugin.StateIsSet | plugin.StateIsNull,
			Error: errors.New("api is having a bad day"),
		}
		assert.Nil(t, readableOktaList(list))
	})

	t.Run("null list", func(t *testing.T) {
		// What an unlicensed feature leaves behind: read, but absent.
		list := &plugin.TValue[[]any]{State: plugin.StateIsSet | plugin.StateIsNull}
		assert.Nil(t, readableOktaList(list))
	})

	t.Run("unfetched list", func(t *testing.T) {
		assert.Nil(t, readableOktaList(&plugin.TValue[[]any]{}))
	})

	t.Run("empty list", func(t *testing.T) {
		list := &plugin.TValue[[]any]{Data: []any{}, State: plugin.StateIsSet}
		assert.Empty(t, readableOktaList(list))
	})
}

func TestFindCachedOktaResource(t *testing.T) {
	wanted := testOktaUser("00u2")
	list := []any{testOktaUser("00u1"), wanted, testOktaUser("00u3")}

	t.Run("hit returns the listed resource", func(t *testing.T) {
		found, ok := findCachedOktaResource(list, oktaUserID, "00u2")
		require.True(t, ok)
		assert.Same(t, wanted, found)
	})

	t.Run("id absent from the list is a miss", func(t *testing.T) {
		// A deprovisioned user is not in the user list but is still served by
		// id, so this has to fall through rather than answer null.
		found, ok := findCachedOktaResource(list, oktaUserID, "00uDeprovisioned")
		assert.False(t, ok)
		assert.Nil(t, found)
	})

	t.Run("empty id is a miss", func(t *testing.T) {
		found, ok := findCachedOktaResource([]any{testOktaUser("")}, oktaUserID, "")
		assert.False(t, ok)
		assert.Nil(t, found)
	})

	t.Run("unreadable list is a miss", func(t *testing.T) {
		found, ok := findCachedOktaResource(readableOktaList(&plugin.TValue[[]any]{
			State: plugin.StateIsSet | plugin.StateIsNull,
		}), oktaUserID, "00u2")
		assert.False(t, ok)
		assert.Nil(t, found)
	})

	t.Run("entry of another type is skipped", func(t *testing.T) {
		mixed := []any{testOktaGroup("00u2"), "not a resource", nil, wanted}
		found, ok := findCachedOktaResource(mixed, oktaUserID, "00u2")
		require.True(t, ok)
		assert.Same(t, wanted, found)
	})
}

func TestIndexCachedOktaResources(t *testing.T) {
	first := testOktaUser("00u1")
	second := testOktaUser("00u2")
	list := []any{first, second, testOktaUser("00u3")}

	t.Run("keys only the wanted ids", func(t *testing.T) {
		index := indexCachedOktaResources(list, oktaUserID, []string{"00u1", "00u2"})
		require.Len(t, index, 2)
		assert.Same(t, first, index["00u1"])
		assert.Same(t, second, index["00u2"])
	})

	t.Run("ids absent from the list are absent from the index", func(t *testing.T) {
		index := indexCachedOktaResources(list, oktaUserID, []string{"00u1", "00uGone"})
		require.Len(t, index, 1)
		assert.Same(t, first, index["00u1"])

		_, ok := index["00uGone"]
		assert.False(t, ok)
	})

	t.Run("empty ids are never keyed", func(t *testing.T) {
		index := indexCachedOktaResources([]any{testOktaUser("")}, oktaUserID, []string{""})
		assert.Empty(t, index)

		_, ok := index[""]
		assert.False(t, ok)
	})

	t.Run("unreadable list indexes nothing", func(t *testing.T) {
		unreadable := readableOktaList(&plugin.TValue[[]any]{
			Error: errors.New("api is having a bad day"),
			State: plugin.StateIsSet | plugin.StateIsNull,
		})
		index := indexCachedOktaResources(unreadable, oktaUserID, []string{"00u1"})
		assert.Empty(t, index)

		_, ok := index["00u1"]
		assert.False(t, ok)
	})

	t.Run("no ids wanted", func(t *testing.T) {
		assert.Empty(t, indexCachedOktaResources(list, oktaUserID, nil))
	})

	t.Run("entries of another type are skipped", func(t *testing.T) {
		mixed := []any{testOktaGroup("00u1"), "not a resource", nil, second}
		index := indexCachedOktaResources(mixed, oktaUserID, []string{"00u1", "00u2"})
		require.Len(t, index, 1)
		assert.Same(t, second, index["00u2"])
	})

	t.Run("group collection keys by group id", func(t *testing.T) {
		group := testOktaGroup("00g1")
		index := indexCachedOktaResources([]any{group}, oktaGroupID, []string{"00g1"})
		require.Len(t, index, 1)
		assert.Same(t, group, index["00g1"])
	})
}

// Reference accessors are called from blocks the executor runs in parallel, so
// the collection they share is read concurrently. Run under -race, this covers
// the whole shared path: reaching the root through the runtime's resource
// cache, reading its collection, and scanning it.
func TestCachedOktaCollectionConcurrentReads(t *testing.T) {
	runtime := &plugin.Runtime{Resources: &syncx.Map[plugin.Resource]{}}

	root, err := getOkta(runtime)
	require.NoError(t, err)

	wanted := testOktaUser("00u2")
	root.Users = plugin.TValue[[]any]{
		Data:  []any{testOktaUser("00u1"), wanted, testOktaUser("00u3")},
		State: plugin.StateIsSet,
	}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			users := cachedOktaUsers(runtime)
			found, ok := findCachedOktaResource(users, oktaUserID, "00u2")
			assert.True(t, ok)
			assert.Same(t, wanted, found)

			index := indexCachedOktaResources(users, oktaUserID, []string{"00u2", "00uGone"})
			assert.Same(t, wanted, index["00u2"])
			assert.Len(t, index, 1)
		}()
	}
	wg.Wait()
}

// The collection a reference is answered from is listed once no matter how many
// accessors reach it at the same time. Without the lock the field memo carries
// no synchronization of its own, so this is where a second list request, and a
// race on the field holding the first one, would show up.
func TestCachedOktaCollectionListsOnce(t *testing.T) {
	runtime := &plugin.Runtime{Resources: &syncx.Map[plugin.Resource]{}}

	entries := []any{testOktaUser("00u1"), testOktaUser("00u2")}
	var lists atomic.Int64
	read := func(root *mqlOkta) *plugin.TValue[[]any] {
		return plugin.GetOrCompute(&root.Users, func() ([]any, error) {
			lists.Add(1)
			return entries, nil
		})
	}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.Equal(t, entries, cachedOktaCollection(runtime, read))
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(1), lists.Load())
}

// A collection that could not be read is not retried per reference: the field
// memo holds the failure, so the second accessor sees the same miss without
// issuing a second request.
func TestCachedOktaCollectionRemembersFailure(t *testing.T) {
	runtime := &plugin.Runtime{Resources: &syncx.Map[plugin.Resource]{}}

	var lists atomic.Int64
	read := func(root *mqlOkta) *plugin.TValue[[]any] {
		return plugin.GetOrCompute(&root.Users, func() ([]any, error) {
			lists.Add(1)
			return nil, errors.New("api is having a bad day")
		})
	}

	for i := 0; i < 4; i++ {
		assert.Nil(t, cachedOktaCollection(runtime, read))
	}
	assert.Equal(t, int64(1), lists.Load())
}
