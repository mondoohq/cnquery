// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/goldengate"
	"github.com/oracle/oci-go-sdk/v65/mysql"
	"github.com/oracle/oci-go-sdk/v65/nosql"
	"github.com/oracle/oci-go-sdk/v65/opensearch"
	"github.com/oracle/oci-go-sdk/v65/psql"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

// ociRegionsFor resolves the tenancy's subscribed regions, which every
// compartment fan-out needs before it can build its jobs.
func ociRegionsFor(runtime *plugin.Runtime) ([]any, error) {
	ociResource, err := CreateResource(runtime, "oci", nil)
	if err != nil {
		return nil, err
	}
	regions := ociResource.(*mqlOci).GetRegions()
	if regions.Error != nil {
		return nil, regions.Error
	}
	return regions.Data, nil
}

// ----- MySQL HeatWave -----

func (o *mqlOciMysql) id() (string, error) {
	return "oci.mysql", nil
}

func (o *mqlOciMysql) dbSystems() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	regions, err := ociRegionsFor(o.MqlRuntime)
	if err != nil {
		return nil, err
	}

	return ociRunCompartmentRegionPool(conn, regions,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			client, err := conn.MysqlDbSystemClient(region)
			if err != nil {
				return nil, err
			}

			systems := []mysql.DbSystemSummary{}
			var page *string
			for {
				response, err := client.ListDbSystems(ctx, mysql.ListDbSystemsRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				systems = append(systems, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			res := make([]any, 0, len(systems))
			for i := range systems {
				s := systems[i]

				endpoints, err := convert.JsonToDictSlice(s.Endpoints)
				if err != nil {
					return nil, err
				}

				var backupPolicy, deletionPolicy map[string]any
				if s.BackupPolicy != nil {
					backupPolicy, err = convert.JsonToDict(s.BackupPolicy)
					if err != nil {
						return nil, err
					}
				}
				if s.DeletionPolicy != nil {
					deletionPolicy, err = convert.JsonToDict(s.DeletionPolicy)
					if err != nil {
						return nil, err
					}
				}

				mqlSystem, err := CreateResource(o.MqlRuntime, "oci.mysql.dbSystem", map[string]*llx.RawData{
					"id":                        llx.StringDataPtr(s.Id),
					"name":                      llx.StringDataPtr(s.DisplayName),
					"description":               llx.StringDataPtr(s.Description),
					"version":                   llx.StringDataPtr(s.MysqlVersion),
					"shape":                     llx.StringDataPtr(s.ShapeName),
					"databaseMode":              llx.StringData(string(s.DatabaseMode)),
					"accessMode":                llx.StringData(string(s.AccessMode)),
					"isHighlyAvailable":         llx.BoolData(boolValue(s.IsHighlyAvailable)),
					"isHeatWaveClusterAttached": llx.BoolData(boolValue(s.IsHeatWaveClusterAttached)),
					"crashRecovery":             llx.StringData(string(s.CrashRecovery)),
					"databaseManagement":        llx.StringData(string(s.DatabaseManagement)),
					"endpoints":                 llx.ArrayData(endpoints, types.Dict),
					"backupPolicy":              llx.DictData(backupPolicy),
					"deletionPolicy":            llx.DictData(deletionPolicy),
					"availabilityDomain":        llx.StringDataPtr(s.AvailabilityDomain),
					"faultDomain":               llx.StringDataPtr(s.FaultDomain),
					"state":                     llx.StringData(string(s.LifecycleState)),
					"created":                   sdkTimeData(s.TimeCreated),
					"updated":                   sdkTimeData(s.TimeUpdated),
					"freeformTags":              llx.MapData(strMapToAny(s.FreeformTags), types.String),
					"definedTags":               llx.MapData(definedTagsToAny(s.DefinedTags), types.Any),
					"systemTags":                llx.MapData(definedTagsToAny(s.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				mqlSystemTyped := mqlSystem.(*mqlOciMysqlDbSystem)
				mqlSystemTyped.cacheCompartmentId = stringValue(s.CompartmentId)
				mqlSystemTyped.cacheRegion = region
				res = append(res, mqlSystemTyped)
			}

			return res, nil
		})
}

// The MySQL listing omits network placement and both encryption settings, so
// subnet, security groups, the data-at-rest key and the TLS certificate all
// come from one shared detail call rather than four.
type mqlOciMysqlDbSystemInternal struct {
	cacheCompartmentId string
	cacheRegion        string

	detailLock    sync.Mutex
	detailFetched atomic.Bool
	detail        *mysql.DbSystem
	detailErr     error
}

func (o *mqlOciMysqlDbSystem) id() (string, error) {
	return "oci.mysql.dbSystem/" + o.Id.Data, nil
}

func (o *mqlOciMysqlDbSystem) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.cacheCompartmentId, &o.Compartment)
}

func (o *mqlOciMysqlDbSystem) getDetail() (*mysql.DbSystem, error) {
	if o.detailFetched.Load() {
		return o.detail, o.detailErr
	}

	o.detailLock.Lock()
	defer o.detailLock.Unlock()
	if o.detailFetched.Load() {
		return o.detail, o.detailErr
	}

	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	client, err := conn.MysqlDbSystemClient(o.cacheRegion)
	if err != nil {
		// A failed detail call is remembered as well, so a DB system we are
		// not allowed to read is asked for once instead of once per field.
		o.detailErr = err
		o.detailFetched.Store(true)
		return nil, err
	}

	response, err := client.GetDbSystem(context.Background(), mysql.GetDbSystemRequest{
		DbSystemId: common.String(o.Id.Data),
	})
	if err != nil {
		o.detailErr = err
		o.detailFetched.Store(true)
		return nil, err
	}

	o.detail = &response.DbSystem
	o.detailFetched.Store(true)
	return o.detail, nil
}

func (o *mqlOciMysqlDbSystem) subnet() (*mqlOciNetworkSubnet, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	return resolveOciSubnet(o.MqlRuntime, stringValue(detail.SubnetId), &o.Subnet)
}

func (o *mqlOciMysqlDbSystem) securityGroups() ([]any, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	return resolveOciSecurityGroups(o.MqlRuntime, stringsToAny(detail.NsgIds))
}

func (o *mqlOciMysqlDbSystem) encryptionKeyGenerationType() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	if detail.EncryptData == nil {
		return "", nil
	}
	return string(detail.EncryptData.KeyGenerationType), nil
}

func (o *mqlOciMysqlDbSystem) encryptionKey() (*mqlOciKmsKey, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	// A SYSTEM-generated key is Oracle-managed and has no KMS key to resolve;
	// only BYOK carries a customer key OCID.
	if detail.EncryptData == nil || stringValue(detail.EncryptData.KeyId) == "" {
		o.EncryptionKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(o.MqlRuntime, "oci.kms.key", map[string]*llx.RawData{
		"id": llx.StringData(stringValue(detail.EncryptData.KeyId)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOciKmsKey), nil
}

func (o *mqlOciMysqlDbSystem) tlsCertificateType() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	if detail.SecureConnections == nil {
		return "", nil
	}
	return string(detail.SecureConnections.CertificateGenerationType), nil
}

func (o *mqlOciMysqlDbSystem) tlsCertificate() (*mqlOciCertificatesCertificate, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	if detail.SecureConnections == nil || stringValue(detail.SecureConnections.CertificateId) == "" {
		o.TlsCertificate.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(o.MqlRuntime, "oci.certificates.certificate", map[string]*llx.RawData{
		"id": llx.StringData(stringValue(detail.SecureConnections.CertificateId)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOciCertificatesCertificate), nil
}

// ----- Database with PostgreSQL -----

func (o *mqlOciPostgresql) id() (string, error) {
	return "oci.postgresql", nil
}

func (o *mqlOciPostgresql) dbSystems() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	regions, err := ociRegionsFor(o.MqlRuntime)
	if err != nil {
		return nil, err
	}

	return ociRunCompartmentRegionPool(conn, regions,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			client, err := conn.PostgresqlClient(region)
			if err != nil {
				return nil, err
			}

			systems := []psql.DbSystemSummary{}
			var page *string
			for {
				response, err := client.ListDbSystems(ctx, psql.ListDbSystemsRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				systems = append(systems, response.DbSystemCollection.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			res := make([]any, 0, len(systems))
			for i := range systems {
				s := systems[i]

				mqlSystem, err := CreateResource(o.MqlRuntime, "oci.postgresql.dbSystem", map[string]*llx.RawData{
					"id":                      llx.StringDataPtr(s.Id),
					"name":                    llx.StringDataPtr(s.DisplayName),
					"dbVersion":               llx.StringDataPtr(s.DbVersion),
					"shape":                   llx.StringDataPtr(s.Shape),
					"systemType":              llx.StringData(string(s.SystemType)),
					"instanceCount":           llx.IntDataDefault(s.InstanceCount, 0),
					"instanceOcpuCount":       llx.IntDataDefault(s.InstanceOcpuCount, 0),
					"instanceMemorySizeInGBs": llx.IntDataDefault(s.InstanceMemorySizeInGBs, 0),
					"configId":                llx.StringDataPtr(s.ConfigId),
					"state":                   llx.StringData(string(s.LifecycleState)),
					"stateDetails":            llx.StringDataPtr(s.LifecycleDetails),
					"created":                 sdkTimeData(s.TimeCreated),
					"updated":                 sdkTimeData(s.TimeUpdated),
					"freeformTags":            llx.MapData(strMapToAny(s.FreeformTags), types.String),
					"definedTags":             llx.MapData(definedTagsToAny(s.DefinedTags), types.Any),
					"systemTags":              llx.MapData(definedTagsToAny(s.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				mqlSystemTyped := mqlSystem.(*mqlOciPostgresqlDbSystem)
				mqlSystemTyped.cacheCompartmentId = stringValue(s.CompartmentId)
				mqlSystemTyped.cacheRegion = region
				res = append(res, mqlSystemTyped)
			}

			return res, nil
		})
}

// The PostgreSQL listing is identity and sizing only. Network placement, the
// admin user, storage and the backup policy - everything the security posture
// depends on - are detail-only, so they share one fetch.
type mqlOciPostgresqlDbSystemInternal struct {
	cacheCompartmentId string
	cacheRegion        string

	detailLock    sync.Mutex
	detailFetched atomic.Bool
	detail        *psql.DbSystem
	detailErr     error
}

func (o *mqlOciPostgresqlDbSystem) id() (string, error) {
	return "oci.postgresql.dbSystem/" + o.Id.Data, nil
}

func (o *mqlOciPostgresqlDbSystem) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.cacheCompartmentId, &o.Compartment)
}

func (o *mqlOciPostgresqlDbSystem) getDetail() (*psql.DbSystem, error) {
	if o.detailFetched.Load() {
		return o.detail, o.detailErr
	}

	o.detailLock.Lock()
	defer o.detailLock.Unlock()
	if o.detailFetched.Load() {
		return o.detail, o.detailErr
	}

	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	client, err := conn.PostgresqlClient(o.cacheRegion)
	if err != nil {
		o.detailErr = err
		o.detailFetched.Store(true)
		return nil, err
	}

	response, err := client.GetDbSystem(context.Background(), psql.GetDbSystemRequest{
		DbSystemId: common.String(o.Id.Data),
	})
	if err != nil {
		o.detailErr = err
		o.detailFetched.Store(true)
		return nil, err
	}

	o.detail = &response.DbSystem
	o.detailFetched.Store(true)
	return o.detail, nil
}

func (o *mqlOciPostgresqlDbSystem) description() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	return stringValue(detail.Description), nil
}

func (o *mqlOciPostgresqlDbSystem) systemRole() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	return string(detail.SystemRole), nil
}

func (o *mqlOciPostgresqlDbSystem) adminUsername() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	return stringValue(detail.AdminUsername), nil
}

func (o *mqlOciPostgresqlDbSystem) subnet() (*mqlOciNetworkSubnet, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	if detail.NetworkDetails == nil {
		o.Subnet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return resolveOciSubnet(o.MqlRuntime, stringValue(detail.NetworkDetails.SubnetId), &o.Subnet)
}

func (o *mqlOciPostgresqlDbSystem) securityGroups() ([]any, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	if detail.NetworkDetails == nil {
		return []any{}, nil
	}
	return resolveOciSecurityGroups(o.MqlRuntime, stringsToAny(detail.NetworkDetails.NsgIds))
}

func (o *mqlOciPostgresqlDbSystem) primaryEndpointPrivateIp() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	if detail.NetworkDetails == nil {
		return "", nil
	}
	return stringValue(detail.NetworkDetails.PrimaryDbEndpointPrivateIp), nil
}

func (o *mqlOciPostgresqlDbSystem) isReaderEndpointEnabled() (bool, error) {
	detail, err := o.getDetail()
	if err != nil {
		return false, err
	}
	if detail.NetworkDetails == nil {
		return false, nil
	}
	return boolValue(detail.NetworkDetails.IsReaderEndpointEnabled), nil
}

func (o *mqlOciPostgresqlDbSystem) storageDetails() (any, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	if detail.StorageDetails == nil {
		return nil, nil
	}
	return convert.JsonToDict(detail.StorageDetails)
}

func (o *mqlOciPostgresqlDbSystem) managementPolicy() (any, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	if detail.ManagementPolicy == nil {
		return nil, nil
	}
	return convert.JsonToDict(detail.ManagementPolicy)
}

// ----- NoSQL Database -----

func (o *mqlOciNosql) id() (string, error) {
	return "oci.nosql", nil
}

func (o *mqlOciNosql) tables() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	regions, err := ociRegionsFor(o.MqlRuntime)
	if err != nil {
		return nil, err
	}

	return ociRunCompartmentRegionPool(conn, regions,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			client, err := conn.NosqlClient(region)
			if err != nil {
				return nil, err
			}

			tables := []nosql.TableSummary{}
			var page *string
			for {
				response, err := client.ListTables(ctx, nosql.ListTablesRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				tables = append(tables, response.TableCollection.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			res := make([]any, 0, len(tables))
			for i := range tables {
				t := tables[i]

				var tableLimits map[string]any
				if t.TableLimits != nil {
					tableLimits, err = convert.JsonToDict(t.TableLimits)
					if err != nil {
						return nil, err
					}
				}

				mqlTable, err := CreateResource(o.MqlRuntime, "oci.nosql.table", map[string]*llx.RawData{
					"id":                llx.StringDataPtr(t.Id),
					"name":              llx.StringDataPtr(t.Name),
					"tableLimits":       llx.DictData(tableLimits),
					"isMultiRegion":     llx.BoolData(boolValue(t.IsMultiRegion)),
					"isAutoReclaimable": llx.BoolData(boolValue(t.IsAutoReclaimable)),
					"timeOfExpiration":  sdkTimeData(t.TimeOfExpiration),
					"schemaState":       llx.StringData(string(t.SchemaState)),
					"state":             llx.StringData(string(t.LifecycleState)),
					"created":           sdkTimeData(t.TimeCreated),
					"updated":           sdkTimeData(t.TimeUpdated),
					"freeformTags":      llx.MapData(strMapToAny(t.FreeformTags), types.String),
					"definedTags":       llx.MapData(definedTagsToAny(t.DefinedTags), types.Any),
					"systemTags":        llx.MapData(definedTagsToAny(t.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				mqlTableTyped := mqlTable.(*mqlOciNosqlTable)
				mqlTableTyped.cacheCompartmentId = stringValue(t.CompartmentId)
				res = append(res, mqlTableTyped)
			}

			return res, nil
		})
}

type mqlOciNosqlTableInternal struct {
	cacheCompartmentId string
}

func (o *mqlOciNosqlTable) id() (string, error) {
	return "oci.nosql.table/" + o.Id.Data, nil
}

func (o *mqlOciNosqlTable) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.cacheCompartmentId, &o.Compartment)
}

// ----- Search with OpenSearch -----

func (o *mqlOciOpensearch) id() (string, error) {
	return "oci.opensearch", nil
}

func (o *mqlOciOpensearch) clusters() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	regions, err := ociRegionsFor(o.MqlRuntime)
	if err != nil {
		return nil, err
	}

	return ociRunCompartmentRegionPool(conn, regions,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			client, err := conn.OpensearchClusterClient(region)
			if err != nil {
				return nil, err
			}

			clusters := []opensearch.OpensearchClusterSummary{}
			var page *string
			for {
				response, err := client.ListOpensearchClusters(ctx, opensearch.ListOpensearchClustersRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				clusters = append(clusters, response.OpensearchClusterCollection.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			res := make([]any, 0, len(clusters))
			for i := range clusters {
				c := clusters[i]

				var backupPolicy, outboundClusterConfig map[string]any
				if c.BackupPolicy != nil {
					backupPolicy, err = convert.JsonToDict(c.BackupPolicy)
					if err != nil {
						return nil, err
					}
				}
				if c.OutboundClusterConfig != nil {
					outboundClusterConfig, err = convert.JsonToDict(c.OutboundClusterConfig)
					if err != nil {
						return nil, err
					}
				}

				mqlCluster, err := CreateResource(o.MqlRuntime, "oci.opensearch.cluster", map[string]*llx.RawData{
					"id":                    llx.StringDataPtr(c.Id),
					"name":                  llx.StringDataPtr(c.DisplayName),
					"softwareVersion":       llx.StringDataPtr(c.SoftwareVersion),
					"securityMode":          llx.StringData(string(c.SecurityMode)),
					"outboundClusterConfig": llx.DictData(outboundClusterConfig),
					"backupPolicy":          llx.DictData(backupPolicy),
					"totalStorageGB":        llx.IntDataDefault(c.TotalStorageGB, 0),
					"availabilityDomains":   llx.ArrayData(stringsToAny(c.AvailabilityDomains), types.String),
					"securityAttributes":    llx.MapData(definedTagsToAny(c.SecurityAttributes), types.Dict),
					"state":                 llx.StringData(string(c.LifecycleState)),
					"stateDetails":          llx.StringDataPtr(c.LifecycleDetails),
					"created":               sdkTimeData(c.TimeCreated),
					"updated":               sdkTimeData(c.TimeUpdated),
					"freeformTags":          llx.MapData(strMapToAny(c.FreeformTags), types.String),
					"definedTags":           llx.MapData(definedTagsToAny(c.DefinedTags), types.Any),
					"systemTags":            llx.MapData(definedTagsToAny(c.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				mqlClusterTyped := mqlCluster.(*mqlOciOpensearchCluster)
				mqlClusterTyped.cacheCompartmentId = stringValue(c.CompartmentId)
				mqlClusterTyped.cacheRegion = region
				res = append(res, mqlClusterTyped)
			}

			return res, nil
		})
}

// The OpenSearch listing carries securityMode but not where the cluster sits
// or what it answers on, so the network placement and endpoints share a detail
// call. securityMasterUserPasswordHash is deliberately not exposed.
type mqlOciOpensearchClusterInternal struct {
	cacheCompartmentId string
	cacheRegion        string

	detailLock    sync.Mutex
	detailFetched atomic.Bool
	detail        *opensearch.OpensearchCluster
	detailErr     error
}

func (o *mqlOciOpensearchCluster) id() (string, error) {
	return "oci.opensearch.cluster/" + o.Id.Data, nil
}

func (o *mqlOciOpensearchCluster) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.cacheCompartmentId, &o.Compartment)
}

func (o *mqlOciOpensearchCluster) getDetail() (*opensearch.OpensearchCluster, error) {
	if o.detailFetched.Load() {
		return o.detail, o.detailErr
	}

	o.detailLock.Lock()
	defer o.detailLock.Unlock()
	if o.detailFetched.Load() {
		return o.detail, o.detailErr
	}

	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	client, err := conn.OpensearchClusterClient(o.cacheRegion)
	if err != nil {
		o.detailErr = err
		o.detailFetched.Store(true)
		return nil, err
	}

	response, err := client.GetOpensearchCluster(context.Background(), opensearch.GetOpensearchClusterRequest{
		OpensearchClusterId: common.String(o.Id.Data),
	})
	if err != nil {
		o.detailErr = err
		o.detailFetched.Store(true)
		return nil, err
	}

	o.detail = &response.OpensearchCluster
	o.detailFetched.Store(true)
	return o.detail, nil
}

func (o *mqlOciOpensearchCluster) securityMasterUserName() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	return stringValue(detail.SecurityMasterUserName), nil
}

func (o *mqlOciOpensearchCluster) vcn() (*mqlOciNetworkVcn, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	if stringValue(detail.VcnId) == "" {
		o.Vcn.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(o.MqlRuntime, "oci.network.vcn", map[string]*llx.RawData{
		"id": llx.StringData(stringValue(detail.VcnId)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOciNetworkVcn), nil
}

func (o *mqlOciOpensearchCluster) subnet() (*mqlOciNetworkSubnet, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	return resolveOciSubnet(o.MqlRuntime, stringValue(detail.SubnetId), &o.Subnet)
}

func (o *mqlOciOpensearchCluster) opensearchFqdn() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	return stringValue(detail.OpensearchFqdn), nil
}

func (o *mqlOciOpensearchCluster) opensearchPrivateIp() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	return stringValue(detail.OpensearchPrivateIp), nil
}

func (o *mqlOciOpensearchCluster) opendashboardFqdn() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	return stringValue(detail.OpendashboardFqdn), nil
}

func (o *mqlOciOpensearchCluster) opendashboardPrivateIp() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	return stringValue(detail.OpendashboardPrivateIp), nil
}

// ----- GoldenGate -----

func (o *mqlOciGoldenGate) id() (string, error) {
	return "oci.goldenGate", nil
}

func (o *mqlOciGoldenGate) deployments() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	regions, err := ociRegionsFor(o.MqlRuntime)
	if err != nil {
		return nil, err
	}

	return ociRunCompartmentRegionPool(conn, regions,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			client, err := conn.GoldenGateClient(region)
			if err != nil {
				return nil, err
			}

			deployments := []goldengate.DeploymentSummary{}
			var page *string
			for {
				response, err := client.ListDeployments(ctx, goldengate.ListDeploymentsRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				deployments = append(deployments, response.DeploymentCollection.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			res := make([]any, 0, len(deployments))
			for i := range deployments {
				d := deployments[i]

				mqlDeployment, err := CreateResource(o.MqlRuntime, "oci.goldenGate.deployment", map[string]*llx.RawData{
					"id":                   llx.StringDataPtr(d.Id),
					"name":                 llx.StringDataPtr(d.DisplayName),
					"description":          llx.StringDataPtr(d.Description),
					"isPublic":             llx.BoolData(boolValue(d.IsPublic)),
					"publicIpAddress":      llx.StringDataPtr(d.PublicIpAddress),
					"privateIpAddress":     llx.StringDataPtr(d.PrivateIpAddress),
					"fqdn":                 llx.StringDataPtr(d.Fqdn),
					"deploymentUrl":        llx.StringDataPtr(d.DeploymentUrl),
					"deploymentType":       llx.StringData(string(d.DeploymentType)),
					"category":             llx.StringData(string(d.Category)),
					"environmentType":      llx.StringData(string(d.EnvironmentType)),
					"licenseModel":         llx.StringData(string(d.LicenseModel)),
					"isLatestVersion":      llx.BoolData(boolValue(d.IsLatestVersion)),
					"isAutoScalingEnabled": llx.BoolData(boolValue(d.IsAutoScalingEnabled)),
					"cpuCoreCount":         llx.IntDataDefault(d.CpuCoreCount, 0),
					"securityAttributes":   llx.MapData(definedTagsToAny(d.SecurityAttributes), types.Dict),
					"state":                llx.StringData(string(d.LifecycleState)),
					"lifecycleSubState":    llx.StringData(string(d.LifecycleSubState)),
					"stateDetails":         llx.StringDataPtr(d.LifecycleDetails),
					"created":              sdkTimeData(d.TimeCreated),
					"updated":              sdkTimeData(d.TimeUpdated),
					"freeformTags":         llx.MapData(strMapToAny(d.FreeformTags), types.String),
					"definedTags":          llx.MapData(definedTagsToAny(d.DefinedTags), types.Any),
					"systemTags":           llx.MapData(definedTagsToAny(d.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				mqlDeploymentTyped := mqlDeployment.(*mqlOciGoldenGateDeployment)
				mqlDeploymentTyped.cacheCompartmentId = stringValue(d.CompartmentId)
				mqlDeploymentTyped.cacheSubnetId = stringValue(d.SubnetId)
				mqlDeploymentTyped.cacheLoadBalancerId = stringValue(d.LoadBalancerId)
				mqlDeploymentTyped.cacheRegion = region
				res = append(res, mqlDeploymentTyped)
			}

			return res, nil
		})
}

// The deployment listing carries the subnet but not the network security
// groups, so only securityGroups needs the detail call.
type mqlOciGoldenGateDeploymentInternal struct {
	cacheCompartmentId  string
	cacheSubnetId       string
	cacheLoadBalancerId string
	cacheRegion         string

	detailLock    sync.Mutex
	detailFetched atomic.Bool
	detail        *goldengate.Deployment
	detailErr     error
}

func (o *mqlOciGoldenGateDeployment) id() (string, error) {
	return "oci.goldenGate.deployment/" + o.Id.Data, nil
}

func (o *mqlOciGoldenGateDeployment) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.cacheCompartmentId, &o.Compartment)
}

func (o *mqlOciGoldenGateDeployment) subnet() (*mqlOciNetworkSubnet, error) {
	return resolveOciSubnet(o.MqlRuntime, o.cacheSubnetId, &o.Subnet)
}

func (o *mqlOciGoldenGateDeployment) loadBalancer() (*mqlOciLoadBalancerLoadBalancer, error) {
	if o.cacheLoadBalancerId == "" {
		o.LoadBalancer.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(o.MqlRuntime, "oci.loadBalancer.loadBalancer", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheLoadBalancerId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOciLoadBalancerLoadBalancer), nil
}

func (o *mqlOciGoldenGateDeployment) getDetail() (*goldengate.Deployment, error) {
	if o.detailFetched.Load() {
		return o.detail, o.detailErr
	}

	o.detailLock.Lock()
	defer o.detailLock.Unlock()
	if o.detailFetched.Load() {
		return o.detail, o.detailErr
	}

	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	client, err := conn.GoldenGateClient(o.cacheRegion)
	if err != nil {
		o.detailErr = err
		o.detailFetched.Store(true)
		return nil, err
	}

	response, err := client.GetDeployment(context.Background(), goldengate.GetDeploymentRequest{
		DeploymentId: common.String(o.Id.Data),
	})
	if err != nil {
		o.detailErr = err
		o.detailFetched.Store(true)
		return nil, err
	}

	o.detail = &response.Deployment
	o.detailFetched.Store(true)
	return o.detail, nil
}

func (o *mqlOciGoldenGateDeployment) securityGroups() ([]any, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return []any{}, nil
	}
	return resolveOciSecurityGroups(o.MqlRuntime, stringsToAny(detail.NsgIds))
}
