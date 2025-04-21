// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientsCache(t *testing.T) {
	t.Run("cache enabled by default", func(t *testing.T) {
		subject := ClientsCache{}
		subject.Store("key", &CacheEntry{Data: "value"})
		entry, found := subject.Load("key")
		require.True(t, found)
		require.NotNil(t, entry)
	})
	t.Run("cache disabled", func(t *testing.T) {
		subject := ClientsCache{disabled: true}
		subject.Store("key", &CacheEntry{Data: "value"})
		entry, found := subject.Load("key")
		require.False(t, found)
		require.Nil(t, entry)
	})
}
