// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Packages(t *testing.T) {
	res := x.TestQuery(t, "packages")
	assert.NotEmpty(t, res)
}

// FIXME: currently disabled because we have to feed the entire recording
// into the provider
func testResource_Package(t *testing.T) {
	t.Run("existing package", func(t *testing.T) {
		res := x.TestQuery(t, "package(\"acl\").installed")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, true, res[0].Data.Value)
	})

	t.Run("missing package", func(t *testing.T) {
		res := x.TestQuery(t, "package(\"unknown\").installed")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, false, res[0].Data.Value)
	})
}
