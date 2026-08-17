// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/goldengate"
	"github.com/oracle/oci-go-sdk/v65/mysql"
	"github.com/oracle/oci-go-sdk/v65/nosql"
	"github.com/oracle/oci-go-sdk/v65/opensearch"
	"github.com/oracle/oci-go-sdk/v65/psql"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

// ociRegionsFor resolves the regions a fan-out should cover: the tenancy's
// subscribed regions, narrowed by the region filters.
//
// Every fan-out draws its regions from here so the filter is applied once. The
// `oci.regions` resource itself is deliberately left unfiltered - it answers
// "which regions is this tenancy subscribed to", which is a fact about the
// tenancy rather than a description of the current scan, and init_refs resolves
// typed region references through it. A region reference must not stop
// resolving because a scan was scoped elsewhere.
func ociRegionsFor(runtime *plugin.Runtime) ([]any, error) {
	conn := runtime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(runtime, "oci", nil)
	if err != nil {
		return nil, err
	}
	regions := ociResource.(*mqlOci).GetRegions()
	if regions.Error != nil {
		return nil, regions.Error
	}
	if !conn.Filters.HasRegions() {
		return regions.Data, nil
	}

	res := make([]any, 0, len(regions.Data))
	for _, raw := range regions.Data {
		region, ok := raw.(*mqlOciRegion)
		if !ok {
			return nil, errors.New("invalid region type")
		}
		if !conn.Filters.AdmitsRegion(region.Id.Data, region.Name.Data) {
			continue
		}
		res = append(res, raw)
	}
	return res, nil
}

// ----- MySQL HeatWave -----

func (o *mqlOciMysql) id() (string, error) {
	return "oci.mysql", nil
}

func (o *mqlOciMysql) dbSystems() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	return ociCollect(o.MqlRuntime, ociScopeAllCompartments,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			client, err := conn.MysqlDbSystemClient(region)
			if err != nil {
				return nil, err
			}

			systems, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]mysql.DbSystemSummary, *string, error) {
				response, err := client.ListDbSystems(ctx, mysql.ListDbSystemsRequest{
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
				mqlSystemTyped.cacheCompartmentID = stringValue(s.CompartmentId)
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
	cacheCompartmentID string
	cacheRegion        string

	// ociLazy, not ociRetryLazy: a DB system we are not allowed to read is
	// asked for once instead of once per field sharing the fetch.
	detail ociLazy[*mysql.DbSystem]
}

func (o *mqlOciMysqlDbSystem) id() (string, error) {
	return "oci.mysql.dbSystem/" + o.Id.Data, nil
}

func (o *mqlOciMysqlDbSystem) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.cacheCompartmentID, &o.Compartment)
}

func (o *mqlOciMysqlDbSystem) getDetail() (*mysql.DbSystem, error) {
	return o.detail.get(func() (*mysql.DbSystem, error) {
		conn := o.MqlRuntime.Connection.(*connection.OciConnection)
		client, err := conn.MysqlDbSystemClient(o.cacheRegion)
		if err != nil {
			return nil, err
		}

		response, err := client.GetDbSystem(context.Background(), mysql.GetDbSystemRequest{
			DbSystemId: common.String(o.Id.Data),
		})
		if err != nil {
			return nil, err
		}
		return &response.DbSystem, nil
	})
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

	return ociCollect(o.MqlRuntime, ociScopeAllCompartments,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			client, err := conn.PostgresqlClient(region)
			if err != nil {
				return nil, err
			}

			systems, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]psql.DbSystemSummary, *string, error) {
				response, err := client.ListDbSystems(ctx, psql.ListDbSystemsRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
				})
				if err != nil {
					return nil, nil, err
				}
				return response.DbSystemCollection.Items, response.OpcNextPage, nil
			})
			if err != nil {
				return nil, err
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
				mqlSystemTyped.cacheCompartmentID = stringValue(s.CompartmentId)
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
	cacheCompartmentID string
	cacheRegion        string

	detail ociLazy[*psql.DbSystem]
}

