// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/keymanagement"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
)

func (o *mqlOciKms) id() (string, error) {
	return "oci.kms", nil
}

func (o *mqlOciKms) vaults() ([]any, error) {
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
	poolOfJobs := jobpool.CreatePool(o.getVaults(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciKms) getVaultsForRegion(ctx context.Context, client *keymanagement.KmsVaultClient, compartmentID string) ([]keymanagement.VaultSummary, error) {
	entries := []keymanagement.VaultSummary{}
	var page *string
	for {
		request := keymanagement.ListVaultsRequest{
			CompartmentId: common.String(compartmentID),
			Page:          page,
		}

		response, err := client.ListVaults(ctx, request)
		if err != nil {
			return nil, err
		}

		entries = append(entries, response.Items...)

		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	return entries, nil
}

func (o *mqlOciKms) getVaults(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci kms with region %s", regionResource.Id.Data)

			svc, err := conn.KmsVaultClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var res []any
			vaults, err := o.getVaultsForRegion(ctx, svc, conn.TenantID())
			if err != nil {
				return nil, err
			}

			for i := range vaults {
				vault := vaults[i]

				var created *time.Time
				if vault.TimeCreated != nil {
					created = &vault.TimeCreated.Time
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.kms.vault", map[string]*llx.RawData{
					"id":                 llx.StringDataPtr(vault.Id),
					"name":               llx.StringDataPtr(vault.DisplayName),
					"compartmentID":      llx.StringDataPtr(vault.CompartmentId),
					"vaultType":          llx.StringData(string(vault.VaultType)),
					"state":              llx.StringData(string(vault.LifecycleState)),
					"managementEndpoint": llx.StringDataPtr(vault.ManagementEndpoint),
					"created":            llx.TimeDataPtr(created),
				})
				if err != nil {
					return nil, err
				}
				mqlInstance.(*mqlOciKmsVault).region = regionResource.Id.Data
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciKmsVaultInternal struct {
	region string
}

func initOciKmsVault(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch oci.kms.vault")
	}
	idVal := args["id"].Value.(string)

	obj, err := CreateResource(runtime, "oci.kms", nil)
	if err != nil {
		return nil, nil, err
	}
	kms := obj.(*mqlOciKms)

	rawVaults := kms.GetVaults()
	if rawVaults.Error != nil {
		return nil, nil, rawVaults.Error
	}

	for _, raw := range rawVaults.Data {
		vault := raw.(*mqlOciKmsVault)
		if vault.Id.Data == idVal {
			return args, vault, nil
		}
	}

	return nil, nil, errors.New("oci.kms.vault not found: " + idVal)
}

func (o *mqlOciKmsVault) id() (string, error) {
	return "oci.kms.vault/" + o.Id.Data, nil
}

func (o *mqlOciKmsVault) keys() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	ctx := context.Background()

	managementEndpoint := o.ManagementEndpoint.Data
	if managementEndpoint == "" {
		return []any{}, nil
	}

	svc, err := conn.KmsManagementClient(managementEndpoint)
	if err != nil {
		return nil, err
	}

	keys, err := o.getKeysForVault(ctx, svc, o.CompartmentID.Data)
	if err != nil {
		return nil, err
	}

	res := []any{}
	for i := range keys {
		key := keys[i]

		var created *time.Time
		if key.TimeCreated != nil {
			created = &key.TimeCreated.Time
		}

		algorithm := string(key.Algorithm)

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.kms.key", map[string]*llx.RawData{
			"id":                    llx.StringDataPtr(key.Id),
			"name":                  llx.StringDataPtr(key.DisplayName),
			"compartmentID":         llx.StringDataPtr(key.CompartmentId),
			"vaultId":               llx.StringDataPtr(key.VaultId),
			"algorithm":             llx.StringData(algorithm),
			"protectionMode":        llx.StringData(string(key.ProtectionMode)),
			"state":                 llx.StringData(string(key.LifecycleState)),
			"isAutoRotationEnabled": llx.BoolDataPtr(key.IsAutoRotationEnabled),
			"created":               llx.TimeDataPtr(created),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (o *mqlOciKmsVault) getKeysForVault(ctx context.Context, client *keymanagement.KmsManagementClient, compartmentID string) ([]keymanagement.KeySummary, error) {
	entries := []keymanagement.KeySummary{}
	var page *string
	for {
		request := keymanagement.ListKeysRequest{
			CompartmentId: common.String(compartmentID),
			Page:          page,
		}

		response, err := client.ListKeys(ctx, request)
		if err != nil {
			return nil, err
		}

		entries = append(entries, response.Items...)

		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	return entries, nil
}

func initOciKmsKey(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch oci.kms.key")
	}
	idVal := args["id"].Value.(string)

	obj, err := CreateResource(runtime, "oci.kms", nil)
	if err != nil {
		return nil, nil, err
	}
	kms := obj.(*mqlOciKms)

	rawVaults := kms.GetVaults()
	if rawVaults.Error != nil {
		return nil, nil, rawVaults.Error
	}

	for _, rawVault := range rawVaults.Data {
		vault := rawVault.(*mqlOciKmsVault)
		rawKeys := vault.GetKeys()
		if rawKeys.Error != nil {
			continue
		}
		for _, rawKey := range rawKeys.Data {
			key := rawKey.(*mqlOciKmsKey)
			if key.Id.Data == idVal {
				return args, key, nil
			}
		}
	}

	return nil, nil, errors.New("oci.kms.key not found: " + idVal)
}

func (o *mqlOciKmsKey) id() (string, error) {
	return "oci.kms.key/" + o.Id.Data, nil
}
