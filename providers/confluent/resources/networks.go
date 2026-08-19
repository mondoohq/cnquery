// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"net/url"

	networkingv1 "github.com/confluentinc/ccloud-sdk-go-v2/networking/v1"
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

// networkStatusRecord is the status block of a network.
//
// It stays local rather than adopting NetworkingV1NetworkStatus, which carries
// an `idle_since` typed as *time.Time. This provider exposes no such field, but
// adopting the block would still let a timestamp it never reads fail the decode
// of every network in the listing.
type networkStatusRecord struct {
	Phase                    string   `json:"phase"`
	SupportedConnectionTypes []string `json:"supported_connection_types"`
	ActiveConnectionTypes    []string `json:"active_connection_types"`
	DnsDomain                string   `json:"dns_domain"`
}

type networkRecord struct {
	ID       string                                `json:"id"`
	Metadata objectMeta                            `json:"metadata"`
	Spec     *networkingv1.NetworkingV1NetworkSpec `json:"spec"`
	Status   *networkStatusRecord                  `json:"status"`
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
				spec = &networkingv1.NetworkingV1NetworkSpec{}
			}
			status := record.Status
			if status == nil {
				status = &networkStatusRecord{}
			}

			dnsConfig := spec.GetDnsConfig()
			resolution := dnsConfig.Resolution

			mqlNetwork, err := CreateResource(r.MqlRuntime, "confluent.network", map[string]*llx.RawData{
				"__id":                     llx.StringData(record.ID),
				"id":                       llx.StringData(record.ID),
				"displayName":              llx.StringData(spec.GetDisplayName()),
				"resourceName":             llx.StringData(record.Metadata.ResourceName),
				"cloud":                    llx.StringData(spec.GetCloud()),
				"region":                   llx.StringData(spec.GetRegion()),
				"cidr":                     llx.StringData(spec.GetCidr()),
				"zones":                    llx.ArrayData(strSliceToAny(spec.GetZones()), types.String),
				"connectionTypes":          llx.ArrayData(strSliceToAny(spec.GetConnectionTypes()), types.String),
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
