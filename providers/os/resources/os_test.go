package resources_test

import "go.mondoo.com/cnquery/providers-sdk/v1/testutils"

var x = testutils.InitTester(testutils.LinuxMock("../../../providers-sdk/v1/testutils"))

// func testWindowsQuery(t *testing.T, query string) []*llx.RawResult {
// 	x := testutils.InitTester(testutils.Mock("../testdata/windows.toml"), os.Registry)
// 	return x.TestQuery(t, query)
// }
