// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"regexp"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// objectIdRegex matches a Microsoft Entra directory object ID (a GUID).
var objectIdRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// resolveDirectoryRefs turns a Conditional Access scope list into typed
// resource references. Conditional Access mixes object IDs with special
// tokens (`All`, `None`, `GuestsOrExternalUsers`, `ServicePrincipalsInMyTenant`,
// …) in the same list, so anything that isn't a GUID is skipped.
func resolveDirectoryRefs(runtime *plugin.Runtime, resource string, ids []any) ([]any, error) {
	res := []any{}
	for _, raw := range ids {
		id, ok := raw.(string)
		if !ok || !objectIdRegex.MatchString(id) {
			continue
		}
		r, err := runtime.NewResource(runtime, resource, map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (c *mqlMicrosoftConditionalAccessPolicyConditionsUsers) includeUsersRefs() ([]any, error) {
	return resolveDirectoryRefs(c.MqlRuntime, "microsoft.user", c.IncludeUsers.Data)
}

func (c *mqlMicrosoftConditionalAccessPolicyConditionsUsers) excludeUsersRefs() ([]any, error) {
	return resolveDirectoryRefs(c.MqlRuntime, "microsoft.user", c.ExcludeUsers.Data)
}

func (c *mqlMicrosoftConditionalAccessPolicyConditionsUsers) includeGroupsRefs() ([]any, error) {
	return resolveDirectoryRefs(c.MqlRuntime, "microsoft.group", c.IncludeGroups.Data)
}

func (c *mqlMicrosoftConditionalAccessPolicyConditionsUsers) excludeGroupsRefs() ([]any, error) {
	return resolveDirectoryRefs(c.MqlRuntime, "microsoft.group", c.ExcludeGroups.Data)
}

func (c *mqlMicrosoftConditionalAccessPolicyConditionsUsers) includeRolesRefs() ([]any, error) {
	return resolveDirectoryRefs(c.MqlRuntime, "microsoft.rolemanagement.roledefinition", c.IncludeRoles.Data)
}

func (c *mqlMicrosoftConditionalAccessPolicyConditionsUsers) excludeRolesRefs() ([]any, error) {
	return resolveDirectoryRefs(c.MqlRuntime, "microsoft.rolemanagement.roledefinition", c.ExcludeRoles.Data)
}

func (c *mqlMicrosoftConditionalAccessPolicyConditionsClientApplications) includeServicePrincipalsRefs() ([]any, error) {
	return resolveDirectoryRefs(c.MqlRuntime, "microsoft.serviceprincipal", c.IncludeServicePrincipals.Data)
}

func (c *mqlMicrosoftConditionalAccessPolicyConditionsClientApplications) excludeServicePrincipalsRefs() ([]any, error) {
	return resolveDirectoryRefs(c.MqlRuntime, "microsoft.serviceprincipal", c.ExcludeServicePrincipals.Data)
}
