// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"

	cmkv2 "github.com/confluentinc/ccloud-sdk-go-v2/cmk/v2"
	"github.com/rs/zerolog/log"
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

// clusterConfigRaw is a sidecar decode of the cluster type block, read from the
// same bytes the SDK's discriminated union reads.
//
// It exists for two reasons the union cannot cover. A tier the union does not
// recognize leaves every variant nil and reports no error, which would blank
// the cluster type, the sizing and the zones on a cluster that reported all
// three. And CmkV2Dedicated types `cku` as a value rather than a pointer, so
// the union cannot tell a cluster that reported no cku from one that reported
// zero; this can, and the schema promises null for the former.
type clusterConfigRaw struct {
	Kind    string   `json:"kind"`
	Cku     *int32   `json:"cku"`
	MaxEcku *int32   `json:"max_ecku"`
	Zones   []string `json:"zones"`
}

// clusterConfig is the resolved cluster type block, whichever source answered.
type clusterConfig struct {
	Kind    string
	Cku     *int32
	MaxEcku *int32
	Zones   []string
}

// kafkaClusterRecord is one entry of the Kafka clusters listing.
type kafkaClusterRecord struct {
	ID       string                    `json:"id"`
	Metadata objectMeta                `json:"metadata"`
	Spec     *cmkv2.CmkV2ClusterSpec   `json:"spec"`
	Status   *cmkv2.CmkV2ClusterStatus `json:"status"`

	// configSidecar holds the cluster type block as it arrived. It is set only
	// when the record was decoded from JSON; a record built in code carries
	// none, and its configuration is then read from the union alone.
	configSidecar *clusterConfigRaw
}

