// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/types"
	"go.mondoo.com/mql/v13/utils/stringx"
	rbacv1 "k8s.io/api/rbac/v1"
)

// rbacPolicyRules converts the typed PolicyRule slice of a Role or ClusterRole
// into k8s.rbac.policyRule resources. parentID is the owning role's id and is
// used to build a stable, unique __id per rule.
func rbacPolicyRules(runtime *plugin.Runtime, parentID string, rules []rbacv1.PolicyRule) ([]any, error) {
	out := make([]any, 0, len(rules))
	for i := range rules {
		rule := rules[i]
		r, err := CreateResource(runtime, "k8s.rbac.policyRule", map[string]*llx.RawData{
			"__id":            llx.StringData(fmt.Sprintf("%s/rule/%d", parentID, i)),
			"verbs":           llx.ArrayData(convert.SliceAnyToInterface(rule.Verbs), types.String),
			"apiGroups":       llx.ArrayData(convert.SliceAnyToInterface(rule.APIGroups), types.String),
			"resources":       llx.ArrayData(convert.SliceAnyToInterface(rule.Resources), types.String),
			"resourceNames":   llx.ArrayData(convert.SliceAnyToInterface(rule.ResourceNames), types.String),
			"nonResourceURLs": llx.ArrayData(convert.SliceAnyToInterface(rule.NonResourceURLs), types.String),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

// rbacListMatches reports whether list contains target or the wildcard "*".
func rbacListMatches(list []string, target string) bool {
	return stringx.Contains(list, "*") || stringx.Contains(list, target)
}

// rbacRuleGrants reports whether the rule grants the given verb on the given
// (apiGroup, resource), honoring wildcards in any of the three dimensions.
func rbacRuleGrants(rule rbacv1.PolicyRule, apiGroup, resource, verb string) bool {
	return rbacListMatches(rule.APIGroups, apiGroup) &&
		rbacListMatches(rule.Resources, resource) &&
		rbacListMatches(rule.Verbs, verb)
}

// rbacHasWildcardRule reports whether any rule uses a wildcard for verbs, API
// groups, or resources.
func rbacHasWildcardRule(rules []rbacv1.PolicyRule) bool {
	for i := range rules {
		rule := rules[i]
		if stringx.Contains(rule.Verbs, "*") ||
			stringx.Contains(rule.APIGroups, "*") ||
			stringx.Contains(rule.Resources, "*") {
			return true
		}
	}
	return false
}

// rbacGrantsClusterAdmin reports whether any rule grants unrestricted access:
// the verb, API group, and resource dimensions all include the wildcard "*".
// This is the rule shape of the built-in cluster-admin ClusterRole and of any
// equivalent custom role.
func rbacGrantsClusterAdmin(rules []rbacv1.PolicyRule) bool {
	for i := range rules {
		rule := rules[i]
		if stringx.Contains(rule.Verbs, "*") &&
			stringx.Contains(rule.APIGroups, "*") &&
			stringx.Contains(rule.Resources, "*") {
			return true
		}
	}
	return false
}

// rbacAllowsPrivilegeEscalation reports whether any rule grants one of the
// canonical RBAC privilege-escalation permissions.
func rbacAllowsPrivilegeEscalation(rules []rbacv1.PolicyRule) bool {
	const rbacGroup = "rbac.authorization.k8s.io"
	for i := range rules {
		rule := rules[i]
		// escalate on roles/clusterroles
		if rbacRuleGrants(rule, rbacGroup, "roles", "escalate") ||
			rbacRuleGrants(rule, rbacGroup, "clusterroles", "escalate") {
			return true
		}
		// bind on roles/clusterroles
		if rbacRuleGrants(rule, rbacGroup, "roles", "bind") ||
			rbacRuleGrants(rule, rbacGroup, "clusterroles", "bind") {
			return true
		}
		// impersonate users/groups/serviceaccounts (core group)
		if rbacRuleGrants(rule, "", "users", "impersonate") ||
			rbacRuleGrants(rule, "", "groups", "impersonate") ||
			rbacRuleGrants(rule, "", "serviceaccounts", "impersonate") {
			return true
		}
	}
	return false
}

// rbacCanReadSecrets reports whether any rule grants read access (get, list, or
// watch) to Secrets in the core API group.
func rbacCanReadSecrets(rules []rbacv1.PolicyRule) bool {
	for i := range rules {
		rule := rules[i]
		if rbacRuleGrants(rule, "", "secrets", "get") ||
			rbacRuleGrants(rule, "", "secrets", "list") ||
			rbacRuleGrants(rule, "", "secrets", "watch") {
			return true
		}
	}
	return false
}
