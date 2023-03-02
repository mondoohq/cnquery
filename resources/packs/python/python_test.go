package python_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.mondoo.com/cnquery/resources/packs/all/info"
	"go.mondoo.com/cnquery/resources/packs/core"
	"go.mondoo.com/cnquery/resources/packs/python"
	"go.mondoo.com/cnquery/resources/packs/testutils"
)

var Registry = info.Registry

func TestResource_Python(t *testing.T) {
	Registry.Add(core.Registry)
	Registry.Add(python.Registry)

	x := testutils.InitTester(testutils.CustomMock("./testdata/linux.toml"), Registry)

	t.Run("parse all packages", func(t *testing.T) {
		res := x.TestQuery(t, "python.packages")
		assert.NotEmpty(t, res)
		require.Empty(t, res[0].Result().Error)
		values, ok := res[0].Data.Value.([]interface{})
		require.True(t, ok, "type assertion failed")
		assert.Equal(t, 2, len(values), "expected two parsed packages")
	})
}

func TestResource_PythonPackage(t *testing.T) {
	Registry.Add(core.Registry)
	Registry.Add(python.Registry)

	x := testutils.InitTester(testutils.CustomMock("./testdata/linux.toml"), Registry)

	t.Run("parse single package", func(t *testing.T) {
		res := x.TestQuery(t, "python.package(\"/usr/lib/python3.11/site-packages/python_ftp_server-1.3.17.dist-info/METADATA\").name")
		assert.NotEmpty(t, res)
		require.Empty(t, res[0].Result().Error)
		assert.Equal(t, "python-ftp-server", res[0].Data.Value, "expected name of parsed package")
	})
}
