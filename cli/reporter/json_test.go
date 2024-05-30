// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/explorer"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/testutils"
	"go.mondoo.com/cnquery/v11/utils/iox"
	"sigs.k8s.io/yaml"
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

func TestQueryConversion(t *testing.T) {
	tests := []simpleTest{
		{
			"users.where(uid==0)",
			`{"users.where.list":[{"gid":0,"name":"root","uid":0}]}`,
		},
	}

	var out strings.Builder
	w := iox.IOWriter{Writer: &out}

	for i := range tests {
		cur := tests[i]
		t.Run(cur.code, func(t *testing.T) {
			bundle, results := testQuery(t, cur.code)
			err := CodeBundleToJSON(bundle, results, &w)
			require.NoError(t, err)
			assert.Equal(t, cur.expected, out.String())
		})
	}
}

func TestJsonReporter(t *testing.T) {
	data, err := os.ReadFile("testdata/kubernetes_report.yaml")
	require.NoError(t, err)

	var report *explorer.ReportCollection
	err = yaml.Unmarshal(data, &report)
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	w := iox.IOWriter{Writer: buf}
	err = ConvertToJSON(report, &w)
	require.NoError(t, err)
}
