package reporter

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/shared"
)

func TestCSVExport(t *testing.T) {
	data, err := os.ReadFile("testdata/kubernetes_report.json")
	require.NoError(t, err)

	var report *explorer.ReportCollection
	err = json.Unmarshal(data, &report)
	require.NoError(t, err)
	w := shared.IOWriter{Writer: os.Stdout}
	err = ReportCollectionToCSV(report, &w)
	require.NoError(t, err)
}
