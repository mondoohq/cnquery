// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/vault"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciVault) id() (string, error) {
	return "oci.vault", nil
}

func (o *mqlOciVault) secrets() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	// Secrets sit in the compartment of the application that consumes them, and
	// ListSecrets accepts one compartment with no recursion. Asking only about
	// the tenancy root left rotation status, expiry, and auto-generation checks
	// with nothing to evaluate, which looked like a tenancy with no secrets
	// rather than a scan that never reached them.
	return ociCollect(o.MqlRuntime, ociScopeAllCompartments,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			log.Debug().Msgf("calling oci vault secrets with region %s", region)

			svc, err := conn.VaultsClient(region)
			if err != nil {
				return nil, err
			}

			secrets, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]vault.SecretSummary, *string, error) {
				response, err := svc.ListSecrets(ctx, vault.ListSecretsRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
				})
				if err != nil {
					return nil, nil, err
				}
				return response.Items, response.OpcNextPage, nil
			})
			if err != nil {
				return nil, err
			}

			var res []any
			for i := range secrets {
				s := secrets[i]

				if conn.Filters.IsFilteredOutByTags(s.FreeformTags, s.DefinedTags) {
					continue
				}

				var created *time.Time
				if s.TimeCreated != nil {
					created = &s.TimeCreated.Time
				}
				var lastRotation *time.Time
				if s.LastRotationTime != nil {
					lastRotation = &s.LastRotationTime.Time
				}
				var nextRotation *time.Time
				if s.NextRotationTime != nil {
					nextRotation = &s.NextRotationTime.Time
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.vault.secret", map[string]*llx.RawData{
					"id":                      llx.StringDataPtr(s.Id),
					"name":                    llx.StringDataPtr(s.SecretName),
					"compartmentID":           llx.StringDataPtr(s.CompartmentId),
					"description":             llx.StringDataPtr(s.Description),
					"state":                   llx.StringData(string(s.LifecycleState)),
					"rotationStatus":          llx.StringData(string(s.RotationStatus)),
					"lastRotationTime":        llx.TimeDataPtr(lastRotation),
					"nextRotationTime":        llx.TimeDataPtr(nextRotation),
					"isAutoGenerationEnabled": llx.BoolDataPtr(s.IsAutoGenerationEnabled),
					"created":                 llx.TimeDataPtr(created),
					"freeformTags":            llx.MapData(strMapToAny(s.FreeformTags), types.String),
					"definedTags":             llx.MapData(definedTagsToAny(s.DefinedTags), types.Any),
					"systemTags":              llx.MapData(definedTagsToAny(s.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				mqlS := mqlInstance.(*mqlOciVaultSecret)
				mqlS.cacheKeyID = stringValue(s.KeyId)
				mqlS.cacheVaultID = stringValue(s.VaultId)
				mqlS.cacheRegion = region
				if s.RotationConfig != nil {
					mqlS.cacheRotationInterval = stringValue(s.RotationConfig.RotationInterval)
					mqlS.cacheIsScheduledRotationEnabled = s.RotationConfig.IsScheduledRotationEnabled
				}
				if s.TimeOfCurrentVersionExpiry != nil {
					t := s.TimeOfCurrentVersionExpiry.Time
					mqlS.cacheTimeOfCurrentVersionExpiry = &t
				}
				res = append(res, mqlS)
			}

			return res, nil
		})
}

type mqlOciVaultSecretInternal struct {
	cacheKeyID   string
	cacheVaultID string
	cacheRegion  string

	// Rotation config fields from SecretSummary.RotationConfig
	cacheRotationInterval           string
	cacheIsScheduledRotationEnabled *bool
	cacheTimeOfCurrentVersionExpiry *time.Time
}

func (o *mqlOciVaultSecret) id() (string, error) {
	return "oci.vault.secret/" + o.Id.Data, nil
}