// UnmarshalJSON decodes a cluster and keeps a second, untyped reading of the
// cluster type block alongside the SDK's discriminated union.
func (r *kafkaClusterRecord) UnmarshalJSON(data []byte) error {
	// The alias sheds the methods, which is what stops this recursing.
	type alias kafkaClusterRecord
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*r = kafkaClusterRecord(decoded)

	var probe struct {
		Spec *struct {
			Config json.RawMessage `json:"config"`
		} `json:"spec"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return err
	}
	if probe.Spec == nil || len(probe.Spec.Config) == 0 {
		return nil
	}

	var raw clusterConfigRaw
	if err := json.Unmarshal(probe.Spec.Config, &raw); err != nil {
		// The block is present but unreadable. That is worth saying out loud,
		// and it is not worth failing the cluster over: everything outside the
		// cluster type block still decoded.
		log.Warn().Err(err).Msg("confluent> could not read the cluster type block")
		return nil
	}
	r.configSidecar = &raw
	return nil
}

// configVariant returns the cluster type variant the union matched, or nil when
// it matched none.
//
// CmkV2ClusterSpecConfigOneOf.UnmarshalJSON switches on the `kind`
// discriminator and, when no case matches, returns nil with every variant left
// nil and no error at all. A tier Confluent adds after this SDK release lands
// in exactly that state.
func configVariant(config *cmkv2.CmkV2ClusterSpecConfigOneOf) any {
	if config == nil {
		return nil
	}
	switch {
	case config.CmkV2Basic != nil:
		return config.CmkV2Basic
	case config.CmkV2Standard != nil:
		return config.CmkV2Standard
	case config.CmkV2Dedicated != nil:
		return config.CmkV2Dedicated
	case config.CmkV2Enterprise != nil:
		return config.CmkV2Enterprise
	case config.CmkV2Freight != nil:
		return config.CmkV2Freight
	}
	return nil
}

// clusterConfigOf resolves the cluster type block out of the two readings the
// record carries.
//
// The union supplies the shape, which is where the vendor's field tags earn
// their place. The sidecar supplies `cku`, because the union types it as a
// value and so cannot report an absent one as null. When the union recognized
// no variant the sidecar supplies everything, and that is said out loud rather
// than reported as a cluster with no type, no size and no zones.
func clusterConfigOf(record *kafkaClusterRecord) clusterConfig {
	if record == nil || record.Spec == nil {
		return clusterConfig{}
	}
	variant := configVariant(record.Spec.Config)

	if variant == nil {
		if record.Spec.Config != nil {
			kind := ""
			if record.configSidecar != nil {
				kind = record.configSidecar.Kind
			}
			log.Warn().Str("kind", kind).
				Msg("confluent> unrecognized Kafka cluster type; reading the cluster type block as it arrived")
		}
		if record.configSidecar == nil {
			return clusterConfig{}
		}
		return clusterConfig{
			Kind:    record.configSidecar.Kind,
			Cku:     record.configSidecar.Cku,
			MaxEcku: record.configSidecar.MaxEcku,
			Zones:   record.configSidecar.Zones,
		}
	}

	out := clusterConfig{}
	switch typed := variant.(type) {
	case *cmkv2.CmkV2Basic:
		out.Kind, out.MaxEcku = typed.Kind, typed.MaxEcku
	case *cmkv2.CmkV2Standard:
		out.Kind, out.MaxEcku = typed.Kind, typed.MaxEcku
	case *cmkv2.CmkV2Enterprise:
		out.Kind, out.MaxEcku = typed.Kind, typed.MaxEcku
	case *cmkv2.CmkV2Freight:
		out.Kind, out.MaxEcku, out.Zones = typed.Kind, typed.MaxEcku, typed.GetZones()
	case *cmkv2.CmkV2Dedicated:
		out.Kind, out.Zones = typed.Kind, typed.GetZones()
		// Only a record built in code reaches for the union's cku. A decoded
		// one takes it from the sidecar below, which can report it as absent.
		if record.configSidecar == nil {
			cku := typed.Cku
			out.Cku = &cku
		}
	}
	if record.configSidecar != nil {
		out.Cku = record.configSidecar.Cku
	}
	return out
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
func endpointViews(endpoints map[string]cmkv2.CmkV2Endpoints) []endpointView {
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
			HTTPEndpoint:      endpoint.HttpEndpoint,
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
	return clusterConfigOf(record).Cku
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
		spec = &cmkv2.CmkV2ClusterSpec{}
	}

	views := endpointViews(spec.GetEndpoints())
	preferred := preferredEndpoint(views)

	bootstrap := preferred.BootstrapEndpoint
	rest := preferred.HTTPEndpoint
	// Responses that predate the endpoints map carry a single endpoint pair at
	// the top of the spec. It is the only endpoint information such a response
	// holds, so it is read when the map is absent.
	//
	// Both are read through the getters rather than the pointers: the SDK types
	// every scalar as optional, and a naive read would turn today's empty
	// string into a null on every response that omits them.
	if len(views) == 0 {
		bootstrap = spec.GetKafkaBootstrapEndpoint()
		rest = spec.GetHttpEndpoint()
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

	config := clusterConfigOf(record)

	envID := refID(spec.Environment)
	if envID == "" {
		envID = fallbackEnvID
	}

	phase := ""
	if record.Status != nil {
		phase = record.Status.Phase
	}

	res, err := CreateResource(runtime, "confluent.kafkaCluster", map[string]*llx.RawData{
		"__id":                      llx.StringData(record.ID),
		"id":                        llx.StringData(record.ID),
		"displayName":               llx.StringData(spec.GetDisplayName()),
		"resourceName":              llx.StringData(record.Metadata.ResourceName),
		"cloud":                     llx.StringData(spec.GetCloud()),
		"region":                    llx.StringData(spec.GetRegion()),
		"availability":              llx.StringData(spec.GetAvailability()),
		"clusterType":               llx.StringData(config.Kind),
		"cku":                       optionalInt(clusterCku(record)),
		"maxEcku":                   optionalInt(config.MaxEcku),
		"zones":                     llx.ArrayData(strSliceToAny(config.Zones), types.String),
		"deletionProtection":        llx.BoolData(spec.GetDeletionProtection()),
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
