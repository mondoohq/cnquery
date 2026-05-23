// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
)

func TestNewMqlHelmChart_KubeVersionAndAnnotations(t *testing.T) {
	t.Run("populated fields surface verbatim", func(t *testing.T) {
		c := &chart.Chart{Metadata: &chart.Metadata{
			Name:        "mychart",
			Version:     "1.2.3",
			KubeVersion: ">=1.27.0-0",
			Annotations: map[string]string{
				"artifacthub.io/license": "Apache-2.0",
				"artifacthub.io/changes": "Initial release",
			},
		}}

		mqlChart, err := newMqlHelmChart(newTestRuntime(), c)
		require.NoError(t, err)
		assert.Equal(t, ">=1.27.0-0", mqlChart.KubeVersion.Data)
		assert.Equal(t, map[string]any{
			"artifacthub.io/license": "Apache-2.0",
			"artifacthub.io/changes": "Initial release",
		}, mqlChart.Annotations.Data)
	})

	t.Run("missing fields stay empty / nil", func(t *testing.T) {
		c := &chart.Chart{Metadata: &chart.Metadata{
			Name:    "mychart",
			Version: "1.2.3",
		}}

		mqlChart, err := newMqlHelmChart(newTestRuntime(), c)
		require.NoError(t, err)
		assert.Equal(t, "", mqlChart.KubeVersion.Data)
		// A chart without `annotations:` parses to nil; the resource layer
		// passes NilData so audits can distinguish "no annotations" from
		// "explicitly empty annotations".
		assert.Nil(t, mqlChart.Annotations.Data)
	})
}