// initOciVaultSecret resolves a single secret from the scan asset's PlatformId
// when policies reference `oci.vault.secret` on a discovered oci-vault-secret
// asset. Explicit id takes precedence.
func initOciVaultSecret(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	idVal := ociArgString(args, "id")
	if idVal == "" {
		conn := runtime.Connection.(*connection.OciConnection)
		if conn.Conf == nil || conn.Conf.PlatformId == "" {
			return args, nil, nil
		}
		parsed, ok := parseOciObjectPlatformID(conn.Conf.PlatformId)
		if !ok || parsed.service != "vault" || parsed.objectType != "secret" {
			return args, nil, nil
		}
		idVal = parsed.id
	}

	obj, err := CreateResource(runtime, "oci.vault", nil)
	if err != nil {
		return nil, nil, err
	}
	vaultSvc := obj.(*mqlOciVault)

	secrets := vaultSvc.GetSecrets()
	if secrets.Error != nil {
		return nil, nil, secrets.Error
	}

	for _, raw := range secrets.Data {
		s := raw.(*mqlOciVaultSecret)
		if s.Id.Data == idVal {
			return args, s, nil
		}
	}

	return nil, nil, errors.New("oci.vault.secret not found: " + idVal)
}

func (o *mqlOciVaultSecret) kmsVault() (*mqlOciKmsVault, error) {
	if o.cacheVaultID == "" {
		o.KmsVault.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlVault, err := NewResource(o.MqlRuntime, "oci.kms.vault", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheVaultID),
	})
	if err != nil {
		return nil, err
	}
	return mqlVault.(*mqlOciKmsVault), nil
}

func (o *mqlOciVaultSecret) kmsKey() (*mqlOciKmsKey, error) {
	if o.cacheKeyID == "" {
		o.KmsKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlKey, err := NewResource(o.MqlRuntime, "oci.kms.key", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheKeyID),
	})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlOciKmsKey), nil
}

func (o *mqlOciVaultSecret) rotationInterval() (string, error) {
	return o.cacheRotationInterval, nil
}

func (o *mqlOciVaultSecret) isScheduledRotationEnabled() (bool, error) {
	return boolValue(o.cacheIsScheduledRotationEnabled), nil
}

func (o *mqlOciVaultSecret) timeOfCurrentVersionExpiry() (*time.Time, error) {
	if o.cacheTimeOfCurrentVersionExpiry == nil {
		o.TimeOfCurrentVersionExpiry.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return o.cacheTimeOfCurrentVersionExpiry, nil
}

func (o *mqlOciVaultSecret) secretVersions() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	ctx := context.Background()

	svc, err := conn.VaultsClient(o.cacheRegion)
	if err != nil {
		return nil, err
	}

	versions, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]vault.SecretVersionSummary, *string, error) {
		response, err := svc.ListSecretVersions(ctx, vault.ListSecretVersionsRequest{
			SecretId: common.String(o.Id.Data),
			Page:     page,
		})
		if err != nil {
			return nil, nil, err
		}
		return response.Items, response.OpcNextPage, nil
	})
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(versions))
	for i := range versions {
		v := versions[i]

		var created *time.Time
		if v.TimeCreated != nil {
			created = &v.TimeCreated.Time
		}
		var timeOfExpiry *time.Time
		if v.TimeOfExpiry != nil {
			timeOfExpiry = &v.TimeOfExpiry.Time
		}
		var timeOfDeletion *time.Time
		if v.TimeOfDeletion != nil {
			timeOfDeletion = &v.TimeOfDeletion.Time
		}

		stages := make([]any, 0, len(v.Stages))
		for _, s := range v.Stages {
			stages = append(stages, string(s))
		}

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.vault.secretVersion", map[string]*llx.RawData{
			"secretId":               llx.StringDataPtr(v.SecretId),
			"versionNumber":          llx.IntData(int64Value(v.VersionNumber)),
			"name":                   llx.StringDataPtr(v.Name),
			"contentType":            llx.StringData(string(v.ContentType)),
			"stages":                 llx.ArrayData(stages, types.String),
			"isContentAutoGenerated": llx.BoolDataPtr(v.IsContentAutoGenerated),
			"created":                llx.TimeDataPtr(created),
			"timeOfExpiry":           llx.TimeDataPtr(timeOfExpiry),
			"timeOfDeletion":         llx.TimeDataPtr(timeOfDeletion),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (o *mqlOciVaultSecretVersion) id() (string, error) {
	return "oci.vault.secretVersion/" + o.SecretId.Data + "/" + strconv.FormatInt(o.VersionNumber.Data, 10), nil
}
