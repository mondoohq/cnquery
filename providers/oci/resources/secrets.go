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
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciVault) id() (string, error) {
	return "oci.vault", nil
}

func (o *mqlOciVault) secrets() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getSecrets(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciVault) getSecrets(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci vault secrets with region %s", regionResource.Id.Data)

			svc, err := conn.VaultsClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			secrets := []vault.SecretSummary{}
			var page *string
			for {
				response, err := svc.ListSecrets(ctx, vault.ListSecretsRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				secrets = append(secrets, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range secrets {
				s := secrets[i]

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
				})
				if err != nil {
					return nil, err
				}
				mqlS := mqlInstance.(*mqlOciVaultSecret)
				mqlS.cacheKeyId = stringValue(s.KeyId)
				mqlS.cacheVaultId = stringValue(s.VaultId)
				mqlS.cacheRegion = regionResource.Id.Data
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

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciVaultSecretInternal struct {
	cacheKeyId   string
	cacheVaultId string
	cacheRegion  string

	// Rotation config fields from SecretSummary.RotationConfig
	cacheRotationInterval           string
	cacheIsScheduledRotationEnabled *bool
	cacheTimeOfCurrentVersionExpiry *time.Time
}

func (o *mqlOciVaultSecret) id() (string, error) {
	return "oci.vault.secret/" + o.Id.Data, nil
}

func (o *mqlOciVaultSecret) kmsVault() (*mqlOciKmsVault, error) {
	if o.cacheVaultId == "" {
		o.KmsVault.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlVault, err := NewResource(o.MqlRuntime, "oci.kms.vault", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheVaultId),
	})
	if err != nil {
		return nil, err
	}
	return mqlVault.(*mqlOciKmsVault), nil
}

func (o *mqlOciVaultSecret) kmsKey() (*mqlOciKmsKey, error) {
	if o.cacheKeyId == "" {
		o.KmsKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlKey, err := NewResource(o.MqlRuntime, "oci.kms.key", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheKeyId),
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

	versions := []vault.SecretVersionSummary{}
	var page *string
	for {
		response, err := svc.ListSecretVersions(ctx, vault.ListSecretVersionsRequest{
			SecretId: common.String(o.Id.Data),
			Page:     page,
		})
		if err != nil {
			return nil, err
		}

		versions = append(versions, response.Items...)

		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
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
