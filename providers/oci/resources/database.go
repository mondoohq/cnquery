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

				var connectionUrls map[string]any
				if a.ConnectionUrls != nil {
					connectionUrls, err = convert.JsonToDict(a.ConnectionUrls)
					if err != nil {
						return nil, err
					}
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
					"connectionUrls":           llx.DictData(connectionUrls),
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

// Database backups (VM/BM)

func (o *mqlOciDatabase) backups() ([]any, error) {
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
	poolOfJobs := jobpool.CreatePool(o.getDatabaseBackups(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}
	return res, nil
}

func (o *mqlOciDatabase) getDatabaseBackups(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci database backups with region %s", regionResource.Id.Data)

			svc, err := conn.DatabaseClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var items []database.BackupSummary
			var page *string
			for {
				response, err := svc.ListBackups(ctx, database.ListBackupsRequest{
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
				b := items[i]

				var started, ended, expiry *time.Time
				if b.TimeStarted != nil {
					started = &b.TimeStarted.Time
				}
				if b.TimeEnded != nil {
					ended = &b.TimeEnded.Time
				}
				if b.TimeExpiryScheduled != nil {
					expiry = &b.TimeExpiryScheduled.Time
				}

				var sizeGBs float64
				if b.DatabaseSizeInGBs != nil {
					sizeGBs = *b.DatabaseSizeInGBs
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.database.backup", map[string]*llx.RawData{
					"id":                       llx.StringDataPtr(b.Id),
					"name":                     llx.StringDataPtr(b.DisplayName),
					"compartmentID":            llx.StringDataPtr(b.CompartmentId),
					"databaseId":               llx.StringDataPtr(b.DatabaseId),
					"availabilityDomain":       llx.StringDataPtr(b.AvailabilityDomain),
					"type":                     llx.StringData(string(b.Type)),
					"backupDestinationType":    llx.StringData(string(b.BackupDestinationType)),
					"databaseSizeInGBs":        llx.FloatData(sizeGBs),
					"databaseEdition":          llx.StringData(string(b.DatabaseEdition)),
					"version":                  llx.StringDataPtr(b.Version),
					"shape":                    llx.StringDataPtr(b.Shape),
					"isUsingOracleManagedKeys": llx.BoolDataPtr(b.IsUsingOracleManagedKeys),
					"retentionPeriodInDays":    llx.IntData(intValue(b.RetentionPeriodInDays)),
					"retentionPeriodInYears":   llx.IntData(intValue(b.RetentionPeriodInYears)),
					"timeExpiryScheduled":      llx.TimeDataPtr(expiry),
					"state":                    llx.StringData(string(b.LifecycleState)),
					"timeStarted":              llx.TimeDataPtr(started),
					"timeEnded":                llx.TimeDataPtr(ended),
				})
				if err != nil {
					return nil, err
				}
				mqlBackup := mqlInstance.(*mqlOciDatabaseBackup)
				mqlBackup.cacheKmsKeyId = stringValue(b.KmsKeyId)
				mqlBackup.cacheVaultId = stringValue(b.VaultId)
				res = append(res, mqlBackup)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciDatabaseBackupInternal struct {
	cacheKmsKeyId string
	cacheVaultId  string
}

func (o *mqlOciDatabaseBackup) id() (string, error) {
	return "oci.database.backup/" + o.Id.Data, nil
}

func (o *mqlOciDatabaseBackup) kmsKey() (*mqlOciKmsKey, error) {
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

func (o *mqlOciDatabaseBackup) kmsVault() (*mqlOciKmsVault, error) {
	if o.cacheVaultId == "" || !isOcid(o.cacheVaultId) {
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

// Autonomous Database Backups

func (o *mqlOciDatabase) autonomousDatabaseBackups() ([]any, error) {
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
	poolOfJobs := jobpool.CreatePool(o.getAutonomousDatabaseBackups(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}
	return res, nil
}

// backups on an individual autonomous database returns its backups by
// filtering the service-wide listing. We rely on the tenancy-wide listing
// being cached and filter client-side, which avoids fanning out region calls
// per-database when the parent list is already fetched.
func (o *mqlOciDatabaseAutonomousDatabase) backups() ([]any, error) {
	dbObj, err := CreateResource(o.MqlRuntime, "oci.database", nil)
	if err != nil {
		return nil, err
	}
	db := dbObj.(*mqlOciDatabase)
	raw := db.GetAutonomousDatabaseBackups()
	if raw.Error != nil {
		return nil, raw.Error
	}
	dbID := o.Id.Data
	res := []any{}
	for _, r := range raw.Data {
		b := r.(*mqlOciDatabaseAutonomousDatabaseBackup)
		if b.cacheAutonomousDatabaseId == dbID {
			res = append(res, b)
		}
	}
	return res, nil
}

func (o *mqlOciDatabase) getAutonomousDatabaseBackups(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci autonomous database backups with region %s", regionResource.Id.Data)

			svc, err := conn.DatabaseClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var items []database.AutonomousDatabaseBackupSummary
			var page *string
			for {
				response, err := svc.ListAutonomousDatabaseBackups(ctx, database.ListAutonomousDatabaseBackupsRequest{
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

			res := make([]any, 0, len(items))
			for i := range items {
				b := items[i]

				var started, ended, timeTill *time.Time
				if b.TimeStarted != nil {
					started = &b.TimeStarted.Time
				}
				if b.TimeEnded != nil {
					ended = &b.TimeEnded.Time
				}
				if b.TimeAvailableTill != nil {
					timeTill = &b.TimeAvailableTill.Time
				}

				var dbSizeTBs, sizeTBs float64
				if b.DatabaseSizeInTBs != nil {
					dbSizeTBs = float64(*b.DatabaseSizeInTBs)
				}
				if b.SizeInTBs != nil {
					sizeTBs = *b.SizeInTBs
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.database.autonomousDatabaseBackup", map[string]*llx.RawData{
					"id":                    llx.StringDataPtr(b.Id),
					"name":                  llx.StringDataPtr(b.DisplayName),
					"compartmentID":         llx.StringDataPtr(b.CompartmentId),
					"type":                  llx.StringData(string(b.Type)),
					"isAutomatic":           llx.BoolDataPtr(b.IsAutomatic),
					"isRestorable":          llx.BoolDataPtr(b.IsRestorable),
					"retentionPeriodInDays": llx.IntData(intValue(b.RetentionPeriodInDays)),
					"timeAvailableTill":     llx.TimeDataPtr(timeTill),
					"databaseSizeInTBs":     llx.FloatData(dbSizeTBs),
					"sizeInTBs":             llx.FloatData(sizeTBs),
					"dbVersion":             llx.StringDataPtr(b.DbVersion),
					"infrastructureType":    llx.StringData(string(b.InfrastructureType)),
					"state":                 llx.StringData(string(b.LifecycleState)),
					"timeStarted":           llx.TimeDataPtr(started),
					"timeEnded":             llx.TimeDataPtr(ended),
				})
				if err != nil {
					return nil, err
				}
				mqlBackup := mqlInstance.(*mqlOciDatabaseAutonomousDatabaseBackup)
				mqlBackup.cacheAutonomousDatabaseId = stringValue(b.AutonomousDatabaseId)
				mqlBackup.cacheKmsKeyId = stringValue(b.KmsKeyId)
				mqlBackup.cacheVaultId = stringValue(b.VaultId)
				res = append(res, mqlBackup)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciDatabaseAutonomousDatabaseBackupInternal struct {
	cacheAutonomousDatabaseId string
	cacheKmsKeyId             string
	cacheVaultId              string
}

func (o *mqlOciDatabaseAutonomousDatabaseBackup) id() (string, error) {
	return "oci.database.autonomousDatabaseBackup/" + o.Id.Data, nil
}

func (o *mqlOciDatabaseAutonomousDatabaseBackup) autonomousDatabase() (*mqlOciDatabaseAutonomousDatabase, error) {
	if o.cacheAutonomousDatabaseId == "" {
		o.AutonomousDatabase.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.database.autonomousDatabase", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheAutonomousDatabaseId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciDatabaseAutonomousDatabase), nil
}

func (o *mqlOciDatabaseAutonomousDatabaseBackup) kmsKey() (*mqlOciKmsKey, error) {
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

func (o *mqlOciDatabaseAutonomousDatabaseBackup) kmsVault() (*mqlOciKmsVault, error) {
	if o.cacheVaultId == "" || !isOcid(o.cacheVaultId) {
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
