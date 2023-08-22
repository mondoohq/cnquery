// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleLoad(t *testing.T) {
	t.Run("load bundle from file", func(t *testing.T) {
		bundle, err := BundleFromPaths("../examples/os.mql.yaml")
		require.NoError(t, err)
		assert.Equal(t, 1, len(bundle.Packs))
		assert.Equal(t, 3, len(bundle.Packs[0].Queries))

		// ensure that the uid is generated
		assert.True(t, len(bundle.Packs[0].Queries[0].Uid) > 0)
	})

	t.Run("load bundle from memory", func(t *testing.T) {
		data, err := os.ReadFile("../examples/os.mql.yaml")
		require.NoError(t, err)
		bundle, err := BundleFromYAML(data)
		require.NoError(t, err)
		assert.Equal(t, 1, len(bundle.Packs))
		assert.Equal(t, 3, len(bundle.Packs[0].Queries))

		// ensure that the uid is generated
		assert.True(t, len(bundle.Packs[0].Queries[0].Uid) > 0)
	})
}
