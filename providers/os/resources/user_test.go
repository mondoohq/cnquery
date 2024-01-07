// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Users(t *testing.T) {
	t.Run("users list", func(t *testing.T) {
		res := x.TestQuery(t, "users.list")
		assert.NotEmpty(t, res)
	})

	t.Run("test a specific user's name", func(t *testing.T) {
		res := x.TestQuery(t, "users.list[0].name")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "root", res[0].Data.Value)
	})

	t.Run("test contains", func(t *testing.T) {
		res := x.TestQuery(t, "users.contains(name == 'root')")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, true, res[1].Data.Value)
	})

	t.Run("test contains", func(t *testing.T) {
		res := x.TestQuery(t, "users.contains(group != null)")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, true, res[1].Data.Value)
	})

	t.Run("test user init (uid)", func(t *testing.T) {
		res := x.TestQuery(t, "user(uid: 1000).name")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "chris", res[0].Data.Value)
	})

	t.Run("test user init (name)", func(t *testing.T) {
		res := x.TestQuery(t, "user(name: 'chris').uid")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(1000), res[0].Data.Value)
	})
}
