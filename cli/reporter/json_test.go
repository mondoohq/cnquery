// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/testutils"
	"go.mondoo.com/cnquery/v10/shared"
)

var x = testutils.InitTester(testutils.LinuxMock())

func testQuery(t *testing.T, query string) (*llx.CodeBundle, map[string]*llx.RawResult) {
	codeBundle, err := x.Compile(query)
	require.NoError(t, err)

	results, err := x.ExecuteCode(codeBundle, nil)
	require.NoError(t, err)

	return codeBundle, results
}

type simpleTest struct {
	code     string
	expected string
}

func runSimpleTests(t *testing.T, tests []simpleTest) {
	var out strings.Builder
	w := shared.IOWriter{Writer: &out}

	for i := range tests {
		cur := tests[i]
		t.Run(cur.code, func(t *testing.T) {
			bundle, results := testQuery(t, cur.code)
			err := BundleResultsToJSON(bundle, results, &w)
			require.NoError(t, err)
			assert.Equal(t, cur.expected, out.String())
		})
	}
}

func TestJsonReporter(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"users.where(uid==0)",
			`{"users.where.list":[{"gid":0,"name":"root","uid":0}]}`,
		},
	})
}
