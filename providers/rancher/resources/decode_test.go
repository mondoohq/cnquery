// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixture reads a collection fixture and returns its records.
func fixture(t *testing.T, name string) []json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)

	var envelope struct {
		Data []json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &envelope))
	require.NotEmpty(t, envelope.Data)
	return envelope.Data
}

// nestedFixture reads one named collection out of testdata/rbac.json, which
// keeps several small collections in one file.
func nestedFixture(t *testing.T, key string) []json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "rbac.json"))
	require.NoError(t, err)

	var file map[string]struct {
		Data []json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &file))

	collection, ok := file[key]
	require.True(t, ok, "no collection %q in rbac.json", key)
	require.NotEmpty(t, collection.Data)
	return collection.Data
}

func decodeAll[T any](t *testing.T, records []json.RawMessage) []T {
	t.Helper()
	out := make([]T, 0, len(records))
	for _, entry := range records {
		var record T
		require.NoError(t, json.Unmarshal(entry, &record))
		out = append(out, record)
	}
	return out
}

func TestClusterDecode(t *testing.T) {
	clusters := decodeAll[clusterRecord](t, fixture(t, "clusters.json"))
	require.Len(t, clusters, 3)

	local := clusters[0]
	assert.Equal(t, "local", local.ID)
	// Norman moves spec.displayName onto name and metadata.name onto id, so a
	// cluster's name is its display name and never its Kubernetes name.
	assert.Equal(t, "local", local.Name)
	assert.Equal(t, "active", local.State)
	assert.Equal(t, "imported", local.Driver)
	assert.Equal(t, "k3s", local.Provider)
	assert.True(t, local.Internal)
	assert.Equal(t, int64(1), local.NodeCount)
	require.NotNil(t, local.Version)
	assert.Equal(t, "v1.31.4+k3s1", local.Version.GitVersion)
	// A cluster that does not carry the flag must stay absent rather than
	// decoding to false, which would report network isolation as deliberately
	// turned off.
	assert.Nil(t, local.EnableNetworkPolicy)
	require.NotNil(t, local.LocalClusterAuthEndpoint)
	assert.False(t, local.LocalClusterAuthEndpoint.Enabled)

	prod := clusters[1]
	assert.Equal(t, "c-m-8xz2nvqp", prod.ID)
	assert.Equal(t, "prod-eu", prod.Name)
	require.NotNil(t, prod.EnableNetworkPolicy)
	assert.True(t, *prod.EnableNetworkPolicy)
	assert.True(t, prod.AppliedEnableNetworkPolicy)
	require.NotNil(t, prod.LocalClusterAuthEndpoint)
	assert.True(t, prod.LocalClusterAuthEndpoint.Enabled)
	assert.Equal(t, "prod-eu.k8s.example.com", prod.LocalClusterAuthEndpoint.FQDN)
	// This field keeps its Name suffix: it is not a Norman reference, so the
	// rename that turns clusterName into clusterId does not apply to it.
	assert.Equal(t, "rancher-restricted", prod.PodSecurityTemplateName)
	assert.Equal(t, "cluster-member", prod.DefaultClusterRoleForProjectMembers)

	provisioning := clusters[2]
	assert.Nil(t, provisioning.Version, "a cluster without a version must not report one")
	assert.Equal(t, "yes", provisioning.Transitioning)
}

