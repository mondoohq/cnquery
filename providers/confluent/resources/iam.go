// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"net/url"
	"strings"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/confluent/connection"
)

// --- service accounts -----------------------------------------------------

type serviceAccountRecord struct {
	ID          string     `json:"id"`
	DisplayName string     `json:"display_name"`
	Description string     `json:"description"`
	Metadata    objectMeta `json:"metadata"`
}

func (r *mqlConfluent) serviceAccounts() ([]any, error) {
	conn, err := confluentConn(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	records, err := connection.GetPaged[serviceAccountRecord](context.Background(), conn,
		conn.CloudTarget(), "/iam/v2/service-accounts", nil)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		record := records[i]
		mqlAccount, err := CreateResource(r.MqlRuntime, "confluent.serviceAccount", map[string]*llx.RawData{
			"__id":         llx.StringData(record.ID),
			"id":           llx.StringData(record.ID),
			"displayName":  llx.StringData(record.DisplayName),
			"description":  llx.StringData(record.Description),
			"resourceName": llx.StringData(record.Metadata.ResourceName),
			"createdAt":    llx.TimeDataPtr(record.Metadata.CreatedAt.Time()),
			"updatedAt":    llx.TimeDataPtr(record.Metadata.UpdatedAt.Time()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAccount)
	}
	return res, nil
}

// serviceAccountByID resolves a service account from the root resource's cached
// list.
func serviceAccountByID(runtime *plugin.Runtime, id string) (*mqlConfluentServiceAccount, error) {
	if id == "" {
		return nil, nil
	}
	root, err := getConfluent(runtime)
	if err != nil {
		return nil, err
	}
	accounts := root.GetServiceAccounts()
	if accounts.Error != nil {
		return nil, accounts.Error
	}
	for _, raw := range accounts.Data {
		account, ok := raw.(*mqlConfluentServiceAccount)
		if !ok {
			continue
		}
		if account.GetId().Data == id {
			return account, nil
		}
	}
	return nil, nil
}

func (r *mqlConfluentServiceAccount) apiKeys() ([]any, error) {
	return apiKeysOwnedBy(r.MqlRuntime, r.GetId().Data)
}

func (r *mqlConfluentServiceAccount) roleBindings() ([]any, error) {
	return roleBindingsFor(r.MqlRuntime, r.GetId().Data)
}

// --- users ----------------------------------------------------------------

type userRecord struct {
	ID       string     `json:"id"`
	Email    string     `json:"email"`
	FullName string     `json:"full_name"`
	AuthType string     `json:"auth_type"`
	Metadata objectMeta `json:"metadata"`
}

func (r *mqlConfluent) users() ([]any, error) {
	conn, err := confluentConn(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	records, err := connection.GetPaged[userRecord](context.Background(), conn,
		conn.CloudTarget(), "/iam/v2/users", nil)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		record := records[i]
		mqlUser, err := CreateResource(r.MqlRuntime, "confluent.user", map[string]*llx.RawData{
			"__id":         llx.StringData(record.ID),
			"id":           llx.StringData(record.ID),
			"email":        llx.StringData(record.Email),
			"fullName":     llx.StringData(record.FullName),
			"authType":     llx.StringData(record.AuthType),
			"resourceName": llx.StringData(record.Metadata.ResourceName),
			"createdAt":    llx.TimeDataPtr(record.Metadata.CreatedAt.Time()),
			"updatedAt":    llx.TimeDataPtr(record.Metadata.UpdatedAt.Time()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlUser)
	}
	return res, nil
}

// userByID resolves a user from the root resource's cached list.
func userByID(runtime *plugin.Runtime, id string) (*mqlConfluentUser, error) {
	if id == "" {
		return nil, nil
	}
	root, err := getConfluent(runtime)
	if err != nil {
		return nil, err
	}
	users := root.GetUsers()
	if users.Error != nil {
		return nil, users.Error
	}
	for _, raw := range users.Data {
		user, ok := raw.(*mqlConfluentUser)
		if !ok {
			continue
		}
		if user.GetId().Data == id {
			return user, nil
		}
	}
	return nil, nil
}

func (r *mqlConfluentUser) apiKeys() ([]any, error) {
	return apiKeysOwnedBy(r.MqlRuntime, r.GetId().Data)
}

func (r *mqlConfluentUser) roleBindings() ([]any, error) {
	return roleBindingsFor(r.MqlRuntime, r.GetId().Data)
}

// --- API keys -------------------------------------------------------------

// mqlConfluentApiKeyInternal caches the owner and the scoped resource, which
// the payload carries as references rather than as objects.
type mqlConfluentApiKeyInternal struct {
	cachedOwnerID    string
	cachedResourceID string
}

type apiKeySpecRecord struct {
	DisplayName string            `json:"display_name"`
	Description string            `json:"description"`
	Owner       *objectReference  `json:"owner"`
	Resource    *objectReference  `json:"resource"`
	Resources   []objectReference `json:"resources"`
}

type apiKeyRecord struct {
	ID       string            `json:"id"`
	Metadata objectMeta        `json:"metadata"`
	Spec     *apiKeySpecRecord `json:"spec"`
}

// scopedResource returns the resource an API key opens. Newer responses carry a
// `resources` list alongside the single `resource`, so both are read and the
// singular one wins where present.
func (s *apiKeySpecRecord) scopedResource() *objectReference {
	if s == nil {
		return nil
	}
	if s.Resource != nil && s.Resource.ID != "" {
		return s.Resource
	}
	for i := range s.Resources {
		if s.Resources[i].ID != "" {
			return &s.Resources[i]
		}
	}
	return nil
}

// referenceKind renders an object reference as the fully qualified kind the
// management API uses, for example "cmk.v2.Cluster". A reference missing either
// half yields the empty string rather than a half-formed kind.
func referenceKind(ref *objectReference) string {
	if ref == nil || ref.APIVersion == "" || ref.Kind == "" {
		return ""
	}
	return strings.ReplaceAll(ref.APIVersion, "/", ".") + "." + ref.Kind
}

// ownerKindOf names the kind of principal that owns a key. The reference
// usually carries it, and the identifier prefix answers it when it does not.
func ownerKindOf(ref *objectReference) string {
	if ref == nil {
		return ""
	}
	if ref.Kind != "" {
		return ref.Kind
	}
	switch {
	case strings.HasPrefix(ref.ID, "sa-"):
		return "ServiceAccount"
	case strings.HasPrefix(ref.ID, "u-"):
		return "User"
	default:
		return ""
	}
}

func (r *mqlConfluent) apiKeys() ([]any, error) {
	conn, err := confluentConn(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	records, err := connection.GetPaged[apiKeyRecord](context.Background(), conn,
		conn.CloudTarget(), "/iam/v2/api-keys", nil)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	res := make([]any, 0, len(records))
	for i := range records {
		record := records[i]
		spec := record.Spec
		if spec == nil {
			spec = &apiKeySpecRecord{}
		}
		resource := spec.scopedResource()

		mqlKey, err := CreateResource(r.MqlRuntime, "confluent.apiKey", map[string]*llx.RawData{
			"__id":         llx.StringData(record.ID),
			"id":           llx.StringData(record.ID),
			"displayName":  llx.StringData(spec.DisplayName),
			"description":  llx.StringData(spec.Description),
			"resourceName": llx.StringData(record.Metadata.ResourceName),
			"createdAt":    llx.TimeDataPtr(record.Metadata.CreatedAt.Time()),
			"updatedAt":    llx.TimeDataPtr(record.Metadata.UpdatedAt.Time()),
			"ageDays":      ageInDays(record.Metadata.CreatedAt.Time(), now),
			"ownerKind":    llx.StringData(ownerKindOf(spec.Owner)),
			"isCloudKey":   llx.BoolData(refID(resource) == ""),
			"resourceKind": llx.StringData(referenceKind(resource)),
		})
		if err != nil {
			return nil, err
		}

		key := mqlKey.(*mqlConfluentApiKey)
		key.cachedOwnerID = refID(spec.Owner)
		key.cachedResourceID = refID(resource)
		res = append(res, key)
	}
	return res, nil
}

// apiKeysOwnedBy returns every key owned by one principal, read from the root
// resource's cached key list.
func apiKeysOwnedBy(runtime *plugin.Runtime, ownerID string) ([]any, error) {
	if ownerID == "" {
		return []any{}, nil
	}
	root, err := getConfluent(runtime)
	if err != nil {
		return nil, err
	}
	keys := root.GetApiKeys()
	if keys.Error != nil {
		return nil, keys.Error
	}

	res := []any{}
	for _, raw := range keys.Data {
		key, ok := raw.(*mqlConfluentApiKey)
		if !ok {
			continue
		}
		if key.cachedOwnerID == ownerID {
			res = append(res, key)
		}
	}
	return res, nil
}

func (r *mqlConfluentApiKey) serviceAccount() (*mqlConfluentServiceAccount, error) {
	if !isServiceAccountID(r.cachedOwnerID) {
		r.ServiceAccount.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	account, err := serviceAccountByID(r.MqlRuntime, r.cachedOwnerID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		r.ServiceAccount.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return account, nil
}

func (r *mqlConfluentApiKey) user() (*mqlConfluentUser, error) {
	if !strings.HasPrefix(r.cachedOwnerID, "u-") {
		r.User.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	user, err := userByID(r.MqlRuntime, r.cachedOwnerID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		r.User.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return user, nil
}

func (r *mqlConfluentApiKey) cluster() (*mqlConfluentKafkaCluster, error) {
	if !strings.HasPrefix(r.cachedResourceID, "lkc-") {
		r.Cluster.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	cluster, err := kafkaClusterByID(r.MqlRuntime, r.cachedResourceID)
	if err != nil {
		return nil, err
	}
	if cluster == nil {
		r.Cluster.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return cluster, nil
}

func (r *mqlConfluentApiKey) schemaRegistryCluster() (*mqlConfluentSchemaRegistryCluster, error) {
	if !strings.HasPrefix(r.cachedResourceID, "lsrc-") {
		r.SchemaRegistryCluster.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	cluster, err := schemaRegistryClusterByID(r.MqlRuntime, r.cachedResourceID)
	if err != nil {
		return nil, err
	}
	if cluster == nil {
		r.SchemaRegistryCluster.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return cluster, nil
}

// --- role bindings --------------------------------------------------------

// mqlConfluentRoleBindingInternal caches what the scope pattern names, so the
// typed accessors and the reverse edges do not reparse it per query.
type mqlConfluentRoleBindingInternal struct {
	cachedPrincipalID   string
	cachedEnvironmentID string
	cachedClusterID     string
}

type roleBindingRecord struct {
	ID         string     `json:"id"`
	Metadata   objectMeta `json:"metadata"`
	Principal  string     `json:"principal"`
	RoleName   string     `json:"role_name"`
	CrnPattern string     `json:"crn_pattern"`
}

// crnSegment is one `key=value` element of a Confluent Resource Name path.
type crnSegment struct {
	Key   string
	Value string
}

// parseCRN splits a Confluent Resource Name into its ordered path segments. The
// authority ("confluent.cloud") carries no key and is skipped, as is any
// element that is not a key/value pair.
func parseCRN(crn string) []crnSegment {
	trimmed := strings.TrimSpace(crn)
	if trimmed == "" {
		return nil
	}
	if idx := strings.Index(trimmed, "://"); idx >= 0 {
		trimmed = trimmed[idx+3:]
	}
	trimmed = strings.Trim(trimmed, "/")

	var out []crnSegment
	for _, part := range strings.Split(trimmed, "/") {
		key, value, found := strings.Cut(part, "=")
		if !found || key == "" {
			continue
		}
		out = append(out, crnSegment{Key: key, Value: value})
	}
	return out
}

// crnValue returns the value of one segment of a Confluent Resource Name, or
// the empty string when the segment is absent.
func crnValue(segments []crnSegment, key string) string {
	for _, segment := range segments {
		if segment.Key == key {
			return segment.Value
		}
	}
	return ""
}

// crnScopeKind names what a scope pattern targets, which is the deepest segment
// it carries.
func crnScopeKind(segments []crnSegment) string {
	if len(segments) == 0 {
		return ""
	}
	return segments[len(segments)-1].Key
}

// crnClusterID returns the Kafka cluster a scope pattern targets. Confluent
// writes the cluster as `cloud-cluster` on a scope and as `kafka` on the
// resource path below it, so both are read.
func crnClusterID(segments []crnSegment) string {
	if id := crnValue(segments, "cloud-cluster"); id != "" && id != "*" {
		return id
	}
	if id := crnValue(segments, "kafka"); id != "" && id != "*" {
		return id
	}
	return ""
}

func (r *mqlConfluent) roleBindings() ([]any, error) {
	conn, err := confluentConn(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	org, err := r.fetchOrganization()
	if err != nil {
		return nil, err
	}

	// The listing requires a scope. Rooting it at the organization is a partial
	// match, so it returns every binding at or below the organization rather
	// than only the organization-wide ones.
	query := url.Values{}
	query.Set("crn_pattern", "crn://confluent.cloud/organization="+org.ID)

	records, err := connection.GetPaged[roleBindingRecord](context.Background(), conn,
		conn.CloudTarget(), "/iam/v2/role-bindings", query)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		record := records[i]
		segments := parseCRN(record.CrnPattern)
		scopeKind := crnScopeKind(segments)

		mqlBinding, err := CreateResource(r.MqlRuntime, "confluent.roleBinding", map[string]*llx.RawData{
			"__id":                 llx.StringData(record.ID),
			"id":                   llx.StringData(record.ID),
			"principal":            llx.StringData(record.Principal),
			"roleName":             llx.StringData(record.RoleName),
			"crnPattern":           llx.StringData(record.CrnPattern),
			"scopeKind":            llx.StringData(scopeKind),
			"isOrganizationScoped": llx.BoolData(scopeKind == "organization"),
		})
		if err != nil {
			return nil, err
		}

		binding := mqlBinding.(*mqlConfluentRoleBinding)
		binding.cachedPrincipalID = principalAccountID(record.Principal)
		binding.cachedEnvironmentID = crnValue(segments, "environment")
		binding.cachedClusterID = crnClusterID(segments)
		res = append(res, binding)
	}
	return res, nil
}

// roleBindingsFor returns every binding held by one principal, read from the
// root resource's cached binding list.
func roleBindingsFor(runtime *plugin.Runtime, principalID string) ([]any, error) {
	if principalID == "" {
		return []any{}, nil
	}
	root, err := getConfluent(runtime)
	if err != nil {
		return nil, err
	}
	bindings := root.GetRoleBindings()
	if bindings.Error != nil {
		return nil, bindings.Error
	}

	res := []any{}
	for _, raw := range bindings.Data {
		binding, ok := raw.(*mqlConfluentRoleBinding)
		if !ok {
			continue
		}
		if binding.cachedPrincipalID == principalID {
			res = append(res, binding)
		}
	}
	return res, nil
}

func (r *mqlConfluentRoleBinding) serviceAccount() (*mqlConfluentServiceAccount, error) {
	if !isServiceAccountID(r.cachedPrincipalID) {
		r.ServiceAccount.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	account, err := serviceAccountByID(r.MqlRuntime, r.cachedPrincipalID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		r.ServiceAccount.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return account, nil
}

func (r *mqlConfluentRoleBinding) user() (*mqlConfluentUser, error) {
	if !strings.HasPrefix(r.cachedPrincipalID, "u-") {
		r.User.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	user, err := userByID(r.MqlRuntime, r.cachedPrincipalID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		r.User.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return user, nil
}

func (r *mqlConfluentRoleBinding) environment() (*mqlConfluentEnvironment, error) {
	env, err := environmentByID(r.MqlRuntime, r.cachedEnvironmentID)
	if err != nil {
		return nil, err
	}
	if env == nil {
		r.Environment.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return env, nil
}

func (r *mqlConfluentRoleBinding) cluster() (*mqlConfluentKafkaCluster, error) {
	cluster, err := kafkaClusterByID(r.MqlRuntime, r.cachedClusterID)
	if err != nil {
		return nil, err
	}
	if cluster == nil {
		r.Cluster.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return cluster, nil
}
