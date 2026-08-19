// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

// The structs below mirror what the Rancher Manager v3 API puts on the wire.
//
// Every field name and type here is taken from Rancher's own generated v3
// client, which is machine generated from the schema a running server serves
// after every Norman mapper has been applied. Two upstream revisions are the
// source, both pinned:
//
//	rancher/rancher v2.12.9 (db2754edc35189187bb10c524601d3d62642ff9b)
//	  pkg/client/generated/management/v3/zz_generated_*.go
//	  pkg/client/generated/project/v3/zz_generated_docker_credential.go
//	  pkg/client/generated/project/v3/zz_generated_registry_credential.go
//	rancher/rancher v2.11.6 (8e2eb63d5d2b4744ab3b4ab44de573106519d77d)
//	  pkg/client/generated/management/v3/zz_generated_cluster_template.go
//	  pkg/client/generated/management/v3/zz_generated_cluster_template_revision.go
//
// The second revision is needed because 2.12 deleted the cluster template
// types along with RKE1, while servers still running 2.11 serve them.
//
// The generated package is not imported, and wireformat_test.go records the
// measurement and the reasons. The short version: the generated structs and the
// Norman client live in one Go package, so the types cannot be taken without
// the transport, and three of those structs decode credentials this provider
// must never hold. What is adopted instead is the generated *field inventory*:
// wireformat_test.go carries the wire names upstream declares for each type,
// verbatim from its generated `<Type>Field<Name>` constants, and asserts that
// every tag below is one of them and that no credential-bearing name is.
//
// The wire shape is not the shape of the Kubernetes objects behind it. Rancher
// serves the management API through Norman, which flattens metadata, spec and
// status into one object, renames creationTimestamp to created and uid to uuid,
// moves a resource's displayName onto name and its Kubernetes name onto id, and
// rewrites every reference field ending in Name to end in Id. So a cluster's
// clusterName arrives as clusterId, a binding's roleTemplateName as
// roleTemplateId, and a role template's roleTemplateNames as roleTemplateIds.
//
// The `id` every record below decodes is contributed by norman's embedded
// types.Resource rather than by the generated struct, which is why it has no
// `<Type>FieldID` constant upstream.
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
//
// The generated Cluster declares 77 fields; this decodes 20 of them, plus the
// id norman contributes. Among the ones left out are aadClientSecret,
// aadClientCertSecret, caCert, clusterSecrets, openStackSecret,
// privateRegistrySecret, s3CredentialSecret, serviceAccountTokenSecret,
// virtualCenterSecret, vsphereSecret and weavePasswordSecret, and the
// aksConfig, eksConfig, gkeConfig, rancherKubernetesEngineConfig, appliedSpec
// and failedSpec specifications that carry provisioning credentials of their
// own. None of them has a field here to land in, and
// TestNoRecordTypeDecodesAnUpstreamCredentialField holds that open against the
// upstream inventory rather than against this list.
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
//
// ResourceQuota and ContainerDefaultResourceLimit stay map[string]any rather
// than the generated *ProjectResourceQuota and *ContainerResourceLimit. Both
// reach MQL as dicts, and the generated structs are closed lists: a quota key
// Rancher adds, or an extended resource under a name the struct does not
// declare, would decode to nothing and a project with a quota would report an
// incomplete one. A map keeps whatever the server sent.
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
//
// Rancher 2.12 deleted the generated ClusterTemplate type with the rest of
// RKE1, so this and clusterTemplateRevisionRecord are pinned to the 2.11.6
// generated client. A server still serving the endpoint is by definition
// running a release that still has the type.
//
// Members stays a []map[string]any rather than the generated []Member. The
// members field reaches MQL as a dict, and decoding through the generated
// struct would silently drop any key outside its five fields.
type clusterTemplateRecord struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	Description       string           `json:"description"`
	Created           string           `json:"created"`
	DefaultRevisionID string           `json:"defaultRevisionId"`
	Members           []map[string]any `json:"members"`
}

// clusterTemplateRevisionRecord is one entry of /v3/clusterTemplateRevisions.
//
// This is the worst-populated type in the whole API for credential material:
// the generated ClusterTemplateRevision declares thirty-one fields, of which
// fifteen are named secrets (aadClientSecret, aadClientCertSecret,
// aciAPICUserKeySecret, aciKafkaClientKeySecret, aciTokenSecret,
// bastionHostSSHKeySecret, kubeletExtraEnvSecret, openStackSecret,
// privateRegistryECRSecret, privateRegistrySecret, s3CredentialSecret,
// secretsEncryptionProvidersSecret, virtualCenterSecret, vsphereSecret,
// weavePasswordSecret) and one, clusterConfig, is the whole cluster
// specification. Seven fields are decoded and none of those is among them.
//
// Questions stays a []map[string]any for the same reason Members does on the
// template: it reaches MQL as a dict, and the generated []Question would
// narrow it to that struct's fields.
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
//
// The generated Token declares a Token string tagged json:"token" alongside
// these, which is the token value itself. It is deliberately absent: a struct
// without the field cannot decode the value whatever the server sends, whereas
// a struct with it would hold the secret in process and one careless field
// mapping away from a query result.
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

// userRecord is one entry of /v3/users.
//
// The generated User declares a Password string tagged json:"password", which
// the schema marks write-only. Write-only is the server's current behavior; not
// having the field is this provider's own guarantee, and the second is the one
// worth relying on.
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
// name only. Upstream this map is a map[string]RegistryCredential, and the
// generated RegistryCredential declares auth, description, email, password and
// username. Substituting a one-field type for it is what keeps a registry
// password, and the pre-encoded auth header that is equivalent to one, from
// ever entering the process, let alone a query result.
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
