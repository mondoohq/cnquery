// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"net/url"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/confluent/connection"
	"go.mondoo.com/mql/v13/types"
)

// mqlConfluentNetworkInternal caches the environment reference, which the
// payload carries as an identifier.
type mqlConfluentNetworkInternal struct {
	cachedEnvironmentID string
}

type networkSpecRecord struct {
	DisplayName     string           `json:"display_name"`
	Cloud           string           `json:"cloud"`
	Region          string           `json:"region"`
	ConnectionTypes []string         `json:"connection_types"`
	Cidr            string           `json:"cidr"`
	Zones           []string         `json:"zones"`
	Environment     *objectReference `json:"environment"`
	DnsConfig       *struct {
		Resolution string `json:"resolution"`
	} `json:"dns_config"`
}

type networkStatusRecord struct {
	Phase                    string   `json:"phase"`
	SupportedConnectionTypes []string `json:"supported_connection_types"`
	ActiveConnectionTypes    []string `json:"active_connection_types"`
	DnsDomain                string   `json:"dns_domain"`
}

type networkRecord struct {
	ID       string               `json:"id"`
	Metadata objectMeta           `json:"metadata"`
	Spec     *networkSpecRecord   `json:"spec"`
	Status   *networkStatusRecord `json:"status"`
}

func (r *mqlConfluent) networks() ([]any, error) {
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

		records, err := connection.GetPaged[networkRecord](context.Background(), conn,
			conn.CloudTarget(), "/networking/v1/networks", query)
		if err != nil {
			return nil, err
		}

		for i := range records {
			record := records[i]
			spec := record.Spec
			if spec == nil {
				spec = &networkSpecRecord{}
			}
			status := record.Status
			if status == nil {
				status = &networkStatusRecord{}
			}

			resolution := ""
			if spec.DnsConfig != nil {
				resolution = spec.DnsConfig.Resolution
			}

			mqlNetwork, err := CreateResource(r.MqlRuntime, "confluent.network", map[string]*llx.RawData{
				"__id":                     llx.StringData(record.ID),
				"id":                       llx.StringData(record.ID),
				"displayName":              llx.StringData(spec.DisplayName),
				"resourceName":             llx.StringData(record.Metadata.ResourceName),
				"cloud":                    llx.StringData(spec.Cloud),
				"region":                   llx.StringData(spec.Region),
				"cidr":                     llx.StringData(spec.Cidr),
				"zones":                    llx.ArrayData(strSliceToAny(spec.Zones), types.String),
				"connectionTypes":          llx.ArrayData(strSliceToAny(spec.ConnectionTypes), types.String),
				"activeConnectionTypes":    llx.ArrayData(strSliceToAny(status.ActiveConnectionTypes), types.String),
				"supportedConnectionTypes": llx.ArrayData(strSliceToAny(status.SupportedConnectionTypes), types.String),
				"dnsDomain":                llx.StringData(status.DnsDomain),
				"dnsResolution":            llx.StringData(resolution),
				"phase":                    llx.StringData(status.Phase),
				"createdAt":                llx.TimeDataPtr(record.Metadata.CreatedAt.Time()),
			})
			if err != nil {
				return nil, err
			}

			network := mqlNetwork.(*mqlConfluentNetwork)
			network.cachedEnvironmentID = refID(spec.Environment)
			if network.cachedEnvironmentID == "" {
				network.cachedEnvironmentID = envID
			}
			res = append(res, network)
		}
	}
	return res, nil
}

func (r *mqlConfluentNetwork) environment() (*mqlConfluentEnvironment, error) {
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

func (r *mqlConfluentNetwork) kafkaClusters() ([]any, error) {
	root, err := getConfluent(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	clusters := root.GetKafkaClusters()
	if clusters.Error != nil {
		return nil, clusters.Error
	}

	networkID := r.GetId().Data
	res := []any{}
	for _, raw := range clusters.Data {
		cluster, ok := raw.(*mqlConfluentKafkaCluster)
		if !ok {
			continue
		}
		if cluster.cachedNetworkID == networkID {
			res = append(res, cluster)
		}
	}
	return res, nil
}

// networkByID resolves a network from the root resource's cached list.
func networkByID(runtime *plugin.Runtime, id string) (*mqlConfluentNetwork, error) {
	if id == "" {
		return nil, nil
	}
	root, err := getConfluent(runtime)
	if err != nil {
		return nil, err
	}
	networks := root.GetNetworks()
	if networks.Error != nil {
		return nil, networks.Error
	}
	for _, raw := range networks.Data {
		network, ok := raw.(*mqlConfluentNetwork)
		if !ok {
			continue
		}
		if network.GetId().Data == id {
			return network, nil
		}
	}
	return nil, nil
}
