// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"net/url"
	"sort"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/confluent/connection"
	"go.mondoo.com/mql/v13/types"
)

// mqlConfluentKafkaClusterInternal caches the references the cluster payload
// carries as identifiers, so the typed accessors resolve them out of the
// listings the root resource already holds rather than fetching one object per
// cluster.
type mqlConfluentKafkaClusterInternal struct {
	cachedEnvironmentID   string
	cachedNetworkID       string
	cachedEncryptionKeyID string
	cachedRestEndpoint    string
}

// connectionTypePublic marks an endpoint served from the public internet. The
// other two connection types Confluent reports both mean private connectivity.
const (
	connectionTypePublic           = "PUBLIC"
	connectionTypePrivateLink      = "PRIVATE_LINK"
	connectionTypePrivateInterface = "PRIVATE_NETWORK_INTERFACE"
)

// clusterEndpointRecord is one entry of a cluster's endpoints map.
type clusterEndpointRecord struct {
	KafkaBootstrapEndpoint string `json:"kafka_bootstrap_endpoint"`
	HTTPEndpoint           string `json:"http_endpoint"`
	ConnectionType         string `json:"connection_type"`
}

// clusterConfigRecord is the cluster type block. Which of the numeric fields is
// populated depends on `kind`: a Dedicated cluster is sized by `cku`, every
// other type auto-scales up to `max_ecku`.
type clusterConfigRecord struct {
	Kind    string   `json:"kind"`
	Cku     *int32   `json:"cku"`
	MaxEcku *int32   `json:"max_ecku"`
	Zones   []string `json:"zones"`
}

type clusterSpecRecord struct {
	DisplayName        string                           `json:"display_name"`
	Availability       string                           `json:"availability"`
	Cloud              string                           `json:"cloud"`
	Region             string                           `json:"region"`
	Config             *clusterConfigRecord             `json:"config"`
	Endpoints          map[string]clusterEndpointRecord `json:"endpoints"`
	DeletionProtection *bool                            `json:"deletion_protection"`
	Environment        *objectReference                 `json:"environment"`
	Network            *objectReference                 `json:"network"`
	Byok               *objectReference                 `json:"byok"`

	// The two endpoint fields below are superseded by `endpoints`. They are
	// read only as a fallback, for responses that carry no endpoints map.
	LegacyBootstrapEndpoint string `json:"kafka_bootstrap_endpoint"`
	LegacyHTTPEndpoint      string `json:"http_endpoint"`
}

type clusterStatusRecord struct {
	Phase string `json:"phase"`
	Cku   *int32 `json:"cku"`
}

// kafkaClusterRecord is one entry of the Kafka clusters listing.
type kafkaClusterRecord struct {
	ID       string               `json:"id"`
	Metadata objectMeta           `json:"metadata"`
	Spec     *clusterSpecRecord   `json:"spec"`
	Status   *clusterStatusRecord `json:"status"`
}

// endpointView is one endpoint of a cluster with its access point identifier
// folded in, which the API keeps as the map key.
type endpointView struct {
	AccessPointID     string
	ConnectionType    string
	BootstrapEndpoint string
	HTTPEndpoint      string
}

// endpointViews flattens a cluster's endpoints map into a stable order. The map
// key is the access point identifier, which is part of the endpoint's identity
// and is otherwise lost.
func endpointViews(endpoints map[string]clusterEndpointRecord) []endpointView {
	if len(endpoints) == 0 {
		return nil
	}
	keys := make([]string, 0, len(endpoints))
	for key := range endpoints {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]endpointView, 0, len(keys))
	for _, key := range keys {
		endpoint := endpoints[key]
		out = append(out, endpointView{
			AccessPointID:     key,
			ConnectionType:    endpoint.ConnectionType,
			BootstrapEndpoint: endpoint.KafkaBootstrapEndpoint,
			HTTPEndpoint:      endpoint.HTTPEndpoint,
		})
	}
	return out
}

// preferredEndpoint picks the endpoint a client would reach the cluster on: the
// public one when the cluster has one, otherwise the first private one. A
// cluster with no endpoints at all yields the zero value.
func preferredEndpoint(views []endpointView) endpointView {
	for _, view := range views {
		if view.ConnectionType == connectionTypePublic {
			return view
		}
	}
	if len(views) > 0 {
		return views[0]
	}
	return endpointView{}
}

