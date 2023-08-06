package resources_test

import (
	"testing"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/testutils"
)

var x = testutils.InitTester(testutils.LinuxMock("../../../providers-sdk/v1/testutils"))

func testWindowsQuery(t *testing.T, query string) []*llx.RawResult {
	win := testutils.InitTester(testutils.WindowsMock("../../../providers-sdk/v1/testutils"))
	return win.TestQuery(t, query)
}
