// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/explorer"
	"go.mondoo.com/cnquery/v11/utils/iox"
	"sigs.k8s.io/yaml"
)

func TestCSVExport(t *testing.T) {
	data, err := os.ReadFile("testdata/kubernetes_report.yaml")
	require.NoError(t, err)

	var report *explorer.ReportCollection
	err = yaml.Unmarshal(data, &report)
	require.NoError(t, err)
	w := iox.IOWriter{Writer: os.Stdout}
	err = ConvertToCSV(report, &w)
	require.NoError(t, err)
}
