// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/vault"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
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