func (o *mqlOciPostgresqlDbSystem) id() (string, error) {
	return "oci.postgresql.dbSystem/" + o.Id.Data, nil
}

func (o *mqlOciPostgresqlDbSystem) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.cacheCompartmentID, &o.Compartment)
}

func (o *mqlOciPostgresqlDbSystem) getDetail() (*psql.DbSystem, error) {
	return o.detail.get(func() (*psql.DbSystem, error) {
		conn := o.MqlRuntime.Connection.(*connection.OciConnection)
		client, err := conn.PostgresqlClient(o.cacheRegion)
		if err != nil {
			return nil, err
		}

		response, err := client.GetDbSystem(context.Background(), psql.GetDbSystemRequest{
			DbSystemId: common.String(o.Id.Data),
		})
		if err != nil {
			return nil, err
		}
		return &response.DbSystem, nil
	})
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

	return ociCollect(o.MqlRuntime, ociScopeAllCompartments,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			client, err := conn.NosqlClient(region)
			if err != nil {
				return nil, err
			}

			tables, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]nosql.TableSummary, *string, error) {
				response, err := client.ListTables(ctx, nosql.ListTablesRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
				})
				if err != nil {
					return nil, nil, err
				}
				return response.TableCollection.Items, response.OpcNextPage, nil
			})
			if err != nil {
				return nil, err
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
				mqlTableTyped.cacheCompartmentID = stringValue(t.CompartmentId)
				res = append(res, mqlTableTyped)
			}

			return res, nil
		})
}

type mqlOciNosqlTableInternal struct {
	cacheCompartmentID string
}

func (o *mqlOciNosqlTable) id() (string, error) {
	return "oci.nosql.table/" + o.Id.Data, nil
}

func (o *mqlOciNosqlTable) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.cacheCompartmentID, &o.Compartment)
}

// ----- Search with OpenSearch -----

func (o *mqlOciOpensearch) id() (string, error) {
	return "oci.opensearch", nil
}

func (o *mqlOciOpensearch) clusters() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	return ociCollect(o.MqlRuntime, ociScopeAllCompartments,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			client, err := conn.OpensearchClusterClient(region)
			if err != nil {
				return nil, err
			}

			clusters, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]opensearch.OpensearchClusterSummary, *string, error) {
				response, err := client.ListOpensearchClusters(ctx, opensearch.ListOpensearchClustersRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
				})
				if err != nil {
					return nil, nil, err
				}
				return response.OpensearchClusterCollection.Items, response.OpcNextPage, nil
			})
			if err != nil {
				return nil, err
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
				mqlClusterTyped.cacheCompartmentID = stringValue(c.CompartmentId)
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
	cacheCompartmentID string
	cacheRegion        string

	detail ociLazy[*opensearch.OpensearchCluster]
}

func (o *mqlOciOpensearchCluster) id() (string, error) {
	return "oci.opensearch.cluster/" + o.Id.Data, nil
}

func (o *mqlOciOpensearchCluster) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.cacheCompartmentID, &o.Compartment)
}

func (o *mqlOciOpensearchCluster) getDetail() (*opensearch.OpensearchCluster, error) {
	return o.detail.get(func() (*opensearch.OpensearchCluster, error) {
		conn := o.MqlRuntime.Connection.(*connection.OciConnection)
		client, err := conn.OpensearchClusterClient(o.cacheRegion)
		if err != nil {
			return nil, err
		}

		response, err := client.GetOpensearchCluster(context.Background(), opensearch.GetOpensearchClusterRequest{
			OpensearchClusterId: common.String(o.Id.Data),
		})
		if err != nil {
			return nil, err
		}
		return &response.OpensearchCluster, nil
	})
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

// A cluster with no SAML configuration carries no SecuritySamlConfig at all.
// Reporting false there is correct: no federated sign-on is configured.
func (o *mqlOciOpensearchCluster) isSamlEnabled() (bool, error) {
	detail, err := o.getDetail()
	if err != nil {
		return false, err
	}
	if detail.SecuritySamlConfig == nil {
		return false, nil
	}
	return boolValue(detail.SecuritySamlConfig.IsEnabled), nil
}

func (o *mqlOciOpensearchCluster) samlIdpEntityId() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	if detail.SecuritySamlConfig == nil {
		return "", nil
	}
	return stringValue(detail.SecuritySamlConfig.IdpEntityId), nil
}