// endpointConnectionTypes lists the distinct connection types across a
// cluster's endpoints, in a stable order.
func endpointConnectionTypes(views []endpointView) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, view := range views {
		if view.ConnectionType == "" || seen[view.ConnectionType] {
			continue
		}
		seen[view.ConnectionType] = true
		out = append(out, view.ConnectionType)
	}
	sort.Strings(out)
	return out
}

// hasPublicEndpoint reports whether the cluster is reachable from the internet.
func hasPublicEndpoint(views []endpointView) bool {
	for _, view := range views {
		if view.ConnectionType == connectionTypePublic {
			return true
		}
	}
	return false
}

// hasPrivateEndpoint reports whether the cluster carries a private endpoint. A
// cluster can carry both, so this is not the negation of hasPublicEndpoint.
func hasPrivateEndpoint(views []endpointView) bool {
	for _, view := range views {
		if view.ConnectionType == connectionTypePrivateLink || view.ConnectionType == connectionTypePrivateInterface {
			return true
		}
	}
	return false
}

// clusterCku reports how many Confluent Kafka Units the cluster has. The status
// carries what is actually provisioned and the spec what was asked for, so the
// status wins where both are present.
func clusterCku(record *kafkaClusterRecord) *int32 {
	if record == nil {
		return nil
	}
	if record.Status != nil && record.Status.Cku != nil {
		return record.Status.Cku
	}
	if record.Spec != nil && record.Spec.Config != nil {
		return record.Spec.Config.Cku
	}
	return nil
}

func (r *mqlConfluent) kafkaClusters() ([]any, error) {
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

		records, err := connection.GetPaged[kafkaClusterRecord](context.Background(), conn,
			conn.CloudTarget(), "/cmk/v2/clusters", query)
		if err != nil {
			return nil, err
		}

		for i := range records {
			mqlCluster, err := newKafkaCluster(r.MqlRuntime, &records[i], envID)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlCluster)
		}
	}
	return res, nil
}

func newKafkaCluster(runtime *plugin.Runtime, record *kafkaClusterRecord, fallbackEnvID string) (*mqlConfluentKafkaCluster, error) {
	spec := record.Spec
	if spec == nil {
		spec = &clusterSpecRecord{}
	}

	views := endpointViews(spec.Endpoints)
	preferred := preferredEndpoint(views)

	bootstrap := preferred.BootstrapEndpoint
	rest := preferred.HTTPEndpoint
	// Responses that predate the endpoints map carry a single endpoint pair at
	// the top of the spec. It is the only endpoint information such a response
	// holds, so it is read when the map is absent.
	if len(views) == 0 {
		bootstrap = spec.LegacyBootstrapEndpoint
		rest = spec.LegacyHTTPEndpoint
	}

	// Without an endpoints map there is nothing that says how the cluster is
	// reached, and guessing from a hostname would report a public cluster as
	// private. The exposure fields stay null instead.
	connTypes := llx.NilData
	isPublic := llx.NilData
	hasPrivate := llx.NilData
	if len(views) > 0 {
		connTypes = llx.ArrayData(strSliceToAny(endpointConnectionTypes(views)), types.String)
		isPublic = llx.BoolData(hasPublicEndpoint(views))
		hasPrivate = llx.BoolData(hasPrivateEndpoint(views))
	}

	endpointDicts := make([]any, 0, len(views))
	for _, view := range views {
		endpointDicts = append(endpointDicts, map[string]any{
			"accessPointId":     view.AccessPointID,
			"connectionType":    view.ConnectionType,
			"bootstrapEndpoint": view.BootstrapEndpoint,
			"httpEndpoint":      view.HTTPEndpoint,
		})
	}

	config := spec.Config
	if config == nil {
		config = &clusterConfigRecord{}
	}

	envID := refID(spec.Environment)
	if envID == "" {
		envID = fallbackEnvID
	}

	deletionProtection := false
	if spec.DeletionProtection != nil {
		deletionProtection = *spec.DeletionProtection
	}

	phase := ""
	if record.Status != nil {
		phase = record.Status.Phase
	}

	res, err := CreateResource(runtime, "confluent.kafkaCluster", map[string]*llx.RawData{
		"__id":                      llx.StringData(record.ID),
		"id":                        llx.StringData(record.ID),
		"displayName":               llx.StringData(spec.DisplayName),
		"resourceName":              llx.StringData(record.Metadata.ResourceName),
		"cloud":                     llx.StringData(spec.Cloud),
		"region":                    llx.StringData(spec.Region),
		"availability":              llx.StringData(spec.Availability),
		"clusterType":               llx.StringData(config.Kind),
		"cku":                       optionalInt(clusterCku(record)),
		"maxEcku":                   optionalInt(config.MaxEcku),
		"zones":                     llx.ArrayData(strSliceToAny(config.Zones), types.String),
		"deletionProtection":        llx.BoolData(deletionProtection),
		"phase":                     llx.StringData(phase),
		"createdAt":                 llx.TimeDataPtr(record.Metadata.CreatedAt.Time()),
		"updatedAt":                 llx.TimeDataPtr(record.Metadata.UpdatedAt.Time()),
		"bootstrapEndpoint":         llx.StringData(bootstrap),
		"restEndpoint":              llx.StringData(rest),
		"endpoints":                 llx.ArrayData(endpointDicts, types.Dict),
		"connectionTypes":           connTypes,
		"isPublic":                  isPublic,
		"hasPrivateEndpoint":        hasPrivate,
		"customerManagedEncryption": llx.BoolData(refID(spec.Byok) != ""),
	})
	if err != nil {
		return nil, err
	}

	cluster := res.(*mqlConfluentKafkaCluster)
	cluster.cachedEnvironmentID = envID
	cluster.cachedNetworkID = refID(spec.Network)
	cluster.cachedEncryptionKeyID = refID(spec.Byok)
	cluster.cachedRestEndpoint = rest
	return cluster, nil
}

