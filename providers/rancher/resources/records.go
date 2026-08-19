// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"time"
)

// The structs below mirror what the Rancher Manager v3 API puts on the wire.
//
// That shape is not the shape of the Kubernetes objects behind it. Rancher
// serves the management API through Norman, which flattens metadata, spec and
// status into one object, renames creationTimestamp to created and uid to uuid,
// moves a resource's displayName onto name and its Kubernetes name onto id, and
// rewrites every reference field ending in Name to end in Id. So a cluster's
// clusterName arrives as clusterId, a binding's roleTemplateName as
// roleTemplateId, and a role template's roleTemplateNames as roleTemplateIds.
// Fields carrying a password are declared write-only in the schema and are not
// returned at all; none of them are modeled here either way.
//
// Timestamps arrive as RFC 3339 strings, and an absent one is an empty string
// rather than a null, which is why every timestamp goes through parseTime and
// stays null instead of becoming the zero date.

// settingRecord is one entry of /v3/settings.
type settingRecord struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Value      string `json:"value"`
	Default    string `json:"default"`
	Customized bool   `json:"customized"`
	Source     string `json:"source"`
}

// clusterRecord is one entry of /v3/clusters.
type clusterRecord struct {
	ID                   string            `json:"id"`
	Name                 string            `json:"name"`
	Description          string            `json:"description"`
	State                string            `json:"state"`
	Transitioning        string            `json:"transitioning"`
	TransitioningMessage string            `json:"transitioningMessage"`
	Driver               string            `json:"driver"`
	Provider             string            `json:"provider"`
	Internal             bool              `json:"internal"`
	Created              string            `json:"created"`
	NodeCount            int64             `json:"nodeCount"`
	FleetWorkspaceName   string            `json:"fleetWorkspaceName"`
	APIEndpoint          string            `json:"apiEndpoint"`
	Labels               map[string]string `json:"labels"`
	Annotations          map[string]string `json:"annotations"`

	// EnableNetworkPolicy is a pointer in the Rancher API and is genuinely
	// absent on clusters where Rancher does not manage the network plugin.
	EnableNetworkPolicy        *bool `json:"enableNetworkPolicy"`
	AppliedEnableNetworkPolicy bool  `json:"appliedEnableNetworkPolicy"`

	Version                  *versionInfo              `json:"version"`
	LocalClusterAuthEndpoint *localClusterAuthEndpoint `json:"localClusterAuthEndpoint"`

	// DefaultPodSecurityAdmissionConfigurationTemplateName is not a Norman
	// reference field, so it keeps its Name suffix on the wire.
	PodSecurityTemplateName string `json:"defaultPodSecurityAdmissionConfigurationTemplateName"`
	// DefaultClusterRoleForProjectMembers is a reference but does not end in
	// Name, so it is not rewritten either.
	DefaultClusterRoleForProjectMembers string `json:"defaultClusterRoleForProjectMembers"`
}

type versionInfo struct {
	GitVersion string `json:"gitVersion"`
	Major      string `json:"major"`
	Minor      string `json:"minor"`
}

type localClusterAuthEndpoint struct {
	Enabled bool   `json:"enabled"`
	FQDN    string `json:"fqdn"`
}

// projectRecord is one entry of /v3/projects.
type projectRecord struct {
	ID                            string            `json:"id"`
	Name                          string            `json:"name"`
	Description                   string            `json:"description"`
	State                         string            `json:"state"`
	Created                       string            `json:"created"`
	BackingNamespace              string            `json:"backingNamespace"`
	ClusterID                     string            `json:"clusterId"`
	Labels                        map[string]string `json:"labels"`
	Annotations                   map[string]string `json:"annotations"`
	ResourceQuota                 map[string]any    `json:"resourceQuota"`
	ContainerDefaultResourceLimit map[string]any    `json:"containerDefaultResourceLimit"`
}

