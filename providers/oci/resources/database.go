// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/database"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciDatabase) id() (string, error) {
	return "oci.database", nil
}

// DB Systems

func (o *mqlOciDatabase) dbSystems() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	return ociCollect(o.MqlRuntime, ociScopeTenancyRoot,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			log.Debug().Msgf("calling oci DB systems with region %s", region)

			svc, err := conn.DatabaseClient(region)
			if err != nil {
				return nil, err
			}

			items, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]database.DbSystemSummary, *string, error) {
				response, err := svc.ListDbSystems(ctx, database.ListDbSystemsRequest{
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
					"listenerPort":         llx.IntData(intValue(s.ListenerPort)),
					"scanDnsName":          llx.StringDataPtr(s.ScanDnsName),
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
					"systemTags":           llx.MapData(definedTagsToAny(s.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				mqlDb := mqlInstance.(*mqlOciDatabaseDbSystem)
				mqlDb.cacheKmsKeyID = stringValue(s.KmsKeyId)
				mqlDb.cacheSubnetID = stringValue(s.SubnetId)
				mqlDb.cacheSourceDbSystemID = stringValue(s.SourceDbSystemId)
				res = append(res, mqlDb)
			}

			return res, nil
		})
}

type mqlOciDatabaseDbSystemInternal struct {
	cacheKmsKeyID         string
	cacheSubnetID         string
	cacheSourceDbSystemID string
}

func (o *mqlOciDatabaseDbSystem) id() (string, error) {
	return "oci.database.dbSystem/" + o.Id.Data, nil
}

func (o *mqlOciDatabaseDbSystem) kmsKey() (*mqlOciKmsKey, error) {
	if o.cacheKmsKeyID == "" || !isOcid(o.cacheKmsKeyID) {
		o.KmsKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.kms.key", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheKmsKeyID),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciKmsKey), nil
}

func (o *mqlOciDatabaseDbSystem) subnet() (*mqlOciNetworkSubnet, error) {
	if o.cacheSubnetID == "" || !isOcid(o.cacheSubnetID) {
		o.Subnet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.subnet", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheSubnetID),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkSubnet), nil
}

// Autonomous Databases

func (o *mqlOciDatabase) autonomousDatabases() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	return ociCollect(o.MqlRuntime, ociScopeTenancyRoot,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			log.Debug().Msgf("calling oci autonomous databases with region %s", region)

			svc, err := conn.DatabaseClient(region)
			if err != nil {
				return nil, err
			}

			items, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]database.AutonomousDatabaseSummary, *string, error) {
				response, err := svc.ListAutonomousDatabases(ctx, database.ListAutonomousDatabasesRequest{
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

				var publicConnectionUrls map[string]any
				if a.PublicConnectionUrls != nil {
					publicConnectionUrls, err = convert.JsonToDict(a.PublicConnectionUrls)
					if err != nil {
						return nil, err
					}
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.database.autonomousDatabase", map[string]*llx.RawData{
					"id":                          llx.StringDataPtr(a.Id),
					"name":                        llx.StringDataPtr(a.DisplayName),
					"compartmentID":               llx.StringDataPtr(a.CompartmentId),
					"dbName":                      llx.StringDataPtr(a.DbName),
					"isRefreshableClone":          llx.BoolDataPtr(a.IsRefreshableClone),
					"dbVersion":                   llx.StringDataPtr(a.DbVersion),
					"dbWorkload":                  llx.StringData(string(a.DbWorkload)),
					"isDedicated":                 llx.BoolDataPtr(a.IsDedicated),
					"isFreeTier":                  llx.BoolDataPtr(a.IsFreeTier),
					"cpuCoreCount":                llx.IntData(intValue(a.CpuCoreCount)),
					"dataStorageSizeInTBs":        llx.IntData(intValue(a.DataStorageSizeInTBs)),
					"isMtlsConnectionRequired":    llx.BoolDataPtr(a.IsMtlsConnectionRequired),
					"isAccessControlEnabled":      llx.BoolDataPtr(a.IsAccessControlEnabled),
					"whitelistedIps":              llx.ArrayData(convert.SliceAnyToInterface(a.WhitelistedIps), types.String),
					"standbyWhitelistedIps":       llx.ArrayData(convert.SliceAnyToInterface(a.StandbyWhitelistedIps), types.String),
					"isAutoScalingEnabled":        llx.BoolDataPtr(a.IsAutoScalingEnabled),
					"isLocalDataGuardEnabled":     llx.BoolDataPtr(a.IsLocalDataGuardEnabled),
					"isRemoteDataGuardEnabled":    llx.BoolDataPtr(a.IsRemoteDataGuardEnabled),
					"isDataGuardEnabled":          llx.BoolDataPtr(a.IsDataGuardEnabled),
					"backupRetentionPeriodInDays": llx.IntData(intValue(a.BackupRetentionPeriodInDays)),
					"isBackupRetentionLocked":     llx.BoolDataPtr(a.IsBackupRetentionLocked),
					"dataSafeStatus":              llx.StringData(string(a.DataSafeStatus)),
					"openMode":                    llx.StringData(string(a.OpenMode)),
					"permissionLevel":             llx.StringData(string(a.PermissionLevel)),
					"licenseModel":                llx.StringData(string(a.LicenseModel)),
					"nsgIds":                      llx.ArrayData(convert.SliceAnyToInterface(a.NsgIds), types.String),
					"privateEndpointIp":           llx.StringDataPtr(a.PrivateEndpointIp),
					"privateEndpointLabel":        llx.StringDataPtr(a.PrivateEndpointLabel),
					"connectionUrls":              llx.DictData(connectionUrls),
					"publicConnectionUrls":        llx.DictData(publicConnectionUrls),
					"state":                       llx.StringData(string(a.LifecycleState)),
					"created":                     llx.TimeDataPtr(created),
					"freeformTags":                llx.MapData(freeformTags, types.String),
					"definedTags":                 llx.MapData(definedTags, types.Any),
					"systemTags":                  llx.MapData(definedTagsToAny(a.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				mqlAdb := mqlInstance.(*mqlOciDatabaseAutonomousDatabase)
				mqlAdb.cacheKmsKeyID = stringValue(a.KmsKeyId)
				mqlAdb.cacheVaultID = stringValue(a.VaultId)
				mqlAdb.cacheSubnetID = stringValue(a.SubnetId)
				mqlAdb.cacheSourceID = stringValue(a.SourceId)
				res = append(res, mqlAdb)
			}

			return res, nil
		})
}

type mqlOciDatabaseAutonomousDatabaseInternal struct {
	cacheKmsKeyID string
	cacheVaultID  string
	cacheSubnetID string
	cacheSourceID string
}

func (o *mqlOciDatabaseAutonomousDatabase) id() (string, error) {
	return "oci.database.autonomousDatabase/" + o.Id.Data, nil
}

func (o *mqlOciDatabaseAutonomousDatabase) kmsKey() (*mqlOciKmsKey, error) {
	if o.cacheKmsKeyID == "" || !isOcid(o.cacheKmsKeyID) {
		o.KmsKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.kms.key", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheKmsKeyID),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciKmsKey), nil
}

func (o *mqlOciDatabaseAutonomousDatabase) kmsVault() (*mqlOciKmsVault, error) {
	if o.cacheVaultID == "" || !isOcid(o.cacheVaultID) {
		o.KmsVault.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.kms.vault", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheVaultID),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciKmsVault), nil
}

func (o *mqlOciDatabaseAutonomousDatabase) subnet() (*mqlOciNetworkSubnet, error) {
	if o.cacheSubnetID == "" || !isOcid(o.cacheSubnetID) {
		o.Subnet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.subnet", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheSubnetID),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkSubnet), nil
}

// Database backups (VM/BM)

func (o *mqlOciDatabase) backups() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	return ociCollect(o.MqlRuntime, ociScopeTenancyRoot,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			log.Debug().Msgf("calling oci database backups with region %s", region)

			svc, err := conn.DatabaseClient(region)
			if err != nil {
				return nil, err
			}

			items, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]database.BackupSummary, *string, error) {
				response, err := svc.ListBackups(ctx, database.ListBackupsRequest{
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
				mqlBackup.cacheKmsKeyID = stringValue(b.KmsKeyId)
				mqlBackup.cacheVaultID = stringValue(b.VaultId)
				res = append(res, mqlBackup)
			}

			return res, nil
		})
}

type mqlOciDatabaseBackupInternal struct {
	cacheKmsKeyID string
	cacheVaultID  string
}

func (o *mqlOciDatabaseBackup) id() (string, error) {
	return "oci.database.backup/" + o.Id.Data, nil
}

func (o *mqlOciDatabaseBackup) kmsKey() (*mqlOciKmsKey, error) {
	if o.cacheKmsKeyID == "" || !isOcid(o.cacheKmsKeyID) {
		o.KmsKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.kms.key", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheKmsKeyID),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciKmsKey), nil
}

func (o *mqlOciDatabaseBackup) kmsVault() (*mqlOciKmsVault, error) {
	if o.cacheVaultID == "" || !isOcid(o.cacheVaultID) {
		o.KmsVault.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.kms.vault", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheVaultID),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciKmsVault), nil
}

// Autonomous Database Backups

func (o *mqlOciDatabase) autonomousDatabaseBackups() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	return ociCollect(o.MqlRuntime, ociScopeTenancyRoot,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			log.Debug().Msgf("calling oci autonomous database backups with region %s", region)

			svc, err := conn.DatabaseClient(region)
			if err != nil {
				return nil, err
			}

			items, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]database.AutonomousDatabaseBackupSummary, *string, error) {
				response, err := svc.ListAutonomousDatabaseBackups(ctx, database.ListAutonomousDatabaseBackupsRequest{
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
				mqlBackup.cacheAutonomousDatabaseID = stringValue(b.AutonomousDatabaseId)
				mqlBackup.cacheKmsKeyID = stringValue(b.KmsKeyId)
				mqlBackup.cacheVaultID = stringValue(b.VaultId)
				res = append(res, mqlBackup)
			}

			return res, nil
		})
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
		if b.cacheAutonomousDatabaseID == dbID {
			res = append(res, b)
		}
	}
	return res, nil
}

type mqlOciDatabaseAutonomousDatabaseBackupInternal struct {
	cacheAutonomousDatabaseID string
	cacheKmsKeyID             string
	cacheVaultID              string
}

func (o *mqlOciDatabaseAutonomousDatabaseBackup) id() (string, error) {
	return "oci.database.autonomousDatabaseBackup/" + o.Id.Data, nil
}

func (o *mqlOciDatabaseAutonomousDatabaseBackup) autonomousDatabase() (*mqlOciDatabaseAutonomousDatabase, error) {
	if o.cacheAutonomousDatabaseID == "" || !isOcid(o.cacheAutonomousDatabaseID) {
		o.AutonomousDatabase.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.database.autonomousDatabase", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheAutonomousDatabaseID),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciDatabaseAutonomousDatabase), nil
}

func (o *mqlOciDatabaseAutonomousDatabaseBackup) kmsKey() (*mqlOciKmsKey, error) {
	if o.cacheKmsKeyID == "" || !isOcid(o.cacheKmsKeyID) {
		o.KmsKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.kms.key", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheKmsKeyID),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciKmsKey), nil
}

func (o *mqlOciDatabaseAutonomousDatabaseBackup) kmsVault() (*mqlOciKmsVault, error) {
	if o.cacheVaultID == "" || !isOcid(o.cacheVaultID) {
		o.KmsVault.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.kms.vault", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheVaultID),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciKmsVault), nil
}