func TestTokenDecode(t *testing.T) {
	tokens := decodeAll[tokenRecord](t, fixture(t, "tokens.json"))
	require.Len(t, tokens, 3)

	apiKey := tokens[0]
	assert.Equal(t, "token-nx7pd", apiKey.ID)
	// The wire name is ttl, not ttlMillis. A tag matching the Go field name
	// would decode to zero and report every token as never expiring.
	assert.Equal(t, int64(0), apiKey.TTLMillis)
	assert.True(t, neverExpires(apiKey.TTLMillis))
	assert.False(t, apiKey.IsDerived)
	// userPrincipal is a reference string on the wire, not the Principal
	// object the stored type declares.
	assert.Equal(t, "local://u-b4qkhsnliz", apiKey.UserPrincipalID)
	require.NotNil(t, apiKey.Enabled)
	assert.True(t, *apiKey.Enabled)
	// An absent expiry is an empty string, and must stay null rather than
	// becoming the zero date.
	assert.Nil(t, parseTime(apiKey.ExpiresAt))
	require.NotNil(t, parseTime(apiKey.LastUsedAt))

	session := tokens[1]
	assert.Equal(t, int64(57600000), session.TTLMillis)
	assert.False(t, neverExpires(session.TTLMillis))
	assert.True(t, session.Current)
	require.NotNil(t, parseTime(session.ActivityLastSeenAt))

	kubeconfig := tokens[2]
	assert.Equal(t, "c-m-8xz2nvqp", kubeconfig.ClusterID)
	assert.True(t, kubeconfig.Expired)
	// This token has never been used. The field must stay null, not year one.
	assert.Nil(t, parseTime(kubeconfig.LastUsedAt))
	// enabled is absent, which is not the same as disabled.
	assert.Nil(t, kubeconfig.Enabled)
}

func TestTokenRecordCarriesNoTokenValue(t *testing.T) {
	// The API declares the token value write-only and does not return it, but
	// the guarantee this provider makes is stronger: there is no field to
	// decode one into, so a server that started returning it would still not
	// surface it.
	recordType := reflect.TypeOf(tokenRecord{})
	for i := 0; i < recordType.NumField(); i++ {
		tag := recordType.Field(i).Tag.Get("json")
		assert.NotEqual(t, "token", strings.Split(tag, ",")[0],
			"tokenRecord must not decode the token value")
	}
}

func TestRoleTemplateDecode(t *testing.T) {
	templates := decodeAll[roleTemplateRecord](t, fixture(t, "role_templates.json"))
	require.Len(t, templates, 4)

	owner := templates[0]
	assert.Equal(t, "cluster-owner", owner.ID)
	assert.Equal(t, "Cluster Owner", owner.Name)
	assert.Equal(t, "cluster", owner.Context)
	assert.True(t, owner.Builtin)
	assert.True(t, owner.ClusterCreatorDefault)
	assert.True(t, grantsFullAdmin(effectiveRules(&owner)))

	member := templates[1]
	assert.False(t, grantsFullAdmin(effectiveRules(&member)))
	assert.False(t, grantsPrivilegeEscalation(effectiveRules(&member)))
	// roleTemplateNames is renamed to roleTemplateIds by the reference mapper.
	assert.Equal(t, []string{"create-ns"}, member.RoleTemplateIDs)

	escalate := templates[2]
	assert.False(t, grantsFullAdmin(effectiveRules(&escalate)))
	assert.True(t, grantsPrivilegeEscalation(effectiveRules(&escalate)))
	// The deprecated administrative marker says nothing about this template,
	// which is precisely why the derived predicates exist.
	assert.False(t, escalate.Administrative)
	assert.Equal(t, []string{"cluster-member"}, escalate.RoleTemplateIDs)

	external := templates[3]
	assert.True(t, external.External)
	assert.Empty(t, external.Rules)
	// Reading rules alone would report an external template as granting
	// nothing at all.
	assert.False(t, grantsFullAdmin(external.Rules))
	assert.True(t, grantsFullAdmin(effectiveRules(&external)))
}

