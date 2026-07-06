// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	authorization "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
)

// scopeForSubscription is the PIM list scope for a subscription's privileged
// role schedules.
func scopeForSubscription(subID string) string {
	return "/subscriptions/" + subID
}

// pimUnavailable reports whether a PIM list error is a 4xx — most commonly
// AadPremiumLicenseRequired on tenants without Entra ID P2 / Governance, but
// also access-denied. Such tenants simply have no PIM data, so callers treat it
// as an empty result rather than failing the whole authorization query.
func pimUnavailable(err error) bool {
	var respErr *azcore.ResponseError
	return errors.As(err, &respErr) && respErr.StatusCode >= 400 && respErr.StatusCode < 500
}

type mqlAzureSubscriptionAuthorizationServiceRoleEligibilityScheduleInternal struct {
	roleDefinitionId string
}

func (a *mqlAzureSubscriptionAuthorizationService) roleEligibilitySchedules() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	client, err := authorization.NewRoleEligibilitySchedulesClient(conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []any{}
	pager := client.NewListForScopePager(scopeForSubscription(a.SubscriptionId.Data), &authorization.RoleEligibilitySchedulesClientListForScopeOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			if pimUnavailable(err) {
				log.Warn().Str("subscription", a.SubscriptionId.Data).Msg("PIM role schedules unavailable (requires Entra ID P2 / Governance)")
				return res, nil
			}
			return nil, err
		}
		for _, sched := range page.Value {
			if sched == nil {
				continue
			}
			var principalId, principalType, principalDisplayName, scope, status string
			var memberType, condition, roleDefinitionId, roleDefinitionName string
			var startDateTime, endDateTime *time.Time
			if p := sched.Properties; p != nil {
				principalId = convert.ToValue(p.PrincipalID)
				if p.PrincipalType != nil {
					principalType = string(*p.PrincipalType)
				}
				scope = convert.ToValue(p.Scope)
				if p.Status != nil {
					status = string(*p.Status)
				}
				if p.MemberType != nil {
					memberType = string(*p.MemberType)
				}
				condition = convert.ToValue(p.Condition)
				roleDefinitionId = convert.ToValue(p.RoleDefinitionID)
				startDateTime = p.StartDateTime
				endDateTime = p.EndDateTime
				principalDisplayName, roleDefinitionName = expandedPrincipalAndRoleNames(p.ExpandedProperties)
			}
			mqlSched, err := CreateResource(a.MqlRuntime, "azure.subscription.authorizationService.roleEligibilitySchedule",
				map[string]*llx.RawData{
					"id":                   llx.StringDataPtr(sched.ID),
					"name":                 llx.StringDataPtr(sched.Name),
					"principalId":          llx.StringData(principalId),
					"principalType":        llx.StringData(principalType),
					"principalDisplayName": llx.StringData(principalDisplayName),
					"scope":                llx.StringData(scope),
					"status":               llx.StringData(status),
					"memberType":           llx.StringData(memberType),
					"startDateTime":        llx.TimeDataPtr(startDateTime),
					"endDateTime":          llx.TimeDataPtr(endDateTime),
					"condition":            llx.StringData(condition),
					"roleDefinitionName":   llx.StringData(roleDefinitionName),
				})
			if err != nil {
				return nil, err
			}
			mqlSched.(*mqlAzureSubscriptionAuthorizationServiceRoleEligibilitySchedule).roleDefinitionId = roleDefinitionId
			res = append(res, mqlSched)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionAuthorizationServiceRoleEligibilitySchedule) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionAuthorizationServiceRoleEligibilitySchedule) roleDefinition() (*mqlAzureSubscriptionAuthorizationServiceRoleDefinition, error) {
	return resolveRoleDefinition(a.MqlRuntime, a.roleDefinitionId, &a.RoleDefinition)
}

type mqlAzureSubscriptionAuthorizationServiceRoleAssignmentScheduleInternal struct {
	roleDefinitionId string
}

