// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package plugin

type Resources[T any] interface {
	Get(key string) (T, bool)
	Set(key string, value T)
}