func TestPodSecurityTemplateDecode(t *testing.T) {
	templates := decodeAll[podSecurityTemplateRecord](t, fixture(t, "pod_security_templates.json"))
	require.Len(t, templates, 2)

	restricted := templates[0]
	require.NotNil(t, restricted.Configuration)
	require.NotNil(t, restricted.Configuration.Defaults)
	assert.Equal(t, "restricted", restricted.Configuration.Defaults.Enforce)
	// The version keys are hyphenated in the upstream admission configuration.
	// A camel-cased tag decodes to an empty string and reports an unpinned
	// policy version on a template that pins one.
	assert.Equal(t, "v1.31", restricted.Configuration.Defaults.EnforceVersion)
	assert.Equal(t, "latest", restricted.Configuration.Defaults.AuditVersion)
	assert.Equal(t, "latest", restricted.Configuration.Defaults.WarnVersion)
	require.NotNil(t, restricted.Configuration.Exemptions)
	assert.Equal(t, []string{"kube-system", "cattle-system"}, restricted.Configuration.Exemptions.Namespaces)

	privileged := templates[1]
	require.NotNil(t, privileged.Configuration)
	require.NotNil(t, privileged.Configuration.Defaults)
	assert.Equal(t, "privileged", privileged.Configuration.Defaults.Enforce)
	// No version is pinned here. It must read as absent, not as a version.
	assert.Nil(t, nilIfEmpty(privileged.Configuration.Defaults.EnforceVersion))
}

func TestGlobalRoleDecode(t *testing.T) {
	roles := decodeAll[globalRoleRecord](t, nestedFixture(t, "globalRoles"))
	require.Len(t, roles, 2)

	admin := roles[0]
	assert.Equal(t, "admin", admin.ID)
	assert.Equal(t, "Administrator", admin.Name)
	assert.True(t, grantsFullAdmin(admin.Rules))
	assert.True(t, grantsPrivilegeEscalation(admin.Rules))

	standard := roles[1]
	assert.True(t, standard.NewUserDefault)
	assert.False(t, grantsFullAdmin(standard.Rules))
	assert.Equal(t, []string{"cluster-member"}, standard.InheritedClusterRoles)
	require.Contains(t, standard.NamespacedRules, "cattle-system")
}

func TestGlobalRoleBindingDecode(t *testing.T) {
	bindings := decodeAll[globalRoleBindingRecord](t, nestedFixture(t, "globalRoleBindings"))
	require.Len(t, bindings, 2)

	userBinding := bindings[0]
	// globalRoleName is renamed to globalRoleId, and userName to userId.
	assert.Equal(t, "admin", userBinding.GlobalRoleID)
	assert.Equal(t, "u-b4qkhsnliz", userBinding.UserID)
	assert.Equal(t, "user", subjectKind(userBinding.UserID, userBinding.UserPrincipalID, "", userBinding.GroupPrincipalID, ""))
	assert.Equal(t, "u-b4qkhsnliz", subjectName(userBinding.UserID, userBinding.UserPrincipalID, "", userBinding.GroupPrincipalID, ""))

	groupBinding := bindings[1]
	assert.Empty(t, groupBinding.UserID)
	assert.Equal(t, "group", subjectKind(groupBinding.UserID, groupBinding.UserPrincipalID, "", groupBinding.GroupPrincipalID, ""))
	assert.Contains(t, subjectName(groupBinding.UserID, groupBinding.UserPrincipalID, "", groupBinding.GroupPrincipalID, ""), "Platform Admins")
}

func TestBindingDecode(t *testing.T) {
	clusterBindings := decodeAll[bindingRecord](t, nestedFixture(t, "clusterRoleTemplateBindings"))
	require.Len(t, clusterBindings, 1)
	assert.Equal(t, "c-m-8xz2nvqp", clusterBindings[0].ClusterID)
	assert.Equal(t, "cluster-owner", clusterBindings[0].RoleTemplateID)

	projectBindings := decodeAll[bindingRecord](t, nestedFixture(t, "projectRoleTemplateBindings"))
	require.Len(t, projectBindings, 1)
	// A project binding's projectId carries the cluster, which is what makes
	// the binding's scope readable without a second lookup.
	assert.Equal(t, "c-m-8xz2nvqp:p-4xktw", projectBindings[0].ProjectID)
	assert.Equal(t, "group", subjectKind("", "", projectBindings[0].GroupID, projectBindings[0].GroupPrincipalID, ""))
}

