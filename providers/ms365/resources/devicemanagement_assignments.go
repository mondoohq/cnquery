// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	betamodels "github.com/microsoftgraph/msgraph-beta-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// assignmentTargetInfo extracts the discriminator, group id, exclusion flag,
// and optional assignment-filter metadata from a Graph assignment target.
// The filter id and type live in AdditionalData because the v1 SDK does not
// expose typed getters for them on the base target interface.
func assignmentTargetInfo(target models.DeviceAndAppManagementAssignmentTargetable) (targetType, groupId string, excluded bool, filterType, filterId string) {
	if target == nil {
		return "", "", false, "", ""
	}
	if t := target.GetOdataType(); t != nil {
		targetType = trimOdataType(*t)
	}
	switch concrete := target.(type) {
	case *models.GroupAssignmentTarget:
		if g := concrete.GetGroupId(); g != nil {
			groupId = *g
		}
	case *models.ExclusionGroupAssignmentTarget:
		excluded = true
		if g := concrete.GetGroupId(); g != nil {
			groupId = *g
		}
	}
	add := target.GetAdditionalData()
	if v, ok := add["deviceAndAppManagementAssignmentFilterType"].(string); ok {
		filterType = v
	}
	if v, ok := add["deviceAndAppManagementAssignmentFilterId"].(string); ok {
		filterId = v
	}
	return targetType, groupId, excluded, filterType, filterId
}

// trimOdataType strips the leading "#microsoft.graph." namespace from an
// @odata.type value so the result reads as a plain discriminator name.
func trimOdataType(s string) string {
	const prefix = "#microsoft.graph."
	if len(s) > len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

// newPolicyAssignmentResource builds a microsoft.devicemanagement.policyAssignment
// from an assignment id and target. The id is used as the resource cache key.
func newPolicyAssignmentResource(runtime *plugin.Runtime, id string, target models.DeviceAndAppManagementAssignmentTargetable) (any, error) {
	targetType, groupId, excluded, filterType, filterId := assignmentTargetInfo(target)
	return CreateResource(runtime, "microsoft.devicemanagement.policyAssignment",
		map[string]*llx.RawData{
			"__id":       llx.StringData(id),
			"id":         llx.StringData(id),
			"targetType": llx.StringData(targetType),
			"groupId":    llx.StringData(groupId),
			"excluded":   llx.BoolData(excluded),
			"filterType": llx.StringData(filterType),
			"filterId":   llx.StringData(filterId),
		})
}

// betaAssignmentTargetInfo mirrors assignmentTargetInfo for the beta SDK's
// parallel target type. Endpoint security intents, app detections per device,
// and Windows Autopilot live on the beta endpoint.
func betaAssignmentTargetInfo(target betamodels.DeviceAndAppManagementAssignmentTargetable) (targetType, groupId string, excluded bool, filterType, filterId string) {
	if target == nil {
		return "", "", false, "", ""
	}
	if t := target.GetOdataType(); t != nil {
		targetType = trimOdataType(*t)
	}
	switch concrete := target.(type) {
	case *betamodels.GroupAssignmentTarget:
		if g := concrete.GetGroupId(); g != nil {
			groupId = *g
		}
	case *betamodels.ExclusionGroupAssignmentTarget:
		excluded = true
		if g := concrete.GetGroupId(); g != nil {
			groupId = *g
		}
	}
	add := target.GetAdditionalData()
	if v, ok := add["deviceAndAppManagementAssignmentFilterType"].(string); ok {
		filterType = v
	}
	if v, ok := add["deviceAndAppManagementAssignmentFilterId"].(string); ok {
		filterId = v
	}
	return targetType, groupId, excluded, filterType, filterId
}

// newBetaPolicyAssignmentResource mirrors newPolicyAssignmentResource for the
// beta SDK. The resulting MQL schema is identical to the v1 path.
func newBetaPolicyAssignmentResource(runtime *plugin.Runtime, id string, target betamodels.DeviceAndAppManagementAssignmentTargetable) (any, error) {
	targetType, groupId, excluded, filterType, filterId := betaAssignmentTargetInfo(target)
	return CreateResource(runtime, "microsoft.devicemanagement.policyAssignment",
		map[string]*llx.RawData{
			"__id":       llx.StringData(id),
			"id":         llx.StringData(id),
			"targetType": llx.StringData(targetType),
			"groupId":    llx.StringData(groupId),
			"excluded":   llx.BoolData(excluded),
			"filterType": llx.StringData(filterType),
			"filterId":   llx.StringData(filterId),
		})
}
