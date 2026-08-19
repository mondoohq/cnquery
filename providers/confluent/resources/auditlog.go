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

// auditLogEnabled reports whether audit logging is switched on, and whether the
// payload answered the question at all.
//
// Confluent marks a disabled audit log by leaving the writing service account
// unset, so the service account is what separates "off" from "on" once a block
// is present. A payload carrying no audit log block has not answered: the
// endpoint this is read from is not part of the versioned management API and
// its shape is not guaranteed, so an absent block means the question could not
// be put rather than that the answer is no.
//
// Reporting "off" for an unanswered question would turn an organization that
// is audited into a clean pass, which is the failure direction that matters,
// so the caller reports the field as null instead.
func auditLogEnabled(record *auditLogRecord) (enabled bool, known bool) {
	if record == nil {
		return false, false
	}
	return record.ServiceAccountID != 0 || record.ServiceAccountResourceID != "", true
}

func (r *mqlConfluent) auditLog() (*mqlConfluentAuditLogConfig, error) {
	conn, err := confluentConn(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	var payload accountPayload
	// A failure here is reported rather than degraded, in either direction. An
	// organization whose audit log could not be read is not an organization
	// without one, and a 401 or 403 says the caller may not look rather than
	// that there is nothing to see. Turning either into `enabled: false` would
	// report a clean audit pass on an organization nobody checked.
	if err := conn.Get(context.Background(), conn.CloudTarget(), auditLogPath, nil, &payload); err != nil {
		return nil, err
	}

	record := payload.auditLog()
	enabled, known := auditLogEnabled(record)

	// An answered question is reported as it was answered; an unanswered one is
	// reported as null. llx.NilData sets the field to StateIsSet|StateIsNull,
	// which is what tells the runtime the field was resolved and holds nothing,
	// rather than leaving it unset or claiming a value the API never gave.
	enabledData := llx.NilData
	topicNameData := llx.NilData
	if known {
		enabledData = llx.BoolData(enabled)
		topicNameData = llx.StringData(record.TopicName)
	}
	if record == nil {
		record = &auditLogRecord{}
	}

	res, err := CreateResource(r.MqlRuntime, "confluent.auditLogConfig", map[string]*llx.RawData{
		"__id":      llx.StringData(connection.NewConfluentOrgIdentifier(conn.OrganizationID()) + "/auditLog"),
		"enabled":   enabledData,
		"topicName": topicNameData,
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