func (r *mqlConfluentKafkaCluster) environment() (*mqlConfluentEnvironment, error) {
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

func (r *mqlConfluentKafkaCluster) network() (*mqlConfluentNetwork, error) {
	if r.cachedNetworkID == "" {
		r.Network.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	network, err := networkByID(r.MqlRuntime, r.cachedNetworkID)
	if err != nil {
		return nil, err
	}
	if network == nil {
		r.Network.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return network, nil
}

func (r *mqlConfluentKafkaCluster) encryptionKey() (*mqlConfluentEncryptionKey, error) {
	if r.cachedEncryptionKeyID == "" {
		r.EncryptionKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	key, err := encryptionKeyByID(r.MqlRuntime, r.cachedEncryptionKeyID)
	if err != nil {
		return nil, err
	}
	if key == nil {
		r.EncryptionKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return key, nil
}

func (r *mqlConfluentKafkaCluster) apiKeys() ([]any, error) {
	root, err := getConfluent(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	keys := root.GetApiKeys()
	if keys.Error != nil {
		return nil, keys.Error
	}

	clusterID := r.GetId().Data
	res := []any{}
	for _, raw := range keys.Data {
		key, ok := raw.(*mqlConfluentApiKey)
		if !ok {
			continue
		}
		if key.cachedResourceID == clusterID {
			res = append(res, key)
		}
	}
	return res, nil
}

func (r *mqlConfluentKafkaCluster) roleBindings() ([]any, error) {
	root, err := getConfluent(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	bindings := root.GetRoleBindings()
	if bindings.Error != nil {
		return nil, bindings.Error
	}

	clusterID := r.GetId().Data
	res := []any{}
	for _, raw := range bindings.Data {
		binding, ok := raw.(*mqlConfluentRoleBinding)
		if !ok {
			continue
		}
		if binding.cachedClusterID == clusterID {
			res = append(res, binding)
		}
	}
	return res, nil
}

// kafkaClusterByID resolves a cluster from the root resource's cached list.
func kafkaClusterByID(runtime *plugin.Runtime, id string) (*mqlConfluentKafkaCluster, error) {
	if id == "" {
		return nil, nil
	}
	root, err := getConfluent(runtime)
	if err != nil {
		return nil, err
	}
	clusters := root.GetKafkaClusters()
	if clusters.Error != nil {
		return nil, clusters.Error
	}
	for _, raw := range clusters.Data {
		cluster, ok := raw.(*mqlConfluentKafkaCluster)
		if !ok {
			continue
		}
		if cluster.GetId().Data == id {
			return cluster, nil
		}
	}
	return nil, nil
}