func (a *mqlAzureSubscriptionAuthorizationService) roleAssignmentSchedules() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	client, err := authorization.NewRoleAssignmentSchedulesClient(conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []any{}
	pager := client.NewListForScopePager(scopeForSubscription(a.SubscriptionId.Data), &authorization.RoleAssignmentSchedulesClientListForScopeOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			if pimUnavailable(err) {
				log.Warn().Str("subscription", a.SubscriptionId.Data).Msg("PIM role schedules unavailable (requires Entra ID P2 / Governance)")
				return res, nil
			}
			return nil, err
		}
		for _, sched := range page.Value {
			if sched == nil {
				continue
			}
			var principalId, principalType, principalDisplayName, scope, status string
			var assignmentType, memberType, condition, roleDefinitionId, roleDefinitionName string
			var startDateTime, endDateTime *time.Time
			if p := sched.Properties; p != nil {
				principalId = convert.ToValue(p.PrincipalID)
				if p.PrincipalType != nil {
					principalType = string(*p.PrincipalType)
				}
				scope = convert.ToValue(p.Scope)
				if p.Status != nil {
					status = string(*p.Status)
				}
				if p.AssignmentType != nil {
					assignmentType = string(*p.AssignmentType)
				}
				if p.MemberType != nil {
					memberType = string(*p.MemberType)
				}
				condition = convert.ToValue(p.Condition)
				roleDefinitionId = convert.ToValue(p.RoleDefinitionID)
				startDateTime = p.StartDateTime
				endDateTime = p.EndDateTime
				principalDisplayName, roleDefinitionName = expandedPrincipalAndRoleNames(p.ExpandedProperties)
			}
			mqlSched, err := CreateResource(a.MqlRuntime, "azure.subscription.authorizationService.roleAssignmentSchedule",
				map[string]*llx.RawData{
					"id":                   llx.StringDataPtr(sched.ID),
					"name":                 llx.StringDataPtr(sched.Name),
					"principalId":          llx.StringData(principalId),
					"principalType":        llx.StringData(principalType),
					"principalDisplayName": llx.StringData(principalDisplayName),
					"scope":                llx.StringData(scope),
					"assignmentType":       llx.StringData(assignmentType),
					"status":               llx.StringData(status),
					"memberType":           llx.StringData(memberType),
					"startDateTime":        llx.TimeDataPtr(startDateTime),
					"endDateTime":          llx.TimeDataPtr(endDateTime),
					"condition":            llx.StringData(condition),
					"roleDefinitionName":   llx.StringData(roleDefinitionName),
				})
			if err != nil {
				return nil, err
			}
			mqlSched.(*mqlAzureSubscriptionAuthorizationServiceRoleAssignmentSchedule).roleDefinitionId = roleDefinitionId
			res = append(res, mqlSched)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionAuthorizationServiceRoleAssignmentSchedule) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionAuthorizationServiceRoleAssignmentSchedule) roleDefinition() (*mqlAzureSubscriptionAuthorizationServiceRoleDefinition, error) {
	return resolveRoleDefinition(a.MqlRuntime, a.roleDefinitionId, &a.RoleDefinition)
}

// expandedPrincipalAndRoleNames pulls the human-readable principal and role
// display names out of a PIM schedule's expanded properties, so audits don't
// have to resolve GUIDs.
func expandedPrincipalAndRoleNames(ep *authorization.ExpandedProperties) (principalDisplayName, roleDefinitionName string) {
	if ep == nil {
		return "", ""
	}
	if ep.Principal != nil {
		principalDisplayName = convert.ToValue(ep.Principal.DisplayName)
	}
	if ep.RoleDefinition != nil {
		roleDefinitionName = convert.ToValue(ep.RoleDefinition.DisplayName)
	}
	return principalDisplayName, roleDefinitionName
}

// resolveRoleDefinition resolves a role definition ID to its typed resource via
// the subscription's cached IAM roles, mirroring roleAssignment.role(). Returns
// null when the ID is empty.
func resolveRoleDefinition(runtime *plugin.Runtime, roleDefinitionId string, field *plugin.TValue[*mqlAzureSubscriptionAuthorizationServiceRoleDefinition]) (*mqlAzureSubscriptionAuthorizationServiceRoleDefinition, error) {
	if roleDefinitionId == "" {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	subId, err := extractSubscriptionID(roleDefinitionId)
	if err != nil {
		return nil, err
	}
	r, err := CreateResource(runtime, "azure.subscription", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(subId),
	})
	if err != nil {
		return nil, err
	}
	iam := r.(*mqlAzureSubscription).GetIam()
	if iam.Error != nil {
		return nil, iam.Error
	}
	rolesVal := iam.Data.GetRoles()
	if rolesVal.Error != nil {
		return nil, rolesVal.Error
	}
	for i := range rolesVal.Data {
		role := rolesVal.Data[i].(*mqlAzureSubscriptionAuthorizationServiceRoleDefinition)
		if role.__id == roleDefinitionId {
			return role, nil
		}
	}
	// The role definition isn't in the subscription's cached list (e.g. a custom
	// role scoped elsewhere, or eventual consistency). Don't fail the schedule
	// query over it — surface the schedule with a null role reference.
	field.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}
