// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Services(t *testing.T) {
	t.Run("list services", func(t *testing.T) {
		res := x.TestQuery(t, "services.list")
		assert.NotEmpty(t, res)
	})

	t.Run("test a specific service entry", func(t *testing.T) {
		res := x.TestQuery(t, "services.list[0].name")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "acpid", res[0].Data.Value)
	})
}

func TestResource_Service(t *testing.T) {
	t.Run("test a specific service name", func(t *testing.T) {
		res := x.TestQuery(t, "service('dbus').name")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "dbus", res[0].Data.Value)
	})

	t.Run("test a specific service enabled", func(t *testing.T) {
		res := x.TestQuery(t, "service('dbus').enabled")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, true, res[0].Data.Value)
	})

	t.Run("test a specific service running", func(t *testing.T) {
		res := x.TestQuery(t, "service('dbus').running")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, true, res[0].Data.Value)
	})
}
