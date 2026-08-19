// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"net/url"

	srcmv3 "github.com/confluentinc/ccloud-sdk-go-v2/srcm/v3"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/confluent/connection"
	"go.mondoo.com/mql/v13/types"
)

// mqlConfluentSchemaRegistryClusterInternal caches the environment reference,
// which the payload carries as an identifier.
type mqlConfluentSchemaRegistryClusterInternal struct {
	cachedEnvironmentID string
}

type schemaRegistryRecord struct {
	ID       string                      `json:"id"`
	Metadata objectMeta                  `json:"metadata"`
	Spec     *srcmv3.SrcmV3ClusterSpec   `json:"spec"`
	Status   *srcmv3.SrcmV3ClusterStatus `json:"status"`
}

func (r *mqlConfluent) schemaRegistryClusters() ([]any, error) {
	conn, err := confluentConn(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	envIDs, err := r.environmentIDs()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, envID := range envIDs {
		query := url.Values{}
		query.Set("environment", envID)

		records, err := connection.GetPaged[schemaRegistryRecord](context.Background(), conn,
			conn.CloudTarget(), "/srcm/v3/clusters", query)
		if err != nil {
			// An environment that has never provisioned Stream Governance has
			// no registry, and the endpoint answers that the object is absent.
			// Only that answer is turned into "no registry here": a permission
			// failure says the caller may not look, which is not the same
			// thing and stays an error.
			if connection.IsNotFound(err) {
				continue
			}
			return nil, err
		}

		for i := range records {
			record := records[i]
			spec := record.Spec
			if spec == nil {
				spec = &srcmv3.SrcmV3ClusterSpec{}
			}

			regional := map[string]string{}
			if config := spec.PrivateNetworkingConfig; config != nil {
				regional = config.GetRegionalEndpoints()
			}

			phase := ""
			if record.Status != nil {
				phase = record.Status.Phase
			}

			mqlCluster, err := CreateResource(r.MqlRuntime, "confluent.schemaRegistryCluster", map[string]*llx.RawData{
				"__id":                     llx.StringData(record.ID),
				"id":                       llx.StringData(record.ID),
				"displayName":              llx.StringData(spec.GetDisplayName()),
				"resourceName":             llx.StringData(record.Metadata.ResourceName),
				"streamGovernancePackage":  llx.StringData(spec.GetPackage()),
				"cloud":                    llx.StringData(spec.GetCloud()),
				"region":                   llx.StringData(spec.GetRegion()),
				"httpEndpoint":             llx.StringData(spec.GetHttpEndpoint()),
				"catalogHttpEndpoint":      llx.StringData(spec.GetCatalogHttpEndpoint()),
				"privateHttpEndpoint":      llx.StringData(spec.GetPrivateHttpEndpoint()),
				"privateRegionalEndpoints": llx.MapData(strMapToAny(regional), types.String),
				"isPublic":                 llx.BoolData(spec.GetHttpEndpoint() != ""),
				"phase":                    llx.StringData(phase),
				"createdAt":                llx.TimeDataPtr(record.Metadata.CreatedAt.Time()),
				"updatedAt":                llx.TimeDataPtr(record.Metadata.UpdatedAt.Time()),
			})
			if err != nil {
				return nil, err
			}

			cluster := mqlCluster.(*mqlConfluentSchemaRegistryCluster)
			cluster.cachedEnvironmentID = refID(spec.Environment)
			if cluster.cachedEnvironmentID == "" {
				cluster.cachedEnvironmentID = envID
			}
			res = append(res, cluster)
		}
	}
	return res, nil
}

func (r *mqlConfluentSchemaRegistryCluster) environment() (*mqlConfluentEnvironment, error) {
	env, err := environmentByID(r.MqlRuntime, r.cachedEnvironmentID)
	if err != nil {
		return nil, err
	}
	if env == nil {
		r.Environment.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return env, nil
}

// schemaRegistryClusterByID resolves a Schema Registry cluster from the root
// resource's cached list.
func schemaRegistryClusterByID(runtime *plugin.Runtime, id string) (*mqlConfluentSchemaRegistryCluster, error) {
	if id == "" {
		return nil, nil
	}
	root, err := getConfluent(runtime)
	if err != nil {
		return nil, err
	}
	clusters := root.GetSchemaRegistryClusters()
	if clusters.Error != nil {
		return nil, clusters.Error
	}
	for _, raw := range clusters.Data {
		cluster, ok := raw.(*mqlConfluentSchemaRegistryCluster)
		if !ok {
			continue
		}
		if cluster.GetId().Data == id {
			return cluster, nil
		}
	}
	return nil, nil
}
