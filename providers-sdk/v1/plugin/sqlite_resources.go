// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin

import (
	"container/list"
	"database/sql"
	"os"
	"sync"

	_ "github.com/glebarez/go-sqlite"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
)

// SqliteResources is an optional Resources[Resource] implementation backed by
// an LRU in-memory cache and a SQLite database for overflow. When the in-memory
// cache reaches its capacity, evicted entries fall back to SQLite storage, from
// which they can be reconstructed on the next Get.
type SqliteResources struct {
	mu       sync.RWMutex
	lru      *lruCache
	db       *sql.DB
	dbPath   string
	runtime  *Runtime
	inFlight map[string]struct{} // keys currently being reconstructed (prevents infinite recursion)
}

// NewSqliteResources creates a new SqliteResources with the given LRU capacity.
// The SQLite database is created as a temp file in WAL mode for concurrent access.
func NewSqliteResources(capacity int, runtime *Runtime) (*SqliteResources, error) {
	f, err := os.CreateTemp("", "mql-resource-cache-*.db")
	if err != nil {
		return nil, err
	}
	dbPath := f.Name()
	f.Close()

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL")
	if err != nil {
		os.Remove(dbPath)
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS resources (
		cache_key TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		res_id TEXT NOT NULL,
		args BLOB,
		internal_data BLOB
	)`)
	if err != nil {
		db.Close()
		os.Remove(dbPath)
		return nil, err
	}

	log.Warn().Str("path", dbPath).Int("capacity", capacity).Msg("sqlite resource cache created")

	sr := &SqliteResources{
		lru:      newLRUCache(capacity),
		db:       db,
		dbPath:   dbPath,
		runtime:  runtime,
		inFlight: make(map[string]struct{}),
	}
	// Persist internal state (e.g. k8s API objects) to SQLite when a resource
	// is evicted from the LRU. This must happen on eviction rather than in
	// SetWithArgs because internal state is set imperatively *after*
	// CreateResource returns (e.g. r.(*mqlK8sNamespace).obj = &ns).
	sr.lru.onEvict = sr.onLRUEvict
	return sr, nil
}

// dbRow holds data read from SQLite for reconstruction outside the lock.
type dbRow struct {
	name         string
	resID        string
	args         []byte
	internalData []byte
}

// Get retrieves a resource by key. It first checks the LRU cache, then falls
// back to SQLite. On a SQLite hit, the resource is reconstructed via
// runtime.CreateResource and promoted back into the LRU.
func (s *SqliteResources) Get(key string) (Resource, bool) {
	// Fast path: check LRU under read lock.
	s.mu.RLock()
	res, ok := s.lru.get(key)
	s.mu.RUnlock()
	if ok {
		return res, true
	}

	// Slow path: try to reconstruct from SQLite.
	row := s.readFromDB(key)
	if row == nil {
		return nil, false
	}

	// Reconstruct the resource outside any lock. CreateResource will call
	// back into Get/Set on this same SqliteResources, which is safe because
	// we marked the key as inFlight (so readFromDB will skip it).
	resource, err := s.reconstruct(key, row)
	if err != nil {
		log.Warn().Err(err).Str("key", key).Msg("sqlite resource cache: failed to reconstruct resource")
		s.clearInFlight(key)
		return nil, false
	}

	// Promote the reconstructed resource into the LRU.
	s.mu.Lock()
	s.lru.set(key, resource)
	delete(s.inFlight, key)
	s.mu.Unlock()

	return resource, true
}

// readFromDB reads a resource row from SQLite under the mutex. Returns nil if
// the key is not found, the DB is closed, or the key is currently in-flight
// (being reconstructed — prevents infinite recursion).
func (s *SqliteResources) readFromDB(key string) *dbRow {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return nil
	}
	if _, ok := s.inFlight[key]; ok {
		return nil
	}

	var row dbRow
	err := s.db.QueryRow(
		`SELECT name, res_id, args, internal_data FROM resources WHERE cache_key = ?`, key,
	).Scan(&row.name, &row.resID, &row.args, &row.internalData)
	if err != nil {
		return nil
	}

	// Mark as in-flight before releasing the lock so that re-entrant Get
	// calls for the same key (triggered by CreateResource) return not-found
	// instead of looping forever.
	s.inFlight[key] = struct{}{}
	return &row
}

// clearInFlight removes a key from the in-flight set.
func (s *SqliteResources) clearInFlight(key string) {
	s.mu.Lock()
	delete(s.inFlight, key)
	s.mu.Unlock()
}

// reconstruct creates a resource from a SQLite row using runtime.CreateResource,
// then restores internal state via SerializableInternal if available.
func (s *SqliteResources) reconstruct(key string, row *dbRow) (Resource, error) {
	pargs, err := unmarshalArgs(row.args)
	if err != nil {
		return nil, err
	}

	// Ensure __id is always present so CreateResource can identify the resource.
	if _, ok := pargs["__id"]; !ok {
		pargs["__id"] = llx.StringPrimitive(row.resID)
	}

	rawArgs := PrimitiveArgsToRawDataArgs(pargs, s.runtime)
	resource, err := s.runtime.CreateResource(s.runtime, row.name, rawArgs)
	if err != nil {
		return nil, err
	}

	// Restore internal state if the resource supports it and we have data.
	if len(row.internalData) > 0 {
		if si, ok := resource.(SerializableInternal); ok {
			if err := si.UnmarshalInternal(row.internalData); err != nil {
				log.Warn().Err(err).Str("key", key).Msg("sqlite resource cache: failed to unmarshal internal data")
			}
		}
	}

	return resource, nil
}

// Set stores a resource in the LRU cache. If the LRU is full, the evicted
// entry is dropped from memory (its args, if any, remain in SQLite).
func (s *SqliteResources) Set(key string, value Resource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lru.set(key, value)
}

// SetWithArgs stores a resource in the LRU cache and persists its creation
// args to SQLite so the resource can be reconstructed after LRU eviction.
// Internal state (SerializableInternal) is NOT persisted here because it is
// set imperatively after CreateResource returns. It is persisted later via
// the onLRUEvict callback when the resource is evicted from the LRU.
func (s *SqliteResources) SetWithArgs(key string, value Resource, args map[string]*llx.RawData) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lru.set(key, value)

	if s.db == nil {
		return
	}

	// During reconstruction, CreateResource calls back into SetWithArgs for
	// the same key. The SQLite row already exists with valid data, so skip
	// the write to avoid overwriting internal_data with NULL.
	if _, ok := s.inFlight[key]; ok {
		return
	}

	argsBytes, err := marshalArgs(args)
	if err != nil {
		log.Warn().Err(err).Str("key", key).Msg("sqlite resource cache: failed to marshal args")
		return
	}

	_, err = s.db.Exec(
		`INSERT OR REPLACE INTO resources (cache_key, name, res_id, args) VALUES (?, ?, ?, ?)`,
		key, value.MqlName(), value.MqlID(), argsBytes,
	)
	if err != nil {
		log.Warn().Err(err).Str("key", key).Msg("sqlite resource cache: failed to persist resource")
	}
}

// onLRUEvict is called when a resource is evicted from the LRU cache.
// It persists the resource's internal state to SQLite so it can be restored
// on reconstruction. Called while s.mu is held — must not acquire the mutex.
func (s *SqliteResources) onLRUEvict(key string, resource Resource) {
	if s.db == nil {
		return
	}
	si, ok := resource.(SerializableInternal)
	if !ok {
		return
	}
	internalData, err := si.MarshalInternal()
	if err != nil {
		log.Warn().Err(err).Str("key", key).Msg("sqlite resource cache: failed to marshal internal data on eviction")
		return
	}
	if len(internalData) == 0 {
		return
	}
	_, err = s.db.Exec(`UPDATE resources SET internal_data = ? WHERE cache_key = ?`, internalData, key)
	if err != nil {
		log.Warn().Err(err).Str("key", key).Msg("sqlite resource cache: failed to persist internal data on eviction")
	}
}

// Close closes the SQLite database and removes the temp file.
func (s *SqliteResources) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var err error
	if s.db != nil {
		err = s.db.Close()
		s.db = nil
	}
	if s.dbPath != "" {
		os.Remove(s.dbPath)
		os.Remove(s.dbPath + "-wal")
		os.Remove(s.dbPath + "-shm")
		s.dbPath = ""
	}
	return err
}

// marshalArgs serializes a map[string]*llx.RawData to bytes using protobuf.
func marshalArgs(args map[string]*llx.RawData) ([]byte, error) {
	if len(args) == 0 {
		return nil, nil
	}

	pmap := make(map[string]*llx.Primitive, len(args))
	for k, v := range args {
		result := v.Result()
		if result.Data != nil {
			pmap[k] = result.Data
		}
	}

	wrapper := &llx.Primitive{
		Type: string("map[string]any"),
		Map:  pmap,
	}
	return wrapper.MarshalVT()
}

// unmarshalArgs deserializes bytes back to a map[string]*llx.Primitive.
func unmarshalArgs(data []byte) (map[string]*llx.Primitive, error) {
	if len(data) == 0 {
		return map[string]*llx.Primitive{}, nil
	}

	wrapper := &llx.Primitive{}
	if err := wrapper.UnmarshalVT(data); err != nil {
		return nil, err
	}
	if wrapper.Map == nil {
		return map[string]*llx.Primitive{}, nil
	}
	return wrapper.Map, nil
}

// lruCache is a simple bounded LRU cache backed by a doubly-linked list.
type lruCache struct {
	capacity int
	items    map[string]*list.Element
	order    *list.List
	onEvict  func(key string, resource Resource)
}

type lruEntry struct {
	key      string
	resource Resource
}

func newLRUCache(capacity int) *lruCache {
	return &lruCache{
		capacity: capacity,
		items:    make(map[string]*list.Element, capacity),
		order:    list.New(),
	}
}

// get returns the resource and promotes it to the front. Not thread-safe.
func (c *lruCache) get(key string) (Resource, bool) {
	elem, ok := c.items[key]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(elem)
	return elem.Value.(*lruEntry).resource, true
}

// set adds or updates an entry, evicting the oldest if at capacity. Not thread-safe.
func (c *lruCache) set(key string, value Resource) {
	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		elem.Value.(*lruEntry).resource = value
		return
	}

	if c.order.Len() >= c.capacity {
		c.evict()
	}

	entry := &lruEntry{key: key, resource: value}
	elem := c.order.PushFront(entry)
	c.items[key] = elem
}

// evict removes the least recently used entry. Not thread-safe.
func (c *lruCache) evict() {
	back := c.order.Back()
	if back == nil {
		return
	}
	entry := back.Value.(*lruEntry)
	if c.onEvict != nil {
		c.onEvict(entry.key, entry.resource)
	}
	delete(c.items, entry.key)
	c.order.Remove(back)
}
