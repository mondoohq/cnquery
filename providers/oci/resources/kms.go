// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/keymanagement"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v12/providers/oci/connection"
)

func (o *mqlOciKms) id() (string, error) {
	return "oci.kms", nil
}

func (o *mqlOciKms) vaults() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getVaults(conn), 5)
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

func (o *mqlOciKms) getVaults(conn *connection.OciConnection) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.GetRegions(ctx)
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci kms with region %s", *region.RegionKey)

			svc, err := conn.KmsVaultClient(*region.RegionKey)
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
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
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
			"id":             llx.StringDataPtr(key.Id),
			"name":           llx.StringDataPtr(key.DisplayName),
			"compartmentID":  llx.StringDataPtr(key.CompartmentId),
			"vaultId":        llx.StringDataPtr(key.VaultId),
			"algorithm":      llx.StringData(algorithm),
			"protectionMode": llx.StringData(string(key.ProtectionMode)),
			"state":          llx.StringData(string(key.LifecycleState)),
			"created":        llx.TimeDataPtr(created),
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

func (o *mqlOciKmsKey) id() (string, error) {
	return "oci.kms.key/" + o.Id.Data, nil
}
