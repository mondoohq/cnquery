// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Containers(t *testing.T) {
	t.Run("list containers", func(t *testing.T) {
		res := x.TestQuery(t, "containers.list")
		assert.NotEmpty(t, res)
	})

	t.Run("test container fields", func(t *testing.T) {
		res := x.TestQuery(t, "containers.list[0].id")
		if len(res) == 0 {
			t.Skip("no containers found")
		}
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
	})

	t.Run("test running containers", func(t *testing.T) {
		res := x.TestQuery(t, "containers.running")
		assert.NotEmpty(t, res)
	})
}