// policyRule is one Kubernetes RBAC rule as the management API reports it.
type policyRule struct {
	APIGroups       []string `json:"apiGroups"`
	Resources       []string `json:"resources"`
	ResourceNames   []string `json:"resourceNames"`
	NonResourceURLs []string `json:"nonResourceURLs"`
	Verbs           []string `json:"verbs"`
}

// globalRoleRecord is one entry of /v3/globalRoles.
type globalRoleRecord struct {
	ID                       string                  `json:"id"`
	Name                     string                  `json:"name"`
	Description              string                  `json:"description"`
	Builtin                  bool                    `json:"builtin"`
	NewUserDefault           bool                    `json:"newUserDefault"`
	Created                  string                  `json:"created"`
	Rules                    []policyRule            `json:"rules"`
	NamespacedRules          map[string][]policyRule `json:"namespacedRules"`
	InheritedNamespacedRules map[string][]policyRule `json:"inheritedNamespacedRules"`
	InheritedClusterRoles    []string                `json:"inheritedClusterRoles"`
}

// globalRoleBindingRecord is one entry of /v3/globalRoleBindings. The subject
// fields carry the Norman rewrite: userName became userId, globalRoleName
// became globalRoleId, and the principal fields gained the same suffix.
type globalRoleBindingRecord struct {
	ID               string `json:"id"`
	Created          string `json:"created"`
	UserID           string `json:"userId"`
	UserPrincipalID  string `json:"userPrincipalId"`
	GroupPrincipalID string `json:"groupPrincipalId"`
	GlobalRoleID     string `json:"globalRoleId"`
}

// roleTemplateRecord is one entry of /v3/roleTemplates.
type roleTemplateRecord struct {
	ID                    string       `json:"id"`
	Name                  string       `json:"name"`
	Description           string       `json:"description"`
	Context               string       `json:"context"`
	Builtin               bool         `json:"builtin"`
	External              bool         `json:"external"`
	Hidden                bool         `json:"hidden"`
	Locked                bool         `json:"locked"`
	ClusterCreatorDefault bool         `json:"clusterCreatorDefault"`
	ProjectCreatorDefault bool         `json:"projectCreatorDefault"`
	Administrative        bool         `json:"administrative"`
	Created               string       `json:"created"`
	Rules                 []policyRule `json:"rules"`
	ExternalRules         []policyRule `json:"externalRules"`
	// RoleTemplateIDs is roleTemplateNames after the Norman rewrite.
	RoleTemplateIDs []string `json:"roleTemplateIds"`
}

// bindingRecord covers both /v3/clusterRoleTemplateBindings and
// /v3/projectRoleTemplateBindings, which differ only in whether the scope is a
// cluster or a project.
type bindingRecord struct {
	ID               string `json:"id"`
	Created          string `json:"created"`
	UserID           string `json:"userId"`
	UserPrincipalID  string `json:"userPrincipalId"`
	GroupID          string `json:"groupId"`
	GroupPrincipalID string `json:"groupPrincipalId"`
	RoleTemplateID   string `json:"roleTemplateId"`
	ClusterID        string `json:"clusterId"`
	ProjectID        string `json:"projectId"`
	ServiceAccount   string `json:"serviceAccount"`
}

// clusterTemplateRecord is one entry of /v3/clusterTemplates, which exists on
// Rancher 2.11 and older only.
type clusterTemplateRecord struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	Description       string           `json:"description"`
	Created           string           `json:"created"`
	DefaultRevisionID string           `json:"defaultRevisionId"`
	Members           []map[string]any `json:"members"`
}

// clusterTemplateRevisionRecord is one entry of /v3/clusterTemplateRevisions.
// The revision's clusterConfig is deliberately not decoded: it carries the
// whole cluster specification, including credential-bearing sections, and none
// of it is modeled.
type clusterTemplateRevisionRecord struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	Created           string           `json:"created"`
	State             string           `json:"state"`
	Enabled           *bool            `json:"enabled"`
	ClusterTemplateID string           `json:"clusterTemplateId"`
	Questions         []map[string]any `json:"questions"`
}

