// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file adopts Rancher's generated v3 client as the authority on the wire
// format, without importing it.
//
// Why not import it. github.com/rancher/rancher/pkg/client is its own Go
// module, so taking it does not drag in the rest of rancher/rancher. It was
// still measured and still rejected, for four reasons, in descending order of
// weight:
//
//  1. The generated structs and the Norman client are one Go package. Every
//     zz_generated_*.go file is `package client`, and zz_generated_client.go
//     imports github.com/rancher/norman/clientbase. Go imports packages, not
//     files, so there is no way to take the types and leave the transport.
//
//  2. Three of the structs decode credentials. client.Token carries
//     `Token string json:"token"`, client.User carries
//     `Password string json:"password"`, and client.RegistryCredential carries
//     both `Password` and `Auth` (the pre-encoded registry header). Decoding
//     into them would put a live token value and a registry password in this
//     process. The whole point of the local records is that there is no field
//     for those to land in. client.Cluster and client.ClusterTemplateRevision
//     carry secret-bearing fields by the dozen on top of that.
//
//  3. The module has no tagged version, and its head has deleted types this
//     provider models. Only pseudo-versions off main exist, and both main and
//     the 2.12 release line dropped ClusterTemplate and ClusterTemplateRevision
//     with the rest of RKE1: zz_generated_cluster_template.go and
//     zz_generated_cluster_template_revision.go are absent at v2.12.9, so
//     rancher.clusterTemplate and rancher.clusterTemplate.revision would have
//     no type to decode into. Pinning main breaks those two; pinning a
//     2.11-era commit freezes everything else at 2.11.
//
//  4. Footprint. Importing generated/management/v3 adds 7 modules to
//     providers/rancher/go.mod that are not there today: rancher/norman,
//     rancher/wrangler/v3, k8s.io/apimachinery, gorilla/websocket,
//     sirupsen/logrus, ghodss/yaml and gopkg.in/yaml.v2. Module pruning keeps
//     the compiled surface small (43 non-stdlib packages; k8s.io/apimachinery
//     contributes one, pkg/util/rand) and the provider binary grew only
//     192,404 bytes when this was measured end to end, so the cost is a
//     maintenance cost rather than a size one. On its own it would not decide
//     anything; on top of 1 to 3 it settles it.
//
// What is adopted instead is the generated field inventory. upstreamFields
// below is the complete list of wire names each generated type declares, taken
// verbatim from its `<Type>Field<Name> = "wire"` constants, which is the same
// machine-generated source the struct tags come from. The tests then hold the
// local records against it in both directions: every tag a record decodes has
// to be a name upstream declares, and no name upstream declares that carries a
// credential may appear on a record at all.
//
// Regenerating upstreamFields, should Rancher move something:
//
//	git clone --depth 1 -b v2.12.9 https://github.com/rancher/rancher
//	grep -h '^	<Type>Field' rancher/pkg/client/generated/management/v3/*.go |
//	  sed 's/.*= "//; s/"$//' | sort
//
// with the two cluster template types read from v2.11.6 instead.
//
// Sources, both pinned:
//
//	v2.12.9  db2754edc35189187bb10c524601d3d62642ff9b
//	v2.11.6  8e2eb63d5d2b4744ab3b4ab44de573106519d77d
var upstreamFields = map[string][]string{
	"Setting": {
		"annotations",
		"created",
		"creatorId",
		"customized",
		"default",
		"labels",
		"name",
		"ownerReferences",
		"removed",
		"source",
		"uuid",
		"value",
	},
	"Cluster": {
		"aadClientCertSecret",
		"aadClientSecret",
		"agentEnvVars",
		"agentFeatures",
		"agentImage",
		"agentImageOverride",
		"aksConfig",
		"aksStatus",
		"allocatable",
		"annotations",
		"answers",
		"apiEndpoint",
		"appliedAgentEnvVars",
		"appliedClusterAgentDeploymentCustomization",
		"appliedEnableNetworkPolicy",
		"appliedSpec",
		"authImage",
		"caCert",
		"capabilities",
		"capacity",
		"certificatesExpiration",
		"clusterAgentDeploymentCustomization",
		"clusterSecrets",
		"clusterTemplateId",
		"clusterTemplateRevisionId",
		"componentStatuses",
		"conditions",
		"created",
		"creatorId",
		"currentCisRunName",
		"defaultClusterRoleForProjectMembers",
		"defaultPodSecurityAdmissionConfigurationTemplateName",
		"description",
		"desiredAgentImage",
		"desiredAuthImage",
		"dockerRootDir",
		"driver",
		"eksConfig",
		"eksStatus",
		"enableNetworkPolicy",
		"failedSpec",
		"fleetAgentDeploymentCustomization",
		"fleetWorkspaceName",
		"gkeConfig",
		"gkeStatus",
		"importedConfig",
		"internal",
		"istioEnabled",
		"k3sConfig",
		"labels",
		"limits",
		"linuxWorkerCount",
		"localClusterAuthEndpoint",
		"name",
		"nodeCount",
		"nodeVersion",
		"openStackSecret",
		"ownerReferences",
		"privateRegistrySecret",
		"provider",
		"questions",
		"rancherKubernetesEngineConfig",
		"removed",
		"requested",
		"rke2Config",
		"s3CredentialSecret",
		"serviceAccountTokenSecret",
		"state",
		"transitioning",
		"transitioningMessage",
		"uuid",
		"version",
		"virtualCenterSecret",
		"vsphereSecret",
		"weavePasswordSecret",
		"windowsPreferedCluster",
		"windowsWorkerCount",
	},
	"Info": {
		"buildDate",
		"compiler",
		"emulationMajor",
		"emulationMinor",
		"gitCommit",
		"gitTreeState",
		"gitVersion",
		"goVersion",
		"major",
		"minCompatibilityMajor",
		"minCompatibilityMinor",
		"minor",
		"platform",
	},
	"LocalClusterAuthEndpoint": {
		"caCerts",
		"enabled",
		"fqdn",
	},
	"Project": {
		"annotations",
		"backingNamespace",
		"clusterId",
		"conditions",
		"containerDefaultResourceLimit",
		"created",
		"creatorId",
		"description",
		"labels",
		"name",
		"namespaceDefaultResourceQuota",
		"namespaceId",
		"ownerReferences",
		"removed",
		"resourceQuota",
		"state",
		"transitioning",
		"transitioningMessage",
		"uuid",
	},
	"PolicyRule": {
		"apiGroups",
		"nonResourceURLs",
		"resourceNames",
		"resources",
		"verbs",
	},
	"GlobalRole": {
		"annotations",
		"builtin",
		"created",
		"creatorId",
		"description",
		"inheritedClusterRoles",
		"inheritedFleetWorkspacePermissions",
		"labels",
		"name",
		"namespacedRules",
		"newUserDefault",
		"ownerReferences",
		"removed",
		"rules",
		"status",
		"uuid",
	},
	"GlobalRoleBinding": {
		"annotations",
		"created",
		"creatorId",
		"globalRoleId",
		"groupPrincipalId",
		"labels",
		"name",
		"ownerReferences",
		"removed",
		"status",
		"userId",
		"userPrincipalId",
		"uuid",
	},
	"RoleTemplate": {
		"administrative",
		"annotations",
		"builtin",
		"clusterCreatorDefault",
		"context",
		"created",
		"creatorId",
		"description",
		"external",
		"externalRules",
		"hidden",
		"labels",
		"locked",
		"name",
		"ownerReferences",
		"projectCreatorDefault",
		"removed",
		"roleTemplateIds",
		"rules",
		"uuid",
	},
	"ClusterRoleTemplateBinding": {
		"annotations",
		"clusterId",
		"created",
		"creatorId",
		"groupId",
		"groupPrincipalId",
		"labels",
		"name",
		"namespaceId",
		"ownerReferences",
		"removed",
		"roleTemplateId",
		"status",
		"userId",
		"userPrincipalId",
		"uuid",
	},
	"ProjectRoleTemplateBinding": {
		"annotations",
		"created",
		"creatorId",
		"groupId",
		"groupPrincipalId",
		"labels",
		"name",
		"namespaceId",
		"ownerReferences",
		"projectId",
		"removed",
		"roleTemplateId",
		"serviceAccount",
		"userId",
		"userPrincipalId",
		"uuid",
	},
	"ClusterTemplate": {
		"annotations",
		"created",
		"creatorId",
		"defaultRevisionId",
		"description",
		"labels",
		"members",
		"name",
		"ownerReferences",
		"removed",
		"uuid",
	},
	"ClusterTemplateRevision": {
		"aadClientCertSecret",
		"aadClientSecret",
		"aciAPICUserKeySecret",
		"aciKafkaClientKeySecret",
		"aciTokenSecret",
		"annotations",
		"bastionHostSSHKeySecret",
		"clusterConfig",
		"clusterTemplateId",
		"conditions",
		"created",
		"creatorId",
		"enabled",
		"kubeletExtraEnvSecret",
		"labels",
		"name",
		"openStackSecret",
		"ownerReferences",
		"privateRegistryECRSecret",
		"privateRegistrySecret",
		"questions",
		"removed",
		"s3CredentialSecret",
		"secretsEncryptionProvidersSecret",
		"state",
		"transitioning",
		"transitioningMessage",
		"uuid",
		"virtualCenterSecret",
		"vsphereSecret",
		"weavePasswordSecret",
	},
	"PodSecurityAdmissionConfigurationTemplate": {
		"annotations",
		"configuration",
		"created",
		"creatorId",
		"description",
		"labels",
		"name",
		"ownerReferences",
		"removed",
		"uuid",
	},
	"PodSecurityAdmissionConfigurationTemplateSpec": {
		"defaults",
		"exemptions",
	},
	"PodSecurityAdmissionConfigurationTemplateDefaults": {
		"audit",
		"audit-version",
		"enforce",
		"enforce-version",
		"warn",
		"warn-version",
	},
	"PodSecurityAdmissionConfigurationTemplateExemptions": {
		"namespaces",
		"runtimeClasses",
		"usernames",
	},
	"AuthConfig": {
		"accessMode",
		"allowedPrincipalIds",
		"annotations",
		"created",
		"creatorId",
		"enabled",
		"labels",
		"logoutAllSupported",
		"name",
		"ownerReferences",
		"removed",
		"status",
		"type",
		"uuid",
	},
	"Token": {
		"activityLastSeenAt",
		"annotations",
		"authProvider",
		"clusterId",
		"created",
		"creatorId",
		"current",
		"description",
		"enabled",
		"expired",
		"expiresAt",
		"groupPrincipals",
		"isDerived",
		"labels",
		"lastUsedAt",
		"name",
		"ownerReferences",
		"providerInfo",
		"removed",
		"token",
		"ttl",
		"userId",
		"userPrincipal",
		"uuid",
	},
	"User": {
		"annotations",
		"conditions",
		"created",
		"creatorId",
		"description",
		"enabled",
		"labels",
		"me",
		"mustChangePassword",
		"name",
		"ownerReferences",
		"password",
		"principalIds",
		"removed",
		"state",
		"transitioning",
		"transitioningMessage",
		"username",
		"uuid",
	},
	"DockerCredential": {
		"annotations",
		"created",
		"creatorId",
		"description",
		"labels",
		"name",
		"namespaceId",
		"ownerReferences",
		"projectId",
		"registries",
		"removed",
		"uuid",
	},
	"NamespacedDockerCredential": {
		"annotations",
		"created",
		"creatorId",
		"description",
		"labels",
		"name",
		"namespaceId",
		"ownerReferences",
		"projectId",
		"registries",
		"removed",
		"uuid",
	},
	"RegistryCredential": {
		"auth",
		"description",
		"email",
		"password",
		"username",
	},
}

