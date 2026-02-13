// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package syncx

import "sync"

type Map[T any] struct {
	sync.Map
}

func (r *Map[T]) Get(key string) (T, bool) {
	res, ok := r.Load(key)
	if !ok {
		var zero T
		return zero, ok
	}
	return res.(T), true
}

func (r *Map[T]) Set(key string, value T) {
	r.Store(key, value)
}
