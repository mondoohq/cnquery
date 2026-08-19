// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"strings"

	byokv1 "github.com/confluentinc/ccloud-sdk-go-v2/byok/v1"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/confluent/connection"
	"go.mondoo.com/mql/v13/types"
)

// encryptionKeyDetailRecord is the cloud-specific half of a self-managed key.
// The API discriminates it by `kind`, and the three shapes share no field of
// conflicting type, so one struct decodes all of them.
//
// It is also the sidecar for the SDK's discriminated union, which is why it
// survives a kind the union does not recognize. See encryptionKeyDetailOf.
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

// encryptionKeyValidationRecord is the validation block of a self-managed key.
//
// It stays local rather than adopting ByokV1KeyValidation, which types `since`
// as a bare time.Time. An absent validation timestamp would then decode to the
// zero time and report 1 January year 1 as the moment the key was last checked,
// which is an invented value where the schema promises null.
type encryptionKeyValidationRecord struct {
	Phase   string        `json:"phase"`
	Message string        `json:"message"`
	Since   confluentTime `json:"since"`
	Region  string        `json:"region"`
}

type encryptionKeyRecord struct {
	ID          string                         `json:"id"`
	Metadata    objectMeta                     `json:"metadata"`
	Key         *byokv1.ByokV1KeyKeyOneOf      `json:"key"`
	DisplayName string                         `json:"display_name"`
	Provider    string                         `json:"provider"`
	State       string                         `json:"state"`
	Validation  *encryptionKeyValidationRecord `json:"validation"`

	// keySidecar holds the key block as it arrived, for the same reason the
	// Kafka cluster keeps one: the union reports no error when it recognizes
	// nothing.
	keySidecar *encryptionKeyDetailRecord
}

// UnmarshalJSON decodes a self-managed key and keeps a second, untyped reading
// of the cloud-specific key block.
func (r *encryptionKeyRecord) UnmarshalJSON(data []byte) error {
	type alias encryptionKeyRecord
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*r = encryptionKeyRecord(decoded)

	var probe struct {
		Key json.RawMessage `json:"key"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return err
	}
	if len(probe.Key) == 0 {
		return nil
	}

	var raw encryptionKeyDetailRecord
	if err := json.Unmarshal(probe.Key, &raw); err != nil {
		log.Warn().Err(err).Msg("confluent> could not read the encryption key block")
		return nil
	}
	r.keySidecar = &raw
	return nil
}

// encryptionKeyDetailOf resolves the cloud-specific key block.
//
// ByokV1KeyKeyOneOf.UnmarshalJSON carries the same shape as the Kafka cluster's
// config union: it switches on `kind` and, matching none of AwsKey, AzureKey or
// GcpKey, returns nil with every variant nil and no error. A fourth cloud would
// blank the key reference on every key it holds, so the sidecar answers instead
// and says so.
func encryptionKeyDetailOf(record *encryptionKeyRecord) *encryptionKeyDetailRecord {
	if record == nil {
		return &encryptionKeyDetailRecord{}
	}

	if record.Key != nil {
		switch {
		case record.Key.ByokV1AwsKey != nil:
			key := record.Key.ByokV1AwsKey
			return &encryptionKeyDetailRecord{
				Kind:   key.Kind,
				KeyArn: key.KeyArn,
				Roles:  key.GetRoles(),
			}
		case record.Key.ByokV1AzureKey != nil:
			key := record.Key.ByokV1AzureKey
			return &encryptionKeyDetailRecord{
				Kind:          key.Kind,
				KeyID:         key.KeyId,
				KeyVaultID:    key.KeyVaultId,
				TenantID:      key.TenantId,
				ApplicationID: key.GetApplicationId(),
			}
		case record.Key.ByokV1GcpKey != nil:
			key := record.Key.ByokV1GcpKey
			return &encryptionKeyDetailRecord{
				Kind:          key.Kind,
				KeyID:         key.KeyId,
				SecurityGroup: key.GetSecurityGroup(),
			}
		}

		kind := ""
		if record.keySidecar != nil {
			kind = record.keySidecar.Kind
		}
		log.Warn().Str("kind", kind).
			Msg("confluent> unrecognized encryption key type; reading the key block as it arrived")
	}

	if record.keySidecar != nil {
		return record.keySidecar
	}
	return &encryptionKeyDetailRecord{}
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
		detail := encryptionKeyDetailOf(&record)
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