// upstreamResourceFields are the four fields norman's types.Resource embeds
// into every generated management and project type. They have no
// `<Type>Field<Name>` constant, which is why `id` appears on the records below
// but in none of the lists above.
var upstreamResourceFields = []string{"id", "type", "links", "actions"}

// recordSources names, for each wire record this provider decodes, the
// generated type or types it mirrors. bindingRecord covers both binding
// endpoints, which differ only in whether the scope is a cluster or a project,
// so it is held against the union of the two.
var recordSources = map[string][]string{
	"settingRecord":                 {"Setting"},
	"clusterRecord":                 {"Cluster"},
	"versionInfo":                   {"Info"},
	"localClusterAuthEndpoint":      {"LocalClusterAuthEndpoint"},
	"projectRecord":                 {"Project"},
	"policyRule":                    {"PolicyRule"},
	"globalRoleRecord":              {"GlobalRole"},
	"globalRoleBindingRecord":       {"GlobalRoleBinding"},
	"roleTemplateRecord":            {"RoleTemplate"},
	"bindingRecord":                 {"ClusterRoleTemplateBinding", "ProjectRoleTemplateBinding"},
	"clusterTemplateRecord":         {"ClusterTemplate"},
	"clusterTemplateRevisionRecord": {"ClusterTemplateRevision"},
	"podSecurityTemplateRecord":     {"PodSecurityAdmissionConfigurationTemplate"},
	"podSecurityConfiguration":      {"PodSecurityAdmissionConfigurationTemplateSpec"},
	"podSecurityDefaults":           {"PodSecurityAdmissionConfigurationTemplateDefaults"},
	"podSecurityExemptions":         {"PodSecurityAdmissionConfigurationTemplateExemptions"},
	"authConfigRecord":              {"AuthConfig"},
	"tokenRecord":                   {"Token"},
	"userRecord":                    {"User"},
	"dockerCredentialRecord":        {"DockerCredential"},
	"registryAccount":               {"RegistryCredential"},
}