// podSecurityTemplateRecord is one entry of
// /v3/podSecurityAdmissionConfigurationTemplates.
type podSecurityTemplateRecord struct {
	ID            string                    `json:"id"`
	Name          string                    `json:"name"`
	Description   string                    `json:"description"`
	Created       string                    `json:"created"`
	Configuration *podSecurityConfiguration `json:"configuration"`
}

type podSecurityConfiguration struct {
	Defaults   *podSecurityDefaults   `json:"defaults"`
	Exemptions *podSecurityExemptions `json:"exemptions"`
}

// podSecurityDefaults mirrors the upstream Kubernetes admission configuration,
// which spells the version keys with a hyphen. A camel-cased tag here would
// decode to an empty string and report an unpinned policy version on a template
// that pins one.
type podSecurityDefaults struct {
	Enforce        string `json:"enforce"`
	EnforceVersion string `json:"enforce-version"`
	Audit          string `json:"audit"`
	AuditVersion   string `json:"audit-version"`
	Warn           string `json:"warn"`
	WarnVersion    string `json:"warn-version"`
}

type podSecurityExemptions struct {
	Namespaces     []string `json:"namespaces"`
	Usernames      []string `json:"usernames"`
	RuntimeClasses []string `json:"runtimeClasses"`
}

// authConfigRecord is one entry of /v3/authConfigs. Every provider, configured
// or not, is listed; type is the discriminator and enabled says which of them
// currently accept logins.
type authConfigRecord struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Type                string   `json:"type"`
	Enabled             bool     `json:"enabled"`
	AccessMode          string   `json:"accessMode"`
	AllowedPrincipalIDs []string `json:"allowedPrincipalIds"`
	LogoutAllSupported  bool     `json:"logoutAllSupported"`
}

// tokenRecord is one entry of /v3/tokens.
//
// TTLMillis is spelled ttl on the wire, not ttlMillis, and the timestamps are
// strings rather than the Kubernetes time objects the stored type uses.
type tokenRecord struct {
	ID                 string `json:"id"`
	UserID             string `json:"userId"`
	UserPrincipalID    string `json:"userPrincipal"`
	Description        string `json:"description"`
	AuthProvider       string `json:"authProvider"`
	IsDerived          bool   `json:"isDerived"`
	Enabled            *bool  `json:"enabled"`
	Expired            bool   `json:"expired"`
	Current            bool   `json:"current"`
	TTLMillis          int64  `json:"ttl"`
	ExpiresAt          string `json:"expiresAt"`
	LastUsedAt         string `json:"lastUsedAt"`
	ActivityLastSeenAt string `json:"activityLastSeenAt"`
	Created            string `json:"created"`
	ClusterID          string `json:"clusterId"`
}

// userRecord is one entry of /v3/users. The password field is write-only in the
// schema and is not decoded here in any case.
type userRecord struct {
	ID                 string   `json:"id"`
	Username           string   `json:"username"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	Enabled            *bool    `json:"enabled"`
	MustChangePassword bool     `json:"mustChangePassword"`
	Created            string   `json:"created"`
	PrincipalIDs       []string `json:"principalIds"`
}

// dockerCredentialRecord is one entry of a project's dockerCredentials or
// namespacedDockerCredentials collection.
//
// The registries map is decoded into registryAccount, which carries the user
// name only. The API's own shape also has password, auth and email; leaving
// them off the struct is what keeps a registry password from ever entering the
// process, let alone a query result.
type dockerCredentialRecord struct {
	ID          string                     `json:"id"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Created     string                     `json:"created"`
	ProjectID   string                     `json:"projectId"`
	NamespaceID string                     `json:"namespaceId"`
	Registries  map[string]registryAccount `json:"registries"`
}

type registryAccount struct {
	Username string `json:"username"`
}

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
