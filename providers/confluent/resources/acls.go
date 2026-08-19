// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"net/url"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/confluent/connection"
)

// mqlConfluentAclInternal caches the cluster the entry belongs to and the
// service account its principal names, both of which are identifiers on the
// payload rather than objects.
type mqlConfluentAclInternal struct {
	cachedClusterID        string
	cachedServiceAccountID string
}

// aclRecord is one entry of a cluster's access control listing.
//
// This stays local, and `ResourceType` deliberately stays a string, rather than
// adopting kafkarest's AclData and its AclResourceType. That enum admits seven
// values (UNKNOWN, ANY, TOPIC, GROUP, CLUSTER, TRANSACTIONAL_ID and
// DELEGATION_TOKEN) and returns an error for anything else. Kafka's own USER
// resource type is not among them, so a single such entry would fail the decode
// of the page it sits on and take an entire cluster's ACL listing with it,
// reporting no access control entries at all on a cluster that has them.
type aclRecord struct {
	ClusterID    string `json:"cluster_id"`
	ResourceType string `json:"resource_type"`
	ResourceName string `json:"resource_name"`
	PatternType  string `json:"pattern_type"`
	Principal    string `json:"principal"`
	Host         string `json:"host"`
	Operation    string `json:"operation"`
	Permission   string `json:"permission"`
}

// aclID builds the cache key of an access control entry. Kafka gives an entry
// no identifier, and the whole tuple is its identity: two entries differing
// only in operation or in permission are separate grants, and collapsing them
// would report one where there are two.
//
// Every component is escaped, because a consumer group or transactional ID may
// contain the separator and would otherwise shift the remaining components.
func aclID(record *aclRecord) string {
	parts := []string{
		record.ClusterID,
		record.Permission,
		record.Principal,
		record.Host,
		record.ResourceType,
		record.PatternType,
		record.ResourceName,
		record.Operation,
	}
	escaped := make([]string, len(parts))
	for i, part := range parts {
		escaped[i] = url.QueryEscape(part)
	}
	return strings.Join(escaped, "/")
}

// aclGrantsWildcardResource reports whether an entry covers every resource of
// its type rather than a named one.
//
// A LITERAL pattern of "*" is Kafka's own spelling of "every resource". A
// PREFIXED pattern with an empty prefix matches every name, which is the same
// reach written differently. ANY and MATCH only appear on filters, and are
// covered so an entry carrying one is not read as narrow.
func aclGrantsWildcardResource(patternType, resourceName string) bool {
	switch strings.ToUpper(patternType) {
	case "LITERAL", "ANY", "MATCH":
		return resourceName == "*"
	case "PREFIXED":
		return resourceName == ""
	default:
		return false
	}
}

// aclGrantsAnyPrincipal reports whether an entry applies to every principal
// rather than to a named one. Kafka writes that as a "*" in place of the
// principal name, in any of the principal types Confluent renders.
func aclGrantsAnyPrincipal(principal string) bool {
	trimmed := strings.TrimSpace(principal)
	if trimmed == "*" {
		return true
	}
	_, name, found := strings.Cut(trimmed, ":")
	return found && name == "*"
}

// aclGrantsAllOperations reports whether an entry carries Kafka's ALL
// operation, which covers every operation on the matched resource.
func aclGrantsAllOperations(operation string) bool {
	return strings.EqualFold(operation, "ALL")
}

// principalAccountID extracts the account identifier out of a Kafka principal.
// Confluent renders a service account as "User:sa-abc123" on some endpoints and
// as "UserV2:sa-abc123" on others; a human user appears as "User:u-abc123".
// Anything that is not one of those, including a wildcard or a numeric legacy
// identifier, yields the empty string.
func principalAccountID(principal string) string {
	kind, name, found := strings.Cut(strings.TrimSpace(principal), ":")
	if !found {
		return ""
	}
	switch strings.ToLower(kind) {
	case "user", "userv2":
	default:
		return ""
	}
	if name == "" || name == "*" {
		return ""
	}
	return name
}

// isServiceAccountID reports whether an account identifier names a service
// account rather than a human user.
func isServiceAccountID(id string) bool {
	return strings.HasPrefix(id, "sa-")
}

func (r *mqlConfluentKafkaCluster) acls() ([]any, error) {
	conn, err := confluentConn(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	clusterID := r.GetId().Data
	target, err := conn.KafkaTarget(clusterID, r.cachedRestEndpoint)
	if err != nil {
		return nil, err
	}

	records, err := connection.GetPaged[aclRecord](context.Background(), conn, target,
		"/kafka/v3/clusters/"+url.PathEscape(clusterID)+"/acls", nil)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		record := records[i]
		// The listing is scoped to one cluster, but the entries carry their own
		// cluster ID. Prefer it and fall back to the cluster being read, so an
		// entry with an empty field still gets a qualified identifier.
		if record.ClusterID == "" {
			record.ClusterID = clusterID
		}

		mqlAcl, err := CreateResource(r.MqlRuntime, "confluent.acl", map[string]*llx.RawData{
			"__id":                   llx.StringData(aclID(&record)),
			"principal":              llx.StringData(record.Principal),
			"host":                   llx.StringData(record.Host),
			"resourceType":           llx.StringData(record.ResourceType),
			"resourceName":           llx.StringData(record.ResourceName),
			"patternType":            llx.StringData(record.PatternType),
			"operation":              llx.StringData(record.Operation),
			"permission":             llx.StringData(record.Permission),
			"grantsWildcardResource": llx.BoolData(aclGrantsWildcardResource(record.PatternType, record.ResourceName)),
			"grantsAnyPrincipal":     llx.BoolData(aclGrantsAnyPrincipal(record.Principal)),
			"grantsAllOperations":    llx.BoolData(aclGrantsAllOperations(record.Operation)),
		})
		if err != nil {
			return nil, err
		}

		acl := mqlAcl.(*mqlConfluentAcl)
		acl.cachedClusterID = record.ClusterID
		if accountID := principalAccountID(record.Principal); isServiceAccountID(accountID) {
			acl.cachedServiceAccountID = accountID
		}
		res = append(res, acl)
	}
	return res, nil
}

func (r *mqlConfluentAcl) cluster() (*mqlConfluentKafkaCluster, error) {
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

func (r *mqlConfluentAcl) serviceAccount() (*mqlConfluentServiceAccount, error) {
	if r.cachedServiceAccountID == "" {
		r.ServiceAccount.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	account, err := serviceAccountByID(r.MqlRuntime, r.cachedServiceAccountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		r.ServiceAccount.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return account, nil
}