// unreleasedFields are wire names this provider decodes that the pinned
// generated clients do not declare. Each one is a deliberate bet on an
// unreleased Rancher and has to carry its evidence, because the same test
// result would otherwise be produced by a typo.
var unreleasedFields = map[string]string{
	// Found by this test rather than by reading. GlobalRole gained
	// inheritedNamespacedRules on rancher/rancher main only: it is absent from
	// the generated client at v2.10.7, v2.11.6, v2.12.0, v2.12.5 and v2.12.9,
	// and present on main. So rancher.globalRole.inheritedNamespacedRules
	// reads null against every Rancher released so far and starts carrying
	// data on 2.13. The field stays, because removing a shipped field is
	// breaking and because it will be correct shortly, but nothing should be
	// read into a null there today.
	"GlobalRole.inheritedNamespacedRules": "main only; not in any release through v2.12.9",
}

// notCredentials are wire names that match the credential patterns without
// being credentials. Each is a claim that has to be true on its own terms, not
// a way to quiet the sweep.
var notCredentials = map[string]string{
	// A boolean saying the account must set a new password at next login. It
	// carries no password and never has a value beyond true or false.
	"mustchangepassword": "a policy flag, not a password",
}

// allRecordTypes is every wire record type in this package. A record missing
// from here is a record nothing below checks, so
// TestEveryRecordTypeIsAccountedFor holds the two lists level with each other.
func allRecordTypes() []any {
	return []any{
		settingRecord{}, clusterRecord{}, versionInfo{}, localClusterAuthEndpoint{},
		projectRecord{}, policyRule{}, globalRoleRecord{}, globalRoleBindingRecord{},
		roleTemplateRecord{}, bindingRecord{}, clusterTemplateRecord{},
		clusterTemplateRevisionRecord{}, podSecurityTemplateRecord{},
		podSecurityConfiguration{}, podSecurityDefaults{}, podSecurityExemptions{},
		authConfigRecord{}, tokenRecord{}, userRecord{}, dockerCredentialRecord{},
		registryAccount{},
	}
}