func TestProjectDecode(t *testing.T) {
	projects := decodeAll[projectRecord](t, nestedFixture(t, "projects"))
	require.Len(t, projects, 1)

	project := projects[0]
	assert.Equal(t, "c-m-8xz2nvqp:p-4xktw", project.ID)
	assert.Equal(t, "Default", project.Name)
	// clusterName is renamed to clusterId.
	assert.Equal(t, "c-m-8xz2nvqp", project.ClusterID)
	assert.Equal(t, "c-m-8xz2nvqp-p-4xktw", project.BackingNamespace)
	require.NotNil(t, project.ResourceQuota)
	assert.Nil(t, project.ContainerDefaultResourceLimit,
		"an unset limit range must stay absent rather than becoming an empty map")
}

func TestUserDecode(t *testing.T) {
	users := decodeAll[userRecord](t, nestedFixture(t, "users"))
	require.Len(t, users, 2)

	person := users[0]
	assert.Equal(t, "admin", person.Username)
	assert.Equal(t, "Ada Lovelace", person.Name)
	require.NotNil(t, person.Enabled)
	assert.True(t, *person.Enabled)
	assert.False(t, isSystemUser(person.PrincipalIDs))

	agent := users[1]
	assert.Nil(t, agent.Enabled, "an absent enabled flag must not decode to disabled")
	assert.True(t, isSystemUser(agent.PrincipalIDs))
}

func TestUserRecordCarriesNoPassword(t *testing.T) {
	recordType := reflect.TypeOf(userRecord{})
	for i := 0; i < recordType.NumField(); i++ {
		tag := strings.Split(recordType.Field(i).Tag.Get("json"), ",")[0]
		assert.NotEqual(t, "password", tag, "userRecord must not decode a password")
	}
}

func TestSettingDecode(t *testing.T) {
	settings := decodeAll[settingRecord](t, nestedFixture(t, "settings"))
	require.Len(t, settings, 4)

	byName := map[string]settingRecord{}
	for _, setting := range settings {
		byName[setting.Name] = setting
	}

	assert.Equal(t, "https://rancher.example.com", byName["server-url"].Value)
	assert.True(t, byName["server-url"].Customized)
	assert.Equal(t, "v2.11.2", byName["server-version"].Value)

	enforcement, ok := parseBoolSetting(byName["cluster-template-enforcement"].Value)
	assert.True(t, ok)
	assert.True(t, enforcement)

	// A setting with no value in force falls back to the shipped default,
	// which is what the server itself acts on.
	assert.Empty(t, byName["password-min-length"].Value)
	assert.Equal(t, "12", byName["password-min-length"].Default)
}

func TestAuthConfigDecode(t *testing.T) {
	configs := decodeAll[authConfigRecord](t, nestedFixture(t, "authConfigs"))
	require.Len(t, configs, 3)

	local := configs[0]
	assert.Equal(t, localAuthConfigType, local.Type)
	assert.True(t, local.Enabled, "local auth stays a way in alongside an external provider")

	activeDirectory := configs[1]
	assert.Equal(t, "activeDirectoryConfig", activeDirectory.Type)
	assert.True(t, activeDirectory.Enabled)
	assert.Equal(t, "restricted", activeDirectory.AccessMode)
	require.Len(t, activeDirectory.AllowedPrincipalIDs, 1)

	github := configs[2]
	assert.False(t, github.Enabled, "a configured but disabled provider must not read as enabled")
}

func TestRegistryCredentialDropsSecrets(t *testing.T) {
	records := fixture(t, "docker_credentials.json")
	credentials := decodeAll[dockerCredentialRecord](t, records)
	require.Len(t, credentials, 2)

	projectCredential := credentials[0]
	assert.Equal(t, "harbor-pull", projectCredential.Name)
	assert.Empty(t, projectCredential.NamespaceID, "a project credential names no namespace")
	require.Contains(t, projectCredential.Registries, "harbor.example.com")
	assert.Equal(t, "robot$ci", projectCredential.Registries["harbor.example.com"].Username)

	namespaced := credentials[1]
	assert.Equal(t, "team-a", namespaced.NamespaceID)

	// The fixture carries the password, auth and email fields the real API
	// declares. Re-marshaling what was decoded is what proves none of them
	// survived: a field added to the record later would fail this immediately.
	roundTripped, err := json.Marshal(credentials)
	require.NoError(t, err)
	for _, secret := range []string{
		"fixture-password-must-not-appear",
		"fixture-auth-must-not-appear",
		"fixture-email-must-not-appear",
		"password",
		"auth",
	} {
		assert.NotContains(t, string(roundTripped), secret,
			"a registry credential secret survived decoding")
	}
}

