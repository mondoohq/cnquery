// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/providers-sdk/v1/testutils"
)

func TestResource_Python(t *testing.T) {
	x := testutils.InitTester(testutils.RecordingMock("../../../providers-sdk/v1/testutils", "./python/testdata/linux.json"))

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
	x := testutils.InitTester(testutils.RecordingMock("../../../providers-sdk/v1/testutils", "./python/testdata/python-package.json"))

	t.Run("parse single package", func(t *testing.T) {
		res := x.TestQuery(t, "python.package(\"/usr/lib/python3/dist-packages/python_ftp_server-1.3.17.dist-info/METADATA\").name")
		assert.NotEmpty(t, res)
		require.Empty(t, res[0].Result().Error)
		assert.Equal(t, "python-ftp-server", res[0].Data.Value, "expected name of parsed package")
	})
}