// taggedField is one decoded field: the record it sits on and the wire name it
// reads.
type taggedField struct {
	record string
	tag    string
}

// walkTags visits every json tag reachable from a type, descending through
// pointers, slices and map values so that a nested record is checked on the
// same terms as a top-level one. Without the descent a credential field on an
// inner type would be invisible: a registry password is only reachable as
// dockerCredentialRecord.Registries[host].Password.
func walkTags(t reflect.Type, seen map[reflect.Type]bool, out *[]taggedField) {
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		t = t.Elem()
	}
	if t.Kind() == reflect.Map {
		walkTags(t.Elem(), seen, out)
		return
	}
	if t.Kind() != reflect.Struct || seen[t] {
		return
	}
	seen[t] = true

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := strings.Split(field.Tag.Get("json"), ",")[0]
		if tag != "" && tag != "-" {
			*out = append(*out, taggedField{record: t.Name(), tag: tag})
		}
		walkTags(field.Type, seen, out)
	}
}

// tagsOf returns every json tag a type decodes, including nested ones.
func tagsOf(value any) []taggedField {
	var out []taggedField
	walkTags(reflect.TypeOf(value), map[reflect.Type]bool{}, &out)
	return out
}

// credentialNames picks, out of a set of wire names, the ones that carry or
// name credential material.
//
// The substring patterns are what Rancher actually spells its secret-bearing
// fields with, read off the inventory above: aadClientSecret, caCert,
// s3CredentialSecret, bastionHostSSHKeySecret and the rest. The two exact
// matches are the bare `token` on Token and the bare `auth` on
// RegistryCredential, which are the values themselves; matching those as
// substrings instead would catch authProvider and authConfigs, which are
// discriminators and are read on purpose.
func credentialNames(names []string) []string {
	substrings := []string{"secret", "password", "privatekey", "cacert", "credential"}
	exact := map[string]bool{"token": true, "auth": true, "kubeconfig": true}

	var found []string
	for _, name := range names {
		lowered := strings.ToLower(name)
		if _, allowed := notCredentials[lowered]; allowed {
			continue
		}
		if exact[lowered] {
			found = append(found, name)
			continue
		}
		for _, pattern := range substrings {
			if strings.Contains(lowered, pattern) {
				found = append(found, name)
				break
			}
		}
	}
	return found
}

