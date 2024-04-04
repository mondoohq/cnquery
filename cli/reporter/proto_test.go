// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/explorer"
	"google.golang.org/protobuf/encoding/protojson"
	"os"
	"sigs.k8s.io/yaml"
	"testing"
)

func TestProtoReporter(t *testing.T) {
	data, err := os.ReadFile("testdata/kubernetes_report.yaml")
	require.NoError(t, err)

	var report explorer.ReportCollection
	err = yaml.Unmarshal(data, &report)
	require.NoError(t, err)

	protoReport, err := ConvertToProto(&report)
	require.NoError(t, err)
	assert.NotNil(t, protoReport)

	// test that the asset data is correctly converted
	assert.Equal(t, "kube-system/kube-proxy-gdsjm", protoReport.Assets["//explorer.api.mondoo.com/assets/2LgMkM8gOGx7j9uDwNADfXFGFpo"].Name)
	// test that errors are correctly converted
	assert.Equal(t, "asset does not match any of the activated query packs", protoReport.Errors["//explorer.api.mondoo.com/assets/2LgMkMIFzzNEh02hTBxOpc5mkdD"])

	// test that the data points are correctly converted
	queryMrn := "//local.cnquery.io/run/local-execution/queries/role-bindings-with-cluster-admin-permissions"
	data, err = protojson.Marshal(protoReport.Data["//explorer.api.mondoo.com/assets/2LgMkOR8vP9j7GgBbPj9hjYqjO2"].Values[queryMrn].Content)
	require.NoError(t, err)
	assert.Equal(t, "{\"k8s.rolebindings.where\":[]}", string(data))

	//data, err = protoReport.ToJSON()
	//require.NoError(t, err)
	//fmt.Print(string(data))
	//os.WriteFile("testdata/proto.json", data, 0700)
}
