// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func TestVulnmgmtSoftFailsWhenReportUnavailable(t *testing.T) {
	v := &mqlVulnmgmt{MqlRuntime: &plugin.Runtime{}}

	cves := v.GetCves()
	require.NoError(t, cves.Error)
	assert.True(t, cves.IsSet())
	assert.False(t, cves.IsNull())
	assert.Len(t, cves.Data, 0)

	advisories := v.GetAdvisories()
	require.NoError(t, advisories.Error)
	assert.Len(t, advisories.Data, 0)

	packages := v.GetPackages()
	require.NoError(t, packages.Error)
	assert.Len(t, packages.Data, 0)

	stats := v.GetStats()
	require.NoError(t, stats.Error)
	assert.True(t, stats.IsSet())
	assert.True(t, stats.IsNull())

	lastAssessment := v.GetLastAssessment()
	require.NoError(t, lastAssessment.Error)
	assert.True(t, lastAssessment.IsSet())
	assert.True(t, lastAssessment.IsNull())

	assert.True(t, v.warnedUnavailable)
}