// TestEveryRecordTypeIsAccountedFor keeps the checks below from going quiet as
// records are added. A new wire type that nobody listed would otherwise be
// checked against neither the inventory nor the credential sweep.
func TestEveryRecordTypeIsAccountedFor(t *testing.T) {
	for _, record := range allRecordTypes() {
		name := reflect.TypeOf(record).Name()
		_, ok := recordSources[name]
		assert.True(t, ok, "%s has no entry in recordSources", name)
	}
	assert.Equal(t, len(recordSources), len(allRecordTypes()),
		"recordSources and allRecordTypes must name the same records")
}

// TestRecordTagsAreDeclaredUpstream holds every tag this provider decodes
// against the generated field inventory. A tag that is not there is a name the
// server does not send, and a field spelled that way reads null on every
// object without ever erroring.
func TestRecordTagsAreDeclaredUpstream(t *testing.T) {
	for _, record := range allRecordTypes() {
		recordName := reflect.TypeOf(record).Name()
		for _, field := range tagsOf(record) {
			sources, ok := recordSources[field.record]
			if !ok {
				continue // reported by TestEveryRecordTypeIsAccountedFor
			}

			declared := append([]string{}, upstreamResourceFields...)
			unreleased := false
			for _, source := range sources {
				fields, known := upstreamFields[source]
				require.True(t, known, "no upstream inventory for %s", source)
				declared = append(declared, fields...)
				if _, ok := unreleasedFields[source+"."+field.tag]; ok {
					unreleased = true
				}
			}
			if unreleased {
				continue
			}

			assert.Contains(t, declared, field.tag,
				"%s (reached from %s) decodes %q, which %s does not declare",
				field.record, recordName, field.tag, strings.Join(sources, " or "))
		}
	}
}

// TestBothRegistryCollectionsHaveTheSameShape backs the one place a single
// record covers two endpoints without saying so in recordSources.
// registryCredentials() reads both /v3/project/<id>/dockerCredentials and
// .../namespacedDockerCredentials into dockerCredentialRecord, which is only
// safe while the two generated types agree. Checking the two inventories
// against each other rather than taking their union is the point: a union
// would swallow a divergence, and this fails on one.
func TestBothRegistryCollectionsHaveTheSameShape(t *testing.T) {
	assert.Equal(t, upstreamFields["DockerCredential"], upstreamFields["NamespacedDockerCredential"],
		"the two docker credential collections no longer share a shape, so dockerCredentialRecord cannot cover both")
}

