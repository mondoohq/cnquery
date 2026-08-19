// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"time"
)

// Everything in this file is computed from a decoded record rather than read
// off the wire: timestamp and empty-value handling, the RBAC predicates the
// schema exposes as derived fields, and the conversions that turn a record into
// the dicts and maps MQL carries. None of it has a counterpart in Rancher's
// generated client, which is why it lives apart from the wire types in
// records.go.

// parseTime turns an API timestamp into a time, and anything unusable into
// null. An absent timestamp must not become the zero date: reported as a real
// value it would place a token's last use in the year one and satisfy any
// "used recently" comparison written against it.
func parseTime(value string) *time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05Z0700"} {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return &parsed
		}
	}
	return nil
}

// nilIfEmpty reports an unset string as null rather than as an empty value.
// An admission template that pins no enforce level and one that pins the empty
// string are the same thing, and neither should read as a level.
func nilIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// wildcard is the RBAC token that matches everything in its position.
const wildcard = "*"

// escalatingVerbs are the verbs that let a principal grant itself, or somebody
// else, permissions it does not already hold.
var escalatingVerbs = map[string]bool{
	"escalate":    true,
	"bind":        true,
	"impersonate": true,
}

// roleResources are the resources through which permissions are handed out. A
// rule granting every verb on one of them is privilege escalation by another
// route, since the holder can simply write itself a better role.
var roleResources = map[string]bool{
	"roles":                       true,
	"clusterroles":                true,
	"rolebindings":                true,
	"clusterrolebindings":         true,
	"roletemplates":               true,
	"globalroles":                 true,
	"globalrolebindings":          true,
	"clusterroletemplatebindings": true,
	"projectroletemplatebindings": true,
}

// grantsFullAdmin reports whether any rule grants every verb on every resource
// in every API group, which is administration of everything the rules reach
// whatever the role is called.
func grantsFullAdmin(rules []policyRule) bool {
	for _, rule := range rules {
		if contains(rule.Verbs, wildcard) &&
			contains(rule.Resources, wildcard) &&
			contains(rule.APIGroups, wildcard) {
			return true
		}
	}
	return false
}

// grantsPrivilegeEscalation reports whether any rule lets the holder raise its
// own or another principal's permissions, either through a verb that does so
// directly or through unrestricted write access to the objects that grant
// permissions.
func grantsPrivilegeEscalation(rules []policyRule) bool {
	for _, rule := range rules {
		for _, verb := range rule.Verbs {
			if escalatingVerbs[strings.ToLower(verb)] {
				return true
			}
		}
		if !contains(rule.Verbs, wildcard) {
			continue
		}
		for _, resource := range rule.Resources {
			lowered := strings.ToLower(resource)
			if lowered == wildcard || roleResources[lowered] {
				return true
			}
		}
	}
	return false
}

// effectiveRules picks the rule set a role template is actually evaluated
// against. An external template takes its permissions from a cluster role in
// the local cluster, and reports them in externalRules when it reports them at
// all, so reading rules alone would report an external template as granting
// nothing.
func effectiveRules(record *roleTemplateRecord) []policyRule {
	if record.External && len(record.ExternalRules) > 0 {
		return record.ExternalRules
	}
	return record.Rules
}

// isSystemUser reports whether the account is one Rancher created for its own
// components rather than for a person.
func isSystemUser(principalIDs []string) bool {
	for _, id := range principalIDs {
		if strings.HasPrefix(id, "system:") {
			return true
		}
	}
	return false
}

// neverExpires reports whether a token was issued without a lifetime. Rancher
// writes a time-to-live of zero for a token that never expires; a negative
// value is not a lifetime either.
func neverExpires(ttlMillis int64) bool {
	return ttlMillis <= 0
}

// subjectKind and subjectName describe who a binding grants access to. Rancher
// spreads the subject over four optional fields, and which one is populated
// depends on whether the subject is a local account, an external identity, or a
// group that only the identity provider can enumerate.
func subjectKind(userID, userPrincipalID, groupID, groupPrincipalID, serviceAccount string) string {
	switch {
	case userID != "" || userPrincipalID != "":
		return "user"
	case groupID != "" || groupPrincipalID != "":
		return "group"
	case serviceAccount != "":
		return "serviceAccount"
	default:
		return "unknown"
	}
}

func subjectName(userID, userPrincipalID, groupID, groupPrincipalID, serviceAccount string) string {
	switch {
	case userID != "":
		return userID
	case userPrincipalID != "":
		return userPrincipalID
	case groupPrincipalID != "":
		return groupPrincipalID
	case groupID != "":
		return groupID
	default:
		return serviceAccount
	}
}

// rulesToDicts turns policy rules into the plain maps the dict fields carry.
func rulesToDicts(rules []policyRule) []any {
	out := make([]any, 0, len(rules))
	for _, rule := range rules {
		out = append(out, map[string]any{
			"apiGroups":       toAnySlice(rule.APIGroups),
			"resources":       toAnySlice(rule.Resources),
			"resourceNames":   toAnySlice(rule.ResourceNames),
			"nonResourceURLs": toAnySlice(rule.NonResourceURLs),
			"verbs":           toAnySlice(rule.Verbs),
		})
	}
	return out
}

// namespacedRulesToDict turns a namespace-keyed rule map into a dict.
func namespacedRulesToDict(rules map[string][]policyRule) map[string]any {
	if rules == nil {
		return nil
	}
	out := make(map[string]any, len(rules))
	for namespace, set := range rules {
		out[namespace] = rulesToDicts(set)
	}
	return out
}

func toAnySlice(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func toStringMap(values map[string]string) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
