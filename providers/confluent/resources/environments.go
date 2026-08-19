// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	orgv2 "github.com/confluentinc/ccloud-sdk-go-v2/org/v2"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/confluent/connection"
)

// environmentRecord is one entry of the environments listing.
type environmentRecord struct {
	ID                     string                             `json:"id"`
	DisplayName            string                             `json:"display_name"`
	Metadata               objectMeta                         `json:"metadata"`
	StreamGovernanceConfig *orgv2.OrgV2StreamGovernanceConfig `json:"stream_governance_config"`
}

func (r *mqlConfluent) environments() ([]any, error) {
	conn, err := confluentConn(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	records, err := connection.GetPaged[environmentRecord](context.Background(), conn,
		conn.CloudTarget(), "/org/v2/environments", nil)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		record := records[i]

		governance := llx.NilData
		if pkg := record.StreamGovernanceConfig.GetPackage(); pkg != "" {
			governance = llx.StringData(pkg)
		}

		mqlEnv, err := CreateResource(r.MqlRuntime, "confluent.environment", map[string]*llx.RawData{
			"__id":                    llx.StringData(record.ID),
			"id":                      llx.StringData(record.ID),
			"displayName":             llx.StringData(record.DisplayName),
			"resourceName":            llx.StringData(record.Metadata.ResourceName),
			"streamGovernancePackage": governance,
			"createdAt":               llx.TimeDataPtr(record.Metadata.CreatedAt.Time()),
			"updatedAt":               llx.TimeDataPtr(record.Metadata.UpdatedAt.Time()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlEnv)
	}
	return res, nil
}

// environmentIDs returns the identifiers of every environment the API key can
// reach. The per-environment listings all need one, and reading them from the
// cached environment list keeps that to a single call.
func (r *mqlConfluent) environmentIDs() ([]string, error) {
	envs := r.GetEnvironments()
	if envs.Error != nil {
		return nil, envs.Error
	}

	ids := make([]string, 0, len(envs.Data))
	for _, raw := range envs.Data {
		env, ok := raw.(*mqlConfluentEnvironment)
		if !ok {
			continue
		}
		id := env.GetId()
		if id.Error != nil {
			return nil, id.Error
		}
		if id.Data != "" {
			ids = append(ids, id.Data)
		}
	}
	return ids, nil
}

// environmentByID resolves an environment from the root resource's cached list.
// Going through that list keeps a query walking from many children back to
// their environment down to the one call the list already made.
func environmentByID(runtime *plugin.Runtime, id string) (*mqlConfluentEnvironment, error) {
	if id == "" {
		return nil, nil
	}
	root, err := getConfluent(runtime)
	if err != nil {
		return nil, err
	}
	envs := root.GetEnvironments()
	if envs.Error != nil {
		return nil, envs.Error
	}
	for _, raw := range envs.Data {
		env, ok := raw.(*mqlConfluentEnvironment)
		if !ok {
			continue
		}
		if env.GetId().Data == id {
			return env, nil
		}
	}
	return nil, nil
}

func (r *mqlConfluentEnvironment) kafkaClusters() ([]any, error) {
	root, err := getConfluent(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	clusters := root.GetKafkaClusters()
	if clusters.Error != nil {
		return nil, clusters.Error
	}

	envID := r.GetId().Data
	res := []any{}
	for _, raw := range clusters.Data {
		cluster, ok := raw.(*mqlConfluentKafkaCluster)
		if !ok {
			continue
		}
		if cluster.cachedEnvironmentID == envID {
			res = append(res, cluster)
		}
	}
	return res, nil
}

func (r *mqlConfluentEnvironment) schemaRegistryClusters() ([]any, error) {
	root, err := getConfluent(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	clusters := root.GetSchemaRegistryClusters()
	if clusters.Error != nil {
		return nil, clusters.Error
	}

	envID := r.GetId().Data
	res := []any{}
	for _, raw := range clusters.Data {
		cluster, ok := raw.(*mqlConfluentSchemaRegistryCluster)
		if !ok {
			continue
		}
		if cluster.cachedEnvironmentID == envID {
			res = append(res, cluster)
		}
	}
	return res, nil
}

func (r *mqlConfluentEnvironment) networks() ([]any, error) {
	root, err := getConfluent(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	networks := root.GetNetworks()
	if networks.Error != nil {
		return nil, networks.Error
	}

	envID := r.GetId().Data
	res := []any{}
	for _, raw := range networks.Data {
		network, ok := raw.(*mqlConfluentNetwork)
		if !ok {
			continue
		}
		if network.cachedEnvironmentID == envID {
			res = append(res, network)
		}
	}
	return res, nil
}
