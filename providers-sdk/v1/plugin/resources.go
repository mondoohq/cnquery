// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package plugin

import "go.mondoo.com/mql/v13/llx"

type Resources[T any] interface {
	Get(key string) (T, bool)
	Set(key string, value T)
}

// ResourcesWithArgs is an optional extension that allows storing creation
// args alongside the resource. The SQLite-backed cache uses these args to
// reconstruct evicted resources from disk. Implementations that do not need
// this (e.g. syncx.Map) simply don't implement this interface.
type ResourcesWithArgs interface {
	SetWithArgs(key string, value Resource, args map[string]*llx.RawData)
}

// SerializableInternal is an optional interface that resources can implement
// to persist their internal state (e.g. k8s API objects) to the SQLite cache.
// Without this, internal state set imperatively after resource creation
// (like `r.(*mqlK8sNamespace).obj = &ns`) would be lost on cache eviction.
type SerializableInternal interface {
	MarshalInternal() ([]byte, error)
	UnmarshalInternal([]byte) error
}