func TestNoRecordTypeDecodesACredentialField(t *testing.T) {
	// A sweep over every wire record this provider decodes. Rancher returns
	// credential-shaped material on several of these objects; none of it may
	// have a field to land in.
	forbidden := map[string]bool{
		"token":                  true,
		"password":               true,
		"secret":                 true,
		"auth":                   true,
		"clientsecret":           true,
		"privatekey":             true,
		"serviceaccounttoken":    true,
		"serviceaccountpassword": true,
		"spkey":                  true,
		"cacert":                 true,
		"cacerts":                true,
		"kubeconfig":             true,
	}

	records := []any{
		settingRecord{}, clusterRecord{}, projectRecord{}, globalRoleRecord{},
		globalRoleBindingRecord{}, roleTemplateRecord{}, bindingRecord{},
		clusterTemplateRecord{}, clusterTemplateRevisionRecord{},
		podSecurityTemplateRecord{}, authConfigRecord{}, tokenRecord{},
		userRecord{}, dockerCredentialRecord{}, registryAccount{},
		versionInfo{}, localClusterAuthEndpoint{},
	}

	for _, record := range records {
		recordType := reflect.TypeOf(record)
		for i := 0; i < recordType.NumField(); i++ {
			tag := strings.ToLower(strings.Split(recordType.Field(i).Tag.Get("json"), ",")[0])
			assert.False(t, forbidden[tag],
				"%s decodes a credential field %q", recordType.Name(), tag)
		}
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"rfc3339", "2026-08-19T06:00:11Z", true},
		{"rfc3339 with fraction", "2026-08-19T06:00:11.123456Z", true},
		{"rfc3339 with offset", "2026-08-19T08:00:11+02:00", true},
		{"empty", "", false},
		{"blank", "   ", false},
		{"not a time", "never", false},
		{"kubernetes null literal", "null", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := parseTime(test.value)
			if !test.want {
				assert.Nil(t, got, "an unreadable timestamp must stay null, never year one")
				return
			}
			require.NotNil(t, got)
			assert.False(t, got.IsZero())
			assert.True(t, got.After(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)))
		})
	}
}

func TestNeverExpires(t *testing.T) {
	assert.True(t, neverExpires(0), "a zero lifetime is Rancher's way of saying no expiry")
	assert.True(t, neverExpires(-1), "a negative lifetime is not a lifetime either")
	assert.False(t, neverExpires(1))
	assert.False(t, neverExpires(2592000000))
}

func TestGrantsFullAdmin(t *testing.T) {
	tests := []struct {
		name  string
		rules []policyRule
		want  bool
	}{
		{"no rules", nil, false},
		{"empty rule", []policyRule{{}}, false},
		{
			"wildcard everything",
			[]policyRule{{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"*"}}},
			true,
		},
		{
			"wildcard verbs on one resource",
			[]policyRule{{APIGroups: []string{"apps"}, Resources: []string{"deployments"}, Verbs: []string{"*"}}},
			false,
		},
		{
			"wildcard resources without wildcard verbs",
			[]policyRule{{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"get", "list"}}},
			false,
		},
		{
			"wildcard split across two rules is not full admin",
			[]policyRule{
				{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"get"}},
				{APIGroups: []string{"apps"}, Resources: []string{"deployments"}, Verbs: []string{"*"}},
			},
			false,
		},
		{
			"non-resource urls only",
			[]policyRule{{NonResourceURLs: []string{"*"}, Verbs: []string{"*"}}},
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, grantsFullAdmin(test.rules))
		})
	}
}