func (o *mqlOciOpensearchCluster) samlAdminBackendRole() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	if detail.SecuritySamlConfig == nil {
		return "", nil
	}
	return stringValue(detail.SecuritySamlConfig.AdminBackendRole), nil
}

func (o *mqlOciOpensearchCluster) samlSubjectKey() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	if detail.SecuritySamlConfig == nil {
		return "", nil
	}
	return stringValue(detail.SecuritySamlConfig.SubjectKey), nil
}

func (o *mqlOciOpensearchCluster) samlRolesKey() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	if detail.SecuritySamlConfig == nil {
		return "", nil
	}
	return stringValue(detail.SecuritySamlConfig.RolesKey), nil
}

func (o *mqlOciOpensearchCluster) samlOpendashboardUrl() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	if detail.SecuritySamlConfig == nil {
		return "", nil
	}
	return stringValue(detail.SecuritySamlConfig.OpendashboardUrl), nil
}

func (o *mqlOciOpensearchCluster) clusterCertificateMode() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	if detail.CertificateConfig == nil {
		return "", nil
	}
	return string(detail.CertificateConfig.ClusterCertificateMode), nil
}

func (o *mqlOciOpensearchCluster) dashboardCertificateMode() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	if detail.CertificateConfig == nil {
		return "", nil
	}
	return string(detail.CertificateConfig.DashboardCertificateMode), nil
}

func (o *mqlOciOpensearchCluster) apiCertificate() (*mqlOciCertificatesCertificate, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	var id string
	if detail.CertificateConfig != nil {
		id = stringValue(detail.CertificateConfig.OpenSearchApiCertificateId)
	}
	return resolveOciOpensearchCertificate(o.MqlRuntime, id, &o.ApiCertificate)
}

func (o *mqlOciOpensearchCluster) dashboardCertificate() (*mqlOciCertificatesCertificate, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	var id string
	if detail.CertificateConfig != nil {
		id = stringValue(detail.CertificateConfig.OpenSearchDashboardCertificateId)
	}
	return resolveOciOpensearchCertificate(o.MqlRuntime, id, &o.DashboardCertificate)
}

// resolveOciOpensearchCertificate resolves a certificate reference, reporting
// null for the Oracle-managed case where the cluster names no certificate of
// its own.
func resolveOciOpensearchCertificate(runtime *plugin.Runtime, id string, field *plugin.TValue[*mqlOciCertificatesCertificate]) (*mqlOciCertificatesCertificate, error) {
	if id == "" {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(runtime, "oci.certificates.certificate", map[string]*llx.RawData{
		"id": llx.StringData(id),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOciCertificatesCertificate), nil
}

func (o *mqlOciOpensearchCluster) securityGroups() ([]any, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	// The SDK models a single group rather than a list, but every sibling
	// resource exposes securityGroups as a list, so keep the shape uniform.
	nsgID := stringValue(detail.NsgId)
	if nsgID == "" {
		return []any{}, nil
	}
	return resolveOciSecurityGroups(o.MqlRuntime, []any{nsgID})
}

func (o *mqlOciOpensearchCluster) fqdn() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	return stringValue(detail.Fqdn), nil
}

func (o *mqlOciOpensearchCluster) loadBalancerServiceType() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	if detail.LoadBalancerConfig == nil {
		return "", nil
	}
	return string(detail.LoadBalancerConfig.LoadBalancerServiceType), nil
}

