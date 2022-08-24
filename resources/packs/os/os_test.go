package os_test

import (
	"testing"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/resources/packs/os"
	"go.mondoo.com/cnquery/resources/packs/testutils"
)

var x = testutils.InitTester(testutils.LinuxMock(), os.Registry)

func testWindowsQuery(t *testing.T, query string) []*llx.RawResult {
	x := testutils.InitTester(testutils.Mock("../testdata/windows.toml"), os.Registry)
	return x.TestQuery(t, query)
}
