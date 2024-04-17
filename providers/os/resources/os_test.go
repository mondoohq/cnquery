// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/testutils"
)

var x = testutils.InitTester(testutils.LinuxMock())

func testWindowsQuery(t *testing.T, query string) []*llx.RawResult {
	win := testutils.InitTester(testutils.WindowsMock())
	return win.TestQuery(t, query)
}