func (o *mqlOciOpensearchCluster) reverseConnectionEndpoints() ([]any, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(detail.ReverseConnectionEndpoints)
}

func (o *mqlOciOpensearchCluster) inboundClusters() ([]any, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	res := make([]any, 0, len(detail.InboundClusterIds))
	for _, id := range detail.InboundClusterIds {
		if id == "" {
			continue
		}
		mqlCluster, err := NewResource(o.MqlRuntime, "oci.opensearch.cluster", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			// A cluster in a compartment we cannot read must not take the rest
			// of the list with it.
			log.Debug().Err(err).Str("cluster", id).Msg("skipping unresolvable oci opensearch cluster")
			continue
		}
		res = append(res, mqlCluster)
	}
	return res, nil
}

// ----- GoldenGate -----

func (o *mqlOciGoldenGate) id() (string, error) {
	return "oci.goldenGate", nil
}

func (o *mqlOciGoldenGate) deployments() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	return ociCollect(o.MqlRuntime, ociScopeAllCompartments,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			client, err := conn.GoldenGateClient(region)
			if err != nil {
				return nil, err
			}

			deployments, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]goldengate.DeploymentSummary, *string, error) {
				response, err := client.ListDeployments(ctx, goldengate.ListDeploymentsRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
				})
				if err != nil {
					return nil, nil, err
				}
				return response.DeploymentCollection.Items, response.OpcNextPage, nil
			})
			if err != nil {
				return nil, err
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
				mqlDeploymentTyped.cacheCompartmentID = stringValue(d.CompartmentId)
				mqlDeploymentTyped.cacheSubnetID = stringValue(d.SubnetId)
				mqlDeploymentTyped.cacheLoadBalancerID = stringValue(d.LoadBalancerId)
				mqlDeploymentTyped.cacheRegion = region
				res = append(res, mqlDeploymentTyped)
			}

			return res, nil
		})
}

// The deployment listing carries the subnet but not the network security
// groups, so only securityGroups needs the detail call.
type mqlOciGoldenGateDeploymentInternal struct {
	cacheCompartmentID  string
	cacheSubnetID       string
	cacheLoadBalancerID string
	cacheRegion         string

	detail ociLazy[*goldengate.Deployment]
}

func (o *mqlOciGoldenGateDeployment) id() (string, error) {
	return "oci.goldenGate.deployment/" + o.Id.Data, nil
}

func (o *mqlOciGoldenGateDeployment) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.cacheCompartmentID, &o.Compartment)
}

func (o *mqlOciGoldenGateDeployment) subnet() (*mqlOciNetworkSubnet, error) {
	return resolveOciSubnet(o.MqlRuntime, o.cacheSubnetID, &o.Subnet)
}

func (o *mqlOciGoldenGateDeployment) loadBalancer() (*mqlOciLoadBalancerLoadBalancer, error) {
	if o.cacheLoadBalancerID == "" {
		o.LoadBalancer.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(o.MqlRuntime, "oci.loadBalancer.loadBalancer", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheLoadBalancerID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOciLoadBalancerLoadBalancer), nil
}

func (o *mqlOciGoldenGateDeployment) getDetail() (*goldengate.Deployment, error) {
	return o.detail.get(func() (*goldengate.Deployment, error) {
		conn := o.MqlRuntime.Connection.(*connection.OciConnection)
		client, err := conn.GoldenGateClient(o.cacheRegion)
		if err != nil {
			return nil, err
		}

		response, err := client.GetDeployment(context.Background(), goldengate.GetDeploymentRequest{
			DeploymentId: common.String(o.Id.Data),
		})
		if err != nil {
			return nil, err
		}
		return &response.Deployment, nil
	})
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
