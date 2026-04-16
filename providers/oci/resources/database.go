// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/database"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciDatabase) id() (string, error) {
	return "oci.database", nil
}

// DB Systems

func (o *mqlOciDatabase) dbSystems() ([]any, error) {
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
	poolOfJobs := jobpool.CreatePool(o.getDbSystems(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciDatabase) getDbSystems(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci DB systems with region %s", regionResource.Id.Data)

			svc, err := conn.DatabaseClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var items []database.DbSystemSummary
			var page *string
			for {
				response, err := svc.ListDbSystems(ctx, database.ListDbSystemsRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				items = append(items, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range items {
				s := items[i]

				var created *time.Time
				if s.TimeCreated != nil {
					created = &s.TimeCreated.Time
				}

				freeformTags := make(map[string]interface{}, len(s.FreeformTags))
				for k, v := range s.FreeformTags {
					freeformTags[k] = v
				}
				definedTags := make(map[string]interface{}, len(s.DefinedTags))
				for k, v := range s.DefinedTags {
					definedTags[k] = v
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.database.dbSystem", map[string]*llx.RawData{
					"id":                   llx.StringDataPtr(s.Id),
					"name":                 llx.StringDataPtr(s.DisplayName),
					"compartmentID":        llx.StringDataPtr(s.CompartmentId),
					"availabilityDomain":   llx.StringDataPtr(s.AvailabilityDomain),
					"shape":                llx.StringDataPtr(s.Shape),
					"databaseEdition":      llx.StringData(string(s.DatabaseEdition)),
					"diskRedundancy":       llx.StringData(string(s.DiskRedundancy)),
					"hostname":             llx.StringDataPtr(s.Hostname),
					"domain":               llx.StringDataPtr(s.Domain),
					"version":              llx.StringDataPtr(s.Version),
					"cpuCoreCount":         llx.IntData(intValue(s.CpuCoreCount)),
					"nodeCount":            llx.IntData(intValue(s.NodeCount)),
					"dataStorageSizeInGBs": llx.IntData(intValue(s.DataStorageSizeInGBs)),
					"licenseModel":         llx.StringData(string(s.LicenseModel)),
					"nsgIds":               llx.ArrayData(convert.SliceAnyToInterface(s.NsgIds), types.String),
					"backupNetworkNsgIds":  llx.ArrayData(convert.SliceAnyToInterface(s.BackupNetworkNsgIds), types.String),
					"state":                llx.StringData(string(s.LifecycleState)),
					"created":              llx.TimeDataPtr(created),
					"freeformTags":         llx.MapData(freeformTags, types.String),
					"definedTags":          llx.MapData(definedTags, types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlDb := mqlInstance.(*mqlOciDatabaseDbSystem)
				mqlDb.cacheKmsKeyId = stringValue(s.KmsKeyId)
				mqlDb.cacheSubnetId = stringValue(s.SubnetId)
				res = append(res, mqlDb)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciDatabaseDbSystemInternal struct {
	cacheKmsKeyId string
	cacheSubnetId string
}

func (o *mqlOciDatabaseDbSystem) id() (string, error) {
	return "oci.database.dbSystem/" + o.Id.Data, nil
}

func (o *mqlOciDatabaseDbSystem) kmsKey() (*mqlOciKmsKey, error) {
	if o.cacheKmsKeyId == "" || !isOcid(o.cacheKmsKeyId) {
		o.KmsKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.kms.key", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheKmsKeyId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciKmsKey), nil
}

func (o *mqlOciDatabaseDbSystem) subnet() (*mqlOciNetworkSubnet, error) {
	if o.cacheSubnetId == "" {
		o.Subnet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.subnet", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheSubnetId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkSubnet), nil
}

// Autonomous Databases

func (o *mqlOciDatabase) autonomousDatabases() ([]any, error) {
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
	poolOfJobs := jobpool.CreatePool(o.getAutonomousDatabases(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciDatabase) getAutonomousDatabases(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci autonomous databases with region %s", regionResource.Id.Data)

			svc, err := conn.DatabaseClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var items []database.AutonomousDatabaseSummary
			var page *string
			for {
				response, err := svc.ListAutonomousDatabases(ctx, database.ListAutonomousDatabasesRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				items = append(items, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range items {
				a := items[i]

				var created *time.Time
				if a.TimeCreated != nil {
					created = &a.TimeCreated.Time
				}

				freeformTags := make(map[string]interface{}, len(a.FreeformTags))
				for k, v := range a.FreeformTags {
					freeformTags[k] = v
				}
				definedTags := make(map[string]interface{}, len(a.DefinedTags))
				for k, v := range a.DefinedTags {
					definedTags[k] = v
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.database.autonomousDatabase", map[string]*llx.RawData{
					"id":                       llx.StringDataPtr(a.Id),
					"name":                     llx.StringDataPtr(a.DisplayName),
					"compartmentID":            llx.StringDataPtr(a.CompartmentId),
					"dbName":                   llx.StringDataPtr(a.DbName),
					"dbVersion":                llx.StringDataPtr(a.DbVersion),
					"dbWorkload":               llx.StringData(string(a.DbWorkload)),
					"isDedicated":              llx.BoolDataPtr(a.IsDedicated),
					"isFreeTier":               llx.BoolDataPtr(a.IsFreeTier),
					"cpuCoreCount":             llx.IntData(intValue(a.CpuCoreCount)),
					"dataStorageSizeInTBs":     llx.IntData(intValue(a.DataStorageSizeInTBs)),
					"isMtlsConnectionRequired": llx.BoolDataPtr(a.IsMtlsConnectionRequired),
					"isAccessControlEnabled":   llx.BoolDataPtr(a.IsAccessControlEnabled),
					"whitelistedIps":           llx.ArrayData(convert.SliceAnyToInterface(a.WhitelistedIps), types.String),
					"isAutoScalingEnabled":     llx.BoolDataPtr(a.IsAutoScalingEnabled),
					"isLocalDataGuardEnabled":  llx.BoolDataPtr(a.IsLocalDataGuardEnabled),
					"dataSafeStatus":           llx.StringData(string(a.DataSafeStatus)),
					"openMode":                 llx.StringData(string(a.OpenMode)),
					"permissionLevel":          llx.StringData(string(a.PermissionLevel)),
					"licenseModel":             llx.StringData(string(a.LicenseModel)),
					"nsgIds":                   llx.ArrayData(convert.SliceAnyToInterface(a.NsgIds), types.String),
					"privateEndpointIp":        llx.StringDataPtr(a.PrivateEndpointIp),
					"privateEndpointLabel":     llx.StringDataPtr(a.PrivateEndpointLabel),
					"state":                    llx.StringData(string(a.LifecycleState)),
					"created":                  llx.TimeDataPtr(created),
					"freeformTags":             llx.MapData(freeformTags, types.String),
					"definedTags":              llx.MapData(definedTags, types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlAdb := mqlInstance.(*mqlOciDatabaseAutonomousDatabase)
				mqlAdb.cacheKmsKeyId = stringValue(a.KmsKeyId)
				mqlAdb.cacheVaultId = stringValue(a.VaultId)
				mqlAdb.cacheSubnetId = stringValue(a.SubnetId)
				res = append(res, mqlAdb)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciDatabaseAutonomousDatabaseInternal struct {
	cacheKmsKeyId string
	cacheVaultId  string
	cacheSubnetId string
}

func (o *mqlOciDatabaseAutonomousDatabase) id() (string, error) {
	return "oci.database.autonomousDatabase/" + o.Id.Data, nil
}

func (o *mqlOciDatabaseAutonomousDatabase) kmsKey() (*mqlOciKmsKey, error) {
	if o.cacheKmsKeyId == "" || !isOcid(o.cacheKmsKeyId) {
		o.KmsKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.kms.key", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheKmsKeyId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciKmsKey), nil
}

func (o *mqlOciDatabaseAutonomousDatabase) kmsVault() (*mqlOciKmsVault, error) {
	if o.cacheVaultId == "" {
		o.KmsVault.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.kms.vault", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheVaultId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciKmsVault), nil
}

func (o *mqlOciDatabaseAutonomousDatabase) subnet() (*mqlOciNetworkSubnet, error) {
	if o.cacheSubnetId == "" {
		o.Subnet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.subnet", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheSubnetId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkSubnet), nil
}