// TestExceptionsAreStillNeeded keeps both allowlists from outliving their
// reason. An exception that no longer applies is a hole nobody is watching:
// once Rancher ships inheritedNamespacedRules the entry has to go, or the
// ordinary check for that field never runs again.
func TestExceptionsAreStillNeeded(t *testing.T) {
	for key := range unreleasedFields {
		source, field, ok := strings.Cut(key, ".")
		require.True(t, ok, "unreleasedFields key %q must read Type.field", key)

		fields, known := upstreamFields[source]
		require.True(t, known, "unreleasedFields names unknown type %q", source)
		assert.NotContains(t, fields, field,
			"%s now declares %q, so the exception is stale and must be removed", source, field)
	}

	for name := range notCredentials {
		found := false
		for _, fields := range upstreamFields {
			for _, candidate := range fields {
				if strings.EqualFold(candidate, name) {
					found = true
				}
			}
		}
		assert.True(t, found,
			"notCredentials excuses %q, which no upstream type declares any more", name)
	}
}

// TestNoRecordTypeDecodesAnUpstreamCredentialField is the structural half of
// the guarantee this provider makes about secrets. Rancher returns credential
// material on several of the objects listed above; none of it may have a field
// to land in, on any record, at any depth.
//
// It is driven by the upstream inventory rather than by a hand-written list of
// forbidden names, so a secret Rancher adds to a type is covered the next time
// the inventory is refreshed rather than the next time somebody remembers.
func TestNoRecordTypeDecodesAnUpstreamCredentialField(t *testing.T) {
	forbidden := map[string]bool{}
	for _, fields := range upstreamFields {
		for _, name := range credentialNames(fields) {
			forbidden[strings.ToLower(name)] = true
		}
	}
	require.NotEmpty(t, forbidden, "the inventory must contain credential fields to exclude")

	for _, record := range allRecordTypes() {
		recordName := reflect.TypeOf(record).Name()
		for _, field := range tagsOf(record) {
			assert.False(t, forbidden[strings.ToLower(field.tag)],
				"%s (reached from %s) decodes the credential field %q",
				field.record, recordName, field.tag)
		}
	}
}

// The shapes below reproduce the generated types this provider refuses to
// decode into, field for field on the parts that matter. They exist so that
// the guard above is shown to fire rather than assumed to: a sweep that
// forbids nothing passes on everything, and a sweep whose recursion is broken
// passes on a credential one level down.
type upstreamTokenShape struct {
	ID        string `json:"id"`
	TTLMillis int64  `json:"ttl"`
	Token     string `json:"token"`
}

type upstreamUserShape struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type upstreamRegistryCredentialShape struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Auth     string `json:"auth"`
	Email    string `json:"email"`
}

type upstreamDockerCredentialShape struct {
	ID         string                                     `json:"id"`
	Registries map[string]upstreamRegistryCredentialShape `json:"registries"`
}

type upstreamClusterShape struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	AADClientSecret string `json:"aadClientSecret"`
	CACert          string `json:"caCert"`
}

type upstreamClusterTemplateRevisionShape struct {
	ID                 string `json:"id"`
	S3CredentialSecret string `json:"s3CredentialSecret"`
}

// TestCredentialGuardIsNotVacuous points the sweep at the upstream shapes it
// exists to exclude and requires it to trip on each of them.
func TestCredentialGuardIsNotVacuous(t *testing.T) {
	forbidden := map[string]bool{}
	for _, fields := range upstreamFields {
		for _, name := range credentialNames(fields) {
			forbidden[strings.ToLower(name)] = true
		}
	}

	tests := []struct {
		name  string
		shape any
		want  string
	}{
		{"token value", upstreamTokenShape{}, "token"},
		{"user password", upstreamUserShape{}, "password"},
		{"registry password behind a map", upstreamDockerCredentialShape{}, "password"},
		{"registry auth header behind a map", upstreamDockerCredentialShape{}, "auth"},
		{"cluster cloud secret", upstreamClusterShape{}, "aadClientSecret"},
		{"cluster CA certificate", upstreamClusterShape{}, "caCert"},
		{"revision credential secret", upstreamClusterTemplateRevisionShape{}, "s3CredentialSecret"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var tripped []string
			for _, field := range tagsOf(test.shape) {
				if forbidden[strings.ToLower(field.tag)] {
					tripped = append(tripped, field.tag)
				}
			}
			assert.Contains(t, tripped, test.want,
				"the sweep did not catch %q, so it proves nothing about the records", test.want)
		})
	}

	// The same shapes must otherwise look like ordinary records: a guard that
	// rejected every field would also "pass" here while saying nothing.
	for _, field := range tagsOf(upstreamTokenShape{}) {
		if field.tag == "id" || field.tag == "ttl" {
			assert.False(t, forbidden[field.tag],
				"%q is not a credential and must not trip the sweep", field.tag)
		}
	}
}