func TestGrantsPrivilegeEscalation(t *testing.T) {
	tests := []struct {
		name  string
		rules []policyRule
		want  bool
	}{
		{"no rules", nil, false},
		{
			"escalate verb",
			[]policyRule{{APIGroups: []string{"rbac.authorization.k8s.io"}, Resources: []string{"roles"}, Verbs: []string{"escalate"}}},
			true,
		},
		{
			"bind verb",
			[]policyRule{{Resources: []string{"clusterroles"}, Verbs: []string{"bind"}}},
			true,
		},
		{
			"impersonate verb, mixed case",
			[]policyRule{{Resources: []string{"users"}, Verbs: []string{"Impersonate"}}},
			true,
		},
		{
			"wildcard verbs on role bindings",
			[]policyRule{{APIGroups: []string{"rbac.authorization.k8s.io"}, Resources: []string{"rolebindings"}, Verbs: []string{"*"}}},
			true,
		},
		{
			"wildcard verbs on rancher role templates",
			[]policyRule{{APIGroups: []string{"management.cattle.io"}, Resources: []string{"roleTemplates"}, Verbs: []string{"*"}}},
			true,
		},
		{
			"wildcard verbs on an unrelated resource",
			[]policyRule{{APIGroups: []string{"apps"}, Resources: []string{"deployments"}, Verbs: []string{"*"}}},
			false,
		},
		{
			"read access to roles is not escalation",
			[]policyRule{{APIGroups: []string{"rbac.authorization.k8s.io"}, Resources: []string{"roles"}, Verbs: []string{"get", "list"}}},
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, grantsPrivilegeEscalation(test.rules))
		})
	}
}

func TestSubjectKindAndName(t *testing.T) {
	tests := []struct {
		name             string
		userID           string
		userPrincipalID  string
		groupID          string
		groupPrincipalID string
		serviceAccount   string
		wantKind         string
		wantName         string
	}{
		{"local user", "u-1", "local://u-1", "", "", "", "user", "u-1"},
		{"external user with no local account", "", "github_user://42", "", "", "", "user", "github_user://42"},
		{"group principal", "", "", "", "ad_group://CN=Ops", "", "group", "ad_group://CN=Ops"},
		{"plain group", "", "", "ops", "", "", "group", "ops"},
		{"service account", "", "", "", "", "ns:builder", "serviceAccount", "ns:builder"},
		{"nothing named", "", "", "", "", "", "unknown", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.wantKind,
				subjectKind(test.userID, test.userPrincipalID, test.groupID, test.groupPrincipalID, test.serviceAccount))
			assert.Equal(t, test.wantName,
				subjectName(test.userID, test.userPrincipalID, test.groupID, test.groupPrincipalID, test.serviceAccount))
		})
	}
}

func TestIsSystemUser(t *testing.T) {
	assert.True(t, isSystemUser([]string{"system://provisioning"}))
	assert.True(t, isSystemUser([]string{"local://u-1", "system://agent"}))
	assert.False(t, isSystemUser([]string{"local://u-1"}))
	assert.False(t, isSystemUser(nil))
	// A principal that merely mentions the word is not a system account.
	assert.False(t, isSystemUser([]string{"activedirectory_user://CN=system.admin"}))
}

func TestNilIfEmpty(t *testing.T) {
	assert.Nil(t, nilIfEmpty(""))
	require.NotNil(t, nilIfEmpty("restricted"))
	assert.Equal(t, "restricted", *nilIfEmpty("restricted"))
}

func TestParseBoolSetting(t *testing.T) {
	tests := []struct {
		value string
		want  bool
		ok    bool
	}{
		{"true", true, true},
		{"false", false, true},
		{"True", true, true},
		{"1", true, true},
		{"0", false, true},
		{"", false, false},
		{"yes", false, false},
		{"enabled", false, false},
	}

	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			got, ok := parseBoolSetting(test.value)
			assert.Equal(t, test.ok, ok)
			assert.Equal(t, test.want, got)
		})
	}
}
