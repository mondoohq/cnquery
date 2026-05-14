// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	betamodels "github.com/microsoftgraph/msgraph-beta-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/stretchr/testify/assert"
)

func ptr[T any](v T) *T { return &v }

func TestTrimOdataType(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"strips graph namespace", "#microsoft.graph.groupAssignmentTarget", "groupAssignmentTarget"},
		{"unrelated prefix kept", "#other.namespace.thing", "#other.namespace.thing"},
		{"shorter than prefix kept", "#microsoft", "#microsoft"},
		{"empty string kept", "", ""},
		{"prefix-only string kept", "#microsoft.graph.", "#microsoft.graph."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, trimOdataType(tc.in))
		})
	}
}

func TestAssignmentTargetInfo_Nil(t *testing.T) {
	tt, gid, ex, ft, fid := assignmentTargetInfo(nil)
	assert.Empty(t, tt)
	assert.Empty(t, gid)
	assert.False(t, ex)
	assert.Empty(t, ft)
	assert.Empty(t, fid)
}

func TestAssignmentTargetInfo_GroupTarget(t *testing.T) {
	target := models.NewGroupAssignmentTarget()
	target.SetOdataType(ptr("#microsoft.graph.groupAssignmentTarget"))
	target.SetGroupId(ptr("group-abc"))
	target.SetAdditionalData(map[string]any{
		"deviceAndAppManagementAssignmentFilterType": "include",
		"deviceAndAppManagementAssignmentFilterId":   "filter-1",
	})

	tt, gid, ex, ft, fid := assignmentTargetInfo(target)
	assert.Equal(t, "groupAssignmentTarget", tt)
	assert.Equal(t, "group-abc", gid)
	assert.False(t, ex)
	assert.Equal(t, "include", ft)
	assert.Equal(t, "filter-1", fid)
}

func TestAssignmentTargetInfo_ExclusionGroupTarget(t *testing.T) {
	target := models.NewExclusionGroupAssignmentTarget()
	target.SetOdataType(ptr("#microsoft.graph.exclusionGroupAssignmentTarget"))
	target.SetGroupId(ptr("group-xyz"))

	tt, gid, ex, ft, fid := assignmentTargetInfo(target)
	assert.Equal(t, "exclusionGroupAssignmentTarget", tt)
	assert.Equal(t, "group-xyz", gid)
	assert.True(t, ex)
	assert.Empty(t, ft)
	assert.Empty(t, fid)
}

func TestAssignmentTargetInfo_FilterAdditionalDataWrongTypes(t *testing.T) {
	target := models.NewGroupAssignmentTarget()
	target.SetAdditionalData(map[string]any{
		"deviceAndAppManagementAssignmentFilterType": 42, // not a string
		"deviceAndAppManagementAssignmentFilterId":   nil,
	})

	_, _, _, ft, fid := assignmentTargetInfo(target)
	assert.Empty(t, ft, "non-string filterType should not be extracted")
	assert.Empty(t, fid, "nil filterId should not be extracted")
}

func TestBetaAssignmentTargetInfo_GroupTarget(t *testing.T) {
	target := betamodels.NewGroupAssignmentTarget()
	target.SetOdataType(ptr("#microsoft.graph.groupAssignmentTarget"))
	target.SetGroupId(ptr("beta-group"))
	target.SetAdditionalData(map[string]any{
		"deviceAndAppManagementAssignmentFilterType": "exclude",
		"deviceAndAppManagementAssignmentFilterId":   "beta-filter",
	})

	tt, gid, ex, ft, fid := betaAssignmentTargetInfo(target)
	assert.Equal(t, "groupAssignmentTarget", tt)
	assert.Equal(t, "beta-group", gid)
	assert.False(t, ex)
	assert.Equal(t, "exclude", ft)
	assert.Equal(t, "beta-filter", fid)
}

func TestBetaAssignmentTargetInfo_ExclusionGroupTarget(t *testing.T) {
	target := betamodels.NewExclusionGroupAssignmentTarget()
	target.SetGroupId(ptr("beta-excl"))

	_, gid, ex, _, _ := betaAssignmentTargetInfo(target)
	assert.Equal(t, "beta-excl", gid)
	assert.True(t, ex)
}

func TestBetaAssignmentTargetInfo_Nil(t *testing.T) {
	tt, gid, ex, ft, fid := betaAssignmentTargetInfo(nil)
	assert.Empty(t, tt)
	assert.Empty(t, gid)
	assert.False(t, ex)
	assert.Empty(t, ft)
	assert.Empty(t, fid)
}
