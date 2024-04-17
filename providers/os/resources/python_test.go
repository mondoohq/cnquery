// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/testutils"
)

func TestResource_Python(t *testing.T) {
	x := testutils.InitTester(testutils.RecordingMock("./python/testdata/linux.json"))

	t.Run("parse all packages", func(t *testing.T) {
		res := x.TestQuery(t, "python.packages")
		assert.NotEmpty(t, res)
		require.Empty(t, res[0].Result().Error)
		values, ok := res[0].Data.Value.([]interface{})
		require.True(t, ok, "type assertion failed")
		assert.Equal(t, 136, len(values), "expected two parsed packages")
	})

	t.Run("parse child packages", func(t *testing.T) {
		res := x.TestQuery(t, "python.toplevel")
		assert.NotEmpty(t, res)
		require.Empty(t, res[0].Result().Error)
		values, ok := res[0].Data.Value.([]interface{})
		require.True(t, ok, "type assertion failed")
		assert.Equal(t, 3, len(values), "expected a single child/leaf package")
	})
}

func TestResource_PythonPackage(t *testing.T) {
	x := testutils.InitTester(testutils.RecordingMock("./python/testdata/rhel.json"))

	t.Run("parse python pkg info", func(t *testing.T) {
		res := x.TestQuery(t, "python.package(\"/usr/lib/python3.6/site-packages/python_dateutil-2.6.1-py3.6.egg-info/PKG-INFO\").name")
		assert.NotEmpty(t, res)
		require.Empty(t, res[0].Result().Error)
		assert.Equal(t, "python-dateutil", res[0].Data.Value, "expected name of parsed package")
	})

	t.Run("parse python metadata", func(t *testing.T) {
		res := x.TestQuery(t, "python.package(\"/usr/lib/python3.6/site-packages/six-1.11.0.dist-info/METADATA\").name")
		assert.NotEmpty(t, res)
		require.Empty(t, res[0].Result().Error)
		assert.Equal(t, "six", res[0].Data.Value, "expected name of parsed package")
	})

	t.Run("test python package cpes", func(t *testing.T) {
		res := x.TestQuery(t, "python.package(\"/usr/lib/python3.6/site-packages/six-1.11.0.dist-info/METADATA\").cpes.map(uri)[0]")
		assert.NotEmpty(t, res)
		require.Empty(t, res[0].Result().Error)
		assert.Equal(t, "cpe:2.3:a:six_project:six:1.11.0:*:*:*:*:*:*:*", res[0].Data.Value, "expected name of parsed package")
	})
}
