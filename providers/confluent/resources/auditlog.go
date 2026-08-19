// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/confluent/connection"
)

// auditLogPath is the endpoint that reports the organization's audit log
// destination. Confluent exposes it on the account endpoint rather than in one
// of the versioned management APIs, which is also where the Confluent CLI reads
// it for `confluent audit-log describe`.
const auditLogPath = "/api/me"

// auditLogRecord is the audit log block of the account payload. The service
// account appears twice: as a numeric legacy identifier and as the modern
// resource identifier, and which of the two is populated depends on the age of
// the organization.
type auditLogRecord struct {
	ClusterID                string `json:"cluster_id"`
	AccountID                string `json:"account_id"`
	TopicName                string `json:"topic_name"`
	ServiceAccountID         int64  `json:"service_account_id"`
	ServiceAccountResourceID string `json:"service_account_resource_id"`
}

// accountPayload is the part of the account endpoint's answer this provider
// reads. The organization block appears at the top level on some responses and
// nested under the user on others, so both positions are decoded and the first
// populated one wins.
type accountPayload struct {
	Organization *struct {
		AuditLog *auditLogRecord `json:"audit_log"`
	} `json:"organization"`
	User *struct {
		Organization *struct {
			AuditLog *auditLogRecord `json:"audit_log"`
		} `json:"organization"`
	} `json:"user"`
}

// auditLog returns the audit log block of an account payload, or nil when the
// payload carries none.
func (p *accountPayload) auditLog() *auditLogRecord {
	if p == nil {
		return nil
	}
	if p.Organization != nil && p.Organization.AuditLog != nil {
		return p.Organization.AuditLog
	}
	if p.User != nil && p.User.Organization != nil && p.User.Organization.AuditLog != nil {
		return p.User.Organization.AuditLog
	}
	return nil
}

// auditLogEnabled reports whether audit logging is switched on. Confluent
// answers with an audit log block on every organization and marks a disabled
// one by leaving the writing service account unset, so the service account is
// what tells the two apart rather than the presence of the block.
func auditLogEnabled(record *auditLogRecord) bool {
	if record == nil {
		return false
	}
	return record.ServiceAccountID != 0 || record.ServiceAccountResourceID != ""
}

func (r *mqlConfluent) auditLog() (*mqlConfluentAuditLogConfig, error) {
	conn, err := confluentConn(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	var payload accountPayload
	// A failure here is reported rather than degraded. An organization whose
	// audit log could not be read is not an organization without one, and
	// answering false would turn a permission or transport problem into a clean
	// audit pass.
	if err := conn.Get(context.Background(), conn.CloudTarget(), auditLogPath, nil, &payload); err != nil {
		return nil, err
	}

	record := payload.auditLog()
	enabled := auditLogEnabled(record)
	if record == nil {
		record = &auditLogRecord{}
	}

	res, err := CreateResource(r.MqlRuntime, "confluent.auditLogConfig", map[string]*llx.RawData{
		"__id":      llx.StringData(connection.NewConfluentOrgIdentifier(conn.OrganizationID()) + "/auditLog"),
		"enabled":   llx.BoolData(enabled),
		"topicName": llx.StringData(record.TopicName),
	})
	if err != nil {
		return nil, err
	}

	config := res.(*mqlConfluentAuditLogConfig)
	config.cachedClusterID = record.ClusterID
	// Modern organizations name the environment; older ones carry a numeric
	// account identifier that no environment listing resolves.
	if strings.HasPrefix(record.AccountID, "env-") {
		config.cachedEnvironmentID = record.AccountID
	}
	config.cachedServiceAccountID = record.ServiceAccountResourceID
	return config, nil
}

// mqlConfluentAuditLogConfigInternal caches the three identifiers the audit log
// block carries.
type mqlConfluentAuditLogConfigInternal struct {
	cachedClusterID        string
	cachedEnvironmentID    string
	cachedServiceAccountID string
}

func (r *mqlConfluentAuditLogConfig) cluster() (*mqlConfluentKafkaCluster, error) {
	cluster, err := kafkaClusterByID(r.MqlRuntime, r.cachedClusterID)
	if err != nil {
		return nil, err
	}
	if cluster == nil {
		// The audit log cluster lives in an environment that a scoped API key
		// may not list, so the identifier can be known while the object is not.
		r.Cluster.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return cluster, nil
}

func (r *mqlConfluentAuditLogConfig) environment() (*mqlConfluentEnvironment, error) {
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

func (r *mqlConfluentAuditLogConfig) serviceAccount() (*mqlConfluentServiceAccount, error) {
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