// TestWireNamesThatContradictTheStoredKubernetesTypes pins the nine places
// where the v3 API disagrees with the Kubernetes object behind it. Each was
// found by reading the generated client, and each is a null field or a wrong
// answer if it is ever "corrected" back to the stored spelling.
func TestWireNamesThatContradictTheStoredKubernetesTypes(t *testing.T) {
	tests := []struct {
		stored     string
		wire       string
		upstream   string
		reportedBy string
	}{
		{"Token.TTLMillis", "ttl", "Token", "every token reads as never expiring"},
		{"Token.LastUsedAt (metav1.Time)", "lastUsedAt", "Token", "a never-used token reads as used in year one"},
		{"Token.UserPrincipal (Principal)", "userPrincipal", "Token", "the principal reads empty"},
		{"GlobalRoleBinding.GlobalRoleName", "globalRoleId", "GlobalRoleBinding", "no binding resolves its role"},
		{"RoleTemplate.RoleTemplateNames", "roleTemplateIds", "RoleTemplate", "inherited templates read empty"},
		{"Project.ClusterName", "clusterId", "Project", "no project resolves its cluster"},
		{"PodSecurityDefaults.EnforceVersion", "enforce-version", "PodSecurityAdmissionConfigurationTemplateDefaults", "a pinned policy version reads as unpinned"},
		{"PodSecurityDefaults.AuditVersion", "audit-version", "PodSecurityAdmissionConfigurationTemplateDefaults", "a pinned policy version reads as unpinned"},
		{"PodSecurityDefaults.WarnVersion", "warn-version", "PodSecurityAdmissionConfigurationTemplateDefaults", "a pinned policy version reads as unpinned"},
	}

	for _, test := range tests {
		t.Run(test.wire, func(t *testing.T) {
			assert.Contains(t, upstreamFields[test.upstream], test.wire,
				"%s is spelled %q on the wire; getting it wrong means %s",
				test.stored, test.wire, test.reportedBy)
		})
	}

	// The two identity renames have no constant of their own, because Norman
	// puts them on the embedded resource: a cluster's spec.displayName becomes
	// name and its metadata.name becomes id.
	assert.Contains(t, upstreamFields["Cluster"], "name")
	assert.Contains(t, upstreamResourceFields, "id")
	assert.NotContains(t, upstreamFields["Cluster"], "displayName")
}

// TestUpstreamCarriesTheFieldsThisProviderRefuses is the other half of the
// provenance: the credential fields named in the record comments have to
// actually exist upstream, or those comments are describing a danger that is
// not there and the reader stops trusting them.
func TestUpstreamCarriesTheFieldsThisProviderRefuses(t *testing.T) {
	tests := []struct{ upstream, field string }{
		{"Token", "token"},
		{"User", "password"},
		{"RegistryCredential", "password"},
		{"RegistryCredential", "auth"},
		{"Cluster", "aadClientSecret"},
		{"Cluster", "caCert"},
		{"Cluster", "clusterSecrets"},
		{"Cluster", "serviceAccountTokenSecret"},
		{"ClusterTemplateRevision", "clusterConfig"},
		{"ClusterTemplateRevision", "s3CredentialSecret"},
		{"ClusterTemplateRevision", "bastionHostSSHKeySecret"},
	}

	for _, test := range tests {
		t.Run(test.upstream+"."+test.field, func(t *testing.T) {
			assert.Contains(t, upstreamFields[test.upstream], test.field)
		})
	}
}
