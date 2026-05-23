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

		mqlChart, err := newMqlHelmChart(newTestRuntime(), c, "")
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

		mqlChart, err := newMqlHelmChart(newTestRuntime(), c, "")
		require.NoError(t, err)
		assert.Equal(t, "", mqlChart.KubeVersion.Data)
		// A chart without `annotations:` parses to nil; the resource layer
		// passes NilData so audits can distinguish "no annotations" from
		// "explicitly empty annotations".
		assert.Nil(t, mqlChart.Annotations.Data)
	})
}

// loader.LoadFile can return a chart with a nil Metadata pointer when
// fed a malformed .tgz; newMqlHelmChart must refuse to dereference it.
func TestNewMqlHelmChart_NilMetadataReturnsError(t *testing.T) {
	t.Run("nil metadata", func(t *testing.T) {
		c := &chart.Chart{Metadata: nil}
		_, err := newMqlHelmChart(newTestRuntime(), c, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "metadata")
	})

	t.Run("nil chart", func(t *testing.T) {
		_, err := newMqlHelmChart(newTestRuntime(), nil, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "metadata")
	})
}

// A chart that lists two maintainers with the same name (permitted by
// the Helm spec — e.g., the same person appearing in two roles) used
// to silently dedupe through the CreateResource cache because the
// __id was `helm.maintainer:<chart>:<name>`. Indexing fixes that.
func TestMaintainersDoNotDedupeOnDuplicateName(t *testing.T) {
	c := &chart.Chart{Metadata: &chart.Metadata{
		Name:    "mychart",
		Version: "1.0.0",
		Maintainers: []*chart.Maintainer{
			{Name: "Alex", Email: "alex-primary@example.com"},
			{Name: "Alex", Email: "alex-secondary@example.com"},
		},
	}}
	mqlChart, err := newMqlHelmChart(newTestRuntime(), c, "")
	require.NoError(t, err)

	mqlChart.chartObj = c
	maints, err := mqlChart.maintainers()
	require.NoError(t, err)
	require.Len(t, maints, 2, "both Alex entries should materialize as distinct resources")

	// Different emails should be reachable through the typed resources.
	emails := map[string]bool{}
	for _, m := range maints {
		emails[m.(*mqlHelmMaintainer).Email.Data] = true
	}
	assert.True(t, emails["alex-primary@example.com"], "first Alex email kept")
	assert.True(t, emails["alex-secondary@example.com"], "second Alex email kept")
}

// Two charts that share name + version (real for feature-branch forks
// in a multi-chart directory) used to collide on the CreateResource
// cache because the __id ignored their path. Including ChartFullPath
// makes them coexist.
func TestChartIDIncludesPathToDeduplicate(t *testing.T) {
	runtime := newTestRuntime()
	a := &chart.Chart{Metadata: &chart.Metadata{Name: "shared", Version: "1.0.0"}}
	b := &chart.Chart{Metadata: &chart.Metadata{Name: "shared", Version: "1.0.0"}}

	// Two charts with identical name+version at distinct paths is the
	// real-world case for feature-branch forks in a multi-chart
	// directory. The connection layer passes each chart's load path
	// into newMqlHelmChart so the __id stays unique.
	mqlA, err := newMqlHelmChart(runtime, a, "/repo/charts/a")
	require.NoError(t, err)
	mqlB, err := newMqlHelmChart(runtime, b, "/repo/charts/b")
	require.NoError(t, err)

	// CreateResource dedupes by __id; identical names would land both
	// pointers on the cached first instance. Distinct paths must keep
	// them separate.
	assert.NotSame(t, mqlA, mqlB, "charts at distinct paths must not dedupe")

	// And the path-less fallback still produces a stable id when no
	// path is available — the failure mode is then "two same-name
	// charts collide" but that matches prior behavior.
	mqlNoPath, err := newMqlHelmChart(runtime, &chart.Chart{
		Metadata: &chart.Metadata{Name: "lone", Version: "1.0.0"},
	}, "")
	require.NoError(t, err)
	assert.NotEmpty(t, mqlNoPath.__id)
}
