// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"net/url"
	"strings"
	"sync"

	kafkarestv3 "github.com/confluentinc/ccloud-sdk-go-v2/kafkarest/v3"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/confluent/connection"
)

// mqlConfluentTopicInternal caches what a topic needs to reach its own cluster,
// and memoizes the configuration read, which is a separate call per topic.
type mqlConfluentTopicInternal struct {
	cachedClusterID    string
	cachedRestEndpoint string

	configsOnce   sync.Once
	cachedConfigs map[string]string
	configsErr    error
}

// topicRecord is one entry of a cluster's topic listing.
type topicRecord = kafkarestv3.TopicData

// topicConfigRecord is one entry of a topic's configuration listing. A
// configuration the broker marks as sensitive carries a null value, which the
// SDK's NullableString keeps distinguishable from one set to the empty string.
type topicConfigRecord = kafkarestv3.TopicConfigData

// topicID builds the cache key of a topic. The topic name alone is not unique
// across an organization, since two clusters may both hold a topic of the same
// name, so the cluster is part of the identity.
func topicID(clusterID, topicName string) string {
	return clusterID + "/topic/" + url.QueryEscape(topicName)
}

func (r *mqlConfluentKafkaCluster) topics() ([]any, error) {
	conn, err := confluentConn(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	clusterID := r.GetId().Data
	target, err := conn.KafkaTarget(clusterID, r.cachedRestEndpoint)
	if err != nil {
		return nil, err
	}

	records, err := connection.GetPaged[topicRecord](context.Background(), conn, target,
		"/kafka/v3/clusters/"+url.PathEscape(clusterID)+"/topics", nil)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		record := records[i]

		mqlTopic, err := CreateResource(r.MqlRuntime, "confluent.topic", map[string]*llx.RawData{
			"__id":              llx.StringData(topicID(clusterID, record.TopicName)),
			"name":              llx.StringData(record.TopicName),
			"isInternal":        llx.BoolData(record.IsInternal),
			"partitionsCount":   llx.IntData(int64(record.PartitionsCount)),
			"replicationFactor": llx.IntData(int64(record.ReplicationFactor)),
		})
		if err != nil {
			return nil, err
		}

		topic := mqlTopic.(*mqlConfluentTopic)
		topic.cachedClusterID = clusterID
		topic.cachedRestEndpoint = r.cachedRestEndpoint
		res = append(res, topic)
	}
	return res, nil
}

func (r *mqlConfluentTopic) cluster() (*mqlConfluentKafkaCluster, error) {
	cluster, err := kafkaClusterByID(r.MqlRuntime, r.cachedClusterID)
	if err != nil {
		return nil, err
	}
	if cluster == nil {
		r.Cluster.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return cluster, nil
}

func (r *mqlConfluentTopic) configs() (map[string]any, error) {
	r.configsOnce.Do(func() {
		r.cachedConfigs, r.configsErr = r.fetchConfigs()
	})
	if r.configsErr != nil {
		return nil, r.configsErr
	}
	return strMapToAny(r.cachedConfigs), nil
}

func (r *mqlConfluentTopic) fetchConfigs() (map[string]string, error) {
	conn, err := confluentConn(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	target, err := conn.KafkaTarget(r.cachedClusterID, r.cachedRestEndpoint)
	if err != nil {
		return nil, err
	}

	path := "/kafka/v3/clusters/" + url.PathEscape(r.cachedClusterID) +
		"/topics/" + url.PathEscape(r.GetName().Data) + "/configs"

	records, err := connection.GetPaged[topicConfigRecord](context.Background(), conn, target, path, nil)
	if err != nil {
		return nil, err
	}

	out := make(map[string]string, len(records))
	for _, record := range records {
		// A sensitive configuration comes back with a null value. Recording the
		// name with an empty value keeps the configuration visible without
		// claiming it is set to nothing.
		value := record.Value.Get()
		if value == nil {
			out[record.Name] = ""
			continue
		}
		out[record.Name] = *value
	}
	return out, nil
}

func (r *mqlConfluentTopic) acls() ([]any, error) {
	cluster, err := kafkaClusterByID(r.MqlRuntime, r.cachedClusterID)
	if err != nil {
		return nil, err
	}
	if cluster == nil {
		return []any{}, nil
	}

	acls := cluster.GetAcls()
	if acls.Error != nil {
		return nil, acls.Error
	}

	name := r.GetName().Data
	res := []any{}
	for _, raw := range acls.Data {
		acl, ok := raw.(*mqlConfluentAcl)
		if !ok {
			continue
		}
		if aclMatchesTopic(acl.GetResourceType().Data, acl.GetPatternType().Data, acl.GetResourceName().Data, name) {
			res = append(res, acl)
		}
	}
	return res, nil
}

// aclMatchesTopic reports whether an access control entry reaches a topic by
// name. Kafka matches a LITERAL pattern by equality (with "*" standing for
// every topic) and a PREFIXED pattern by prefix. ANY and MATCH only appear on
// filters rather than on stored entries, and are treated the same way so an
// entry carrying one is not silently dropped.
func aclMatchesTopic(resourceType, patternType, resourceName, topicName string) bool {
	switch strings.ToUpper(resourceType) {
	case "TOPIC", "ANY":
	default:
		return false
	}

	switch strings.ToUpper(patternType) {
	case "LITERAL":
		return resourceName == "*" || resourceName == topicName
	case "PREFIXED":
		return strings.HasPrefix(topicName, resourceName)
	case "ANY", "MATCH":
		return resourceName == "*" || resourceName == topicName || strings.HasPrefix(topicName, resourceName)
	default:
		return false
	}
}
