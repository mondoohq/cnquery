// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inmemory

import (
	"time"

	"github.com/google/uuid"
	"go.mondoo.com/cnquery/v10/explorer"
	"go.mondoo.com/cnquery/v10/llx"
)

// Db is the database backend, it allows the interaction with the underlying data.
type Db struct {
	cache       kvStore
	services    *explorer.LocalServices // bidirectional connection between db + services
	uuid        string                  // used for all object identifiers to prevent clashes (eg in-memory pubsub)
	nowProvider func() time.Time
}

// NewServices creates a new set of backend services
func NewServices(runtime llx.Runtime) (*Db, *explorer.LocalServices, error) {
	var cache kvStore = newKissDb()

	db := &Db{
		cache:       cache,
		uuid:        uuid.New().String(),
		nowProvider: time.Now,
	}

	services := explorer.NewLocalServices(db, db.uuid, runtime)
	db.services = services // close the connection between db and services

	return db, services, nil
}

// WithDb creates a new set of backend services and closes everything out once the function is done
func WithDb(runtime llx.Runtime, f func(*Db, *explorer.LocalServices) error) error {
	db, ls, err := NewServices(runtime)
	if err != nil {
		return err
	}

	return f(db, ls)
}

// Prefixes for all keys that are stored in the cache.
// Prevent collisions by creating namespaces for different types of data.
const (
	dbIDQuery          = "q\x00"
	dbIDQueryPack      = "qp\x00"
	dbIDBundle         = "qb\x00"
	dbIDListQueryPacks = "qpl\x00"
	dbIDData           = "d\x00"
	dbIDAsset          = "a\x00"
	dbIDExecutionJob   = "ej\x00"
	dbIDresolvedPack   = "rpa\x00"
)

func (db *Db) SetNowProvider(f func() time.Time) {
	db.nowProvider = f
}
