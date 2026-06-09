// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func TestCapturesAllManagementEvents(t *testing.T) {
	selector := func(mgmt bool, readWriteType string) *mqlAwsCloudtrailTrailEventSelector {
		sel := &mqlAwsCloudtrailTrailEventSelector{}
		sel.IncludeManagementEvents = plugin.TValue[bool]{Data: mgmt, State: plugin.StateIsSet}
		sel.ReadWriteType = plugin.TValue[string]{Data: readWriteType, State: plugin.StateIsSet}
		return sel
	}
	trail := func(sels ...*mqlAwsCloudtrailTrailEventSelector) *mqlAwsCloudtrailTrail {
		entries := make([]any, len(sels))
		for i, s := range sels {
			entries[i] = s
		}
		tr := &mqlAwsCloudtrailTrail{}
		tr.EventSelectorEntries = plugin.TValue[[]any]{Data: entries, State: plugin.StateIsSet}
		return tr
	}

	tests := []struct {
		name string
		sels []*mqlAwsCloudtrailTrailEventSelector
		want bool
	}{
		{"management + All", []*mqlAwsCloudtrailTrailEventSelector{selector(true, "All")}, true},
		{"management but write only", []*mqlAwsCloudtrailTrailEventSelector{selector(true, "WriteOnly")}, false},
		{"All but no management", []*mqlAwsCloudtrailTrailEventSelector{selector(false, "All")}, false},
		{"matches among several", []*mqlAwsCloudtrailTrailEventSelector{selector(false, "ReadOnly"), selector(true, "All")}, true},
		{"no selectors", nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := trail(tc.sels...).capturesAllManagementEvents()
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
