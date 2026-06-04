// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// resolveDirectoryRefs turns a Conditional Access scope list into typed
// resource references. Conditional Access mixes object IDs with special
// tokens (`All`, `None`, `GuestsOrExternalUsers`, `ServicePrincipalsInMyTenant`,
// …) in the same list, so anything that isn't a directory object ID (a GUID)
// is skipped. A reference that fails to resolve — e.g. a stale entry pointing
// at an object that has since been deleted from the directory — is logged and
// skipped rather than failing the whole list.
func resolveDirectoryRefs(runtime *plugin.Runtime, resource string, ids []any) ([]any, error) {
	res := []any{}
	for _, raw := range ids {
		id, ok := raw.(string)
		if !ok || uuid.Validate(id) != nil {
			continue
		}
		r, err := runtime.NewResource(runtime, resource, map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			log.Warn().Err(err).Str("resource", resource).Str("id", id).
				Msg("ms365: skipping unresolvable conditional access reference")
			continue
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
