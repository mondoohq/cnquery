// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Groups(t *testing.T) {
	t.Run("list groups", func(t *testing.T) {
		res := x.TestQuery(t, "groups.list")
		assert.NotEmpty(t, res)
	})

	t.Run("test a specific grroup", func(t *testing.T) {
		res := x.TestQuery(t, "groups.list[0].name")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "root", res[0].Data.Value)
	})

	t.Run("test group init (gid)", func(t *testing.T) {
		res := x.TestQuery(t, "group(gid: 1000).name")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "chris", res[0].Data.Value)
	})

	t.Run("test group init (name)", func(t *testing.T) {
		res := x.TestQuery(t, "group(name: 'chris').gid")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(1000), res[0].Data.Value)
	})
}
