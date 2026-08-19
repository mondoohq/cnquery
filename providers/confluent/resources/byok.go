// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/confluent/connection"
	"go.mondoo.com/mql/v13/types"
)

// encryptionKeyDetailRecord is the cloud-specific half of a self-managed key.
// The API discriminates it by `kind`, and the three shapes share no field of
// conflicting type, so one struct decodes all of them.
type encryptionKeyDetailRecord struct {
	Kind string `json:"kind"`

	// AWS
	KeyArn string   `json:"key_arn"`
	Roles  []string `json:"roles"`

	// Azure and Google Cloud
	KeyID string `json:"key_id"`

	// Azure
	KeyVaultID    string `json:"key_vault_id"`
	TenantID      string `json:"tenant_id"`
	ApplicationID string `json:"application_id"`

	// Google Cloud
	SecurityGroup string `json:"security_group"`
}

type encryptionKeyValidationRecord struct {
	Phase   string        `json:"phase"`
	Message string        `json:"message"`
	Since   confluentTime `json:"since"`
	Region  string        `json:"region"`
}

type encryptionKeyRecord struct {
	ID          string                         `json:"id"`
	Metadata    objectMeta                     `json:"metadata"`
	Key         *encryptionKeyDetailRecord     `json:"key"`
	DisplayName string                         `json:"display_name"`
	Provider    string                         `json:"provider"`
	State       string                         `json:"state"`
	Validation  *encryptionKeyValidationRecord `json:"validation"`
}

// keyReferenceOf renders the cloud provider's own name for the key. AWS names a
// key by ARN, Azure and Google Cloud by key identifier, so which field carries
// the reference depends on the kind.
func keyReferenceOf(detail *encryptionKeyDetailRecord) string {
	if detail == nil {
		return ""
	}
	if strings.EqualFold(detail.Kind, "AwsKey") {
		return detail.KeyArn
	}
	if detail.KeyID != "" {
		return detail.KeyID
	}
	// An unrecognized kind still carries one of the two references, and
	// reporting whichever is populated beats reporting nothing.
	return detail.KeyArn
}

func (r *mqlConfluent) encryptionKeys() ([]any, error) {
	conn, err := confluentConn(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	records, err := connection.GetPaged[encryptionKeyRecord](context.Background(), conn,
		conn.CloudTarget(), "/byok/v1/keys", nil)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		record := records[i]
		detail := record.Key
		if detail == nil {
			detail = &encryptionKeyDetailRecord{}
		}
		validation := record.Validation
		if validation == nil {
			validation = &encryptionKeyValidationRecord{}
		}

		mqlKey, err := CreateResource(r.MqlRuntime, "confluent.encryptionKey", map[string]*llx.RawData{
			"__id":              llx.StringData(record.ID),
			"id":                llx.StringData(record.ID),
			"displayName":       llx.StringData(record.DisplayName),
			"resourceName":      llx.StringData(record.Metadata.ResourceName),
			"provider":          llx.StringData(record.Provider),
			"state":             llx.StringData(record.State),
			"keyReference":      llx.StringData(keyReferenceOf(detail)),
			"keyVaultId":        llx.StringData(detail.KeyVaultID),
			"tenantId":          llx.StringData(detail.TenantID),
			"applicationId":     llx.StringData(detail.ApplicationID),
			"securityGroup":     llx.StringData(detail.SecurityGroup),
			"roles":             llx.ArrayData(strSliceToAny(detail.Roles), types.String),
			"validationPhase":   llx.StringData(validation.Phase),
			"validationMessage": llx.StringData(validation.Message),
			"validationSince":   llx.TimeDataPtr(validation.Since.Time()),
			"createdAt":         llx.TimeDataPtr(record.Metadata.CreatedAt.Time()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlKey)
	}
	return res, nil
}

func (r *mqlConfluentEncryptionKey) kafkaClusters() ([]any, error) {
	root, err := getConfluent(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	clusters := root.GetKafkaClusters()
	if clusters.Error != nil {
		return nil, clusters.Error
	}

	keyID := r.GetId().Data
	res := []any{}
	for _, raw := range clusters.Data {
		cluster, ok := raw.(*mqlConfluentKafkaCluster)
		if !ok {
			continue
		}
		if cluster.cachedEncryptionKeyID == keyID {
			res = append(res, cluster)
		}
	}
	return res, nil
}

// encryptionKeyByID resolves a self-managed key from the root resource's cached
// list.
func encryptionKeyByID(runtime *plugin.Runtime, id string) (*mqlConfluentEncryptionKey, error) {
	if id == "" {
		return nil, nil
	}
	root, err := getConfluent(runtime)
	if err != nil {
		return nil, err
	}
	keys := root.GetEncryptionKeys()
	if keys.Error != nil {
		return nil, keys.Error
	}
	for _, raw := range keys.Data {
		key, ok := raw.(*mqlConfluentEncryptionKey)
		if !ok {
			continue
		}
		if key.GetId().Data == id {
			return key, nil
		}
	}
	return nil, nil
}
