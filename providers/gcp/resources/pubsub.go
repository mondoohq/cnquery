// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/gcp/connection"
	"go.mondoo.com/cnquery/v11/types"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectPubsubService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	return fmt.Sprintf("%s/gcp.project.pubsubService", projectId), nil
}

func initGcpProjectPubsubService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.GcpConnection)

	projectId := conn.ResourceID()
	args["projectId"] = llx.StringData(projectId)

	return args, nil, nil
}

func (g *mqlGcpProject) pubsub() (*mqlGcpProjectPubsubService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.pubsubService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectPubsubService), nil
}

func (g *mqlGcpProjectPubsubServiceTopic) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	name := g.Name.Data
	return fmt.Sprintf("%s/%s", projectId, name), nil
}

func (g *mqlGcpProjectPubsubServiceTopicConfig) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	if g.TopicName.Error != nil {
		return "", g.TopicName.Error
	}
	topicName := g.TopicName.Data
	return pubsubConfigId(projectId, topicName), nil
}

func (g *mqlGcpProjectPubsubServiceTopicConfigMessagestoragepolicy) id() (string, error) {
	if g.ConfigId.Error != nil {
		return "", g.ConfigId.Error
	}
	configId := g.ConfigId.Data
	return fmt.Sprintf("%s/messageStoragePolicy", configId), nil
}

func (g *mqlGcpProjectPubsubServiceSubscription) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	name := g.Name.Data
	return fmt.Sprintf("%s/%s", projectId, name), nil
}

func (g *mqlGcpProjectPubsubServiceSubscriptionConfig) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	if g.SubscriptionName.Error != nil {
		return "", g.SubscriptionName.Error
	}
	subscriptionName := g.SubscriptionName.Data
	return pubsubConfigId(projectId, subscriptionName), nil
}

func (g *mqlGcpProjectPubsubServiceSubscriptionConfigPushconfig) id() (string, error) {
	if g.ConfigId.Error != nil {
		return "", g.ConfigId.Error
	}
	configId := g.ConfigId.Data
	return fmt.Sprintf("%s/pushConfig", configId), nil
}

func (g *mqlGcpProjectPubsubServiceSnapshot) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	name := g.Name.Data
	return fmt.Sprintf("%s/%s", projectId, name), nil
}

func (g *mqlGcpProjectPubsubService) topics() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(pubsub.ScopePubSub)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	pubsubSvc, err := pubsub.NewClient(ctx, projectId, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer pubsubSvc.Close()

	var topics []interface{}

	it := pubsubSvc.Topics(ctx)
	for {
		t, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		mqlTopic, err := CreateResource(g.MqlRuntime, "gcp.project.pubsubService.topic", map[string]*llx.RawData{
			"projectId": llx.StringData(projectId),
			"name":      llx.StringData(t.ID()),
		})
		if err != nil {
			return nil, err
		}
		topics = append(topics, mqlTopic)
	}

	return topics, nil
}

func (g *mqlGcpProjectPubsubServiceTopic) config() (*mqlGcpProjectPubsubServiceTopicConfig, error) {
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	name := g.Name.Data

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(pubsub.ScopePubSub)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	pubsubSvc, err := pubsub.NewClient(ctx, projectId, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer pubsubSvc.Close()

	t := pubsubSvc.Topic(name)
	cfg, err := t.Config(ctx)
	if err != nil {
		return nil, err
	}

	messageStoragePolicy, err := CreateResource(g.MqlRuntime, "gcp.project.pubsubService.topic.config.messagestoragepolicy", map[string]*llx.RawData{
		"configId":                  llx.StringData(pubsubConfigId(projectId, t.ID())),
		"allowedPersistenceRegions": llx.ArrayData(convert.SliceAnyToInterface(cfg.MessageStoragePolicy.AllowedPersistenceRegions), types.String),
	})
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(g.MqlRuntime, "gcp.project.pubsubService.topic.config", map[string]*llx.RawData{
		"projectId":            llx.StringData(projectId),
		"topicName":            llx.StringData(t.ID()),
		"labels":               llx.MapData(convert.MapToInterfaceMap(cfg.Labels), types.String),
		"kmsKeyName":           llx.StringData(cfg.KMSKeyName),
		"messageStoragePolicy": llx.ResourceData(messageStoragePolicy, "gcp.project.pubsubService.topic.config.messagestoragepolicy"),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectPubsubServiceTopicConfig), nil
}

func (g *mqlGcpProjectPubsubService) subscriptions() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(pubsub.ScopePubSub)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	pubsubSvc, err := pubsub.NewClient(ctx, projectId, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer pubsubSvc.Close()

	var subs []interface{}

	it := pubsubSvc.Subscriptions(ctx)
	for {
		s, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		mqlSub, err := CreateResource(g.MqlRuntime, "gcp.project.pubsubService.subscription", map[string]*llx.RawData{
			"projectId": llx.StringData(projectId),
			"name":      llx.StringData(s.ID()),
		})
		if err != nil {
			return nil, err
		}
		subs = append(subs, mqlSub)
	}

	return subs, nil
}

func (g *mqlGcpProjectPubsubServiceSubscription) config() (*mqlGcpProjectPubsubServiceSubscriptionConfig, error) {
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	name := g.Name.Data

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(pubsub.ScopePubSub)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	pubsubSvc, err := pubsub.NewClient(ctx, projectId, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer pubsubSvc.Close()

	s := pubsubSvc.Subscription(name)
	cfg, err := s.Config(ctx)
	if err != nil {
		return nil, err
	}

	topic, err := CreateResource(g.MqlRuntime, "gcp.project.pubsubService.topic", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
		"name":      llx.StringData(cfg.Topic.ID()),
	})

	pushConfig, err := CreateResource(g.MqlRuntime, "gcp.project.pubsubService.subscription.config.pushconfig", map[string]*llx.RawData{
		"configId":   llx.StringData(pubsubConfigId(projectId, s.ID())),
		"endpoint":   llx.StringData(cfg.PushConfig.Endpoint),
		"attributes": llx.MapData(convert.MapToInterfaceMap(cfg.PushConfig.Attributes), types.String),
	})
	if err != nil {
		return nil, err
	}
	var expPolicy time.Time
	if exp, ok := cfg.ExpirationPolicy.(time.Duration); ok {
		expPolicy = llx.DurationToTime(int64(exp.Seconds()))
	}
	res, err := CreateResource(g.MqlRuntime, "gcp.project.pubsubService.subscription.config", map[string]*llx.RawData{
		"projectId":           llx.StringData(projectId),
		"subscriptionName":    llx.StringData(s.ID()),
		"topic":               llx.ResourceData(topic, "gcp.project.pubsubService.topic"),
		"pushConfig":          llx.ResourceData(pushConfig, "gcp.project.pubsubService.subscription.config.pushconfig"),
		"ackDeadline":         llx.TimeData(llx.DurationToTime(int64(cfg.AckDeadline.Seconds()))),
		"retainAckedMessages": llx.BoolData(cfg.RetainAckedMessages),
		"retentionDuration":   llx.TimeData(llx.DurationToTime(int64(cfg.RetentionDuration.Seconds()))),
		"expirationPolicy":    llx.TimeData(expPolicy),
		"labels":              llx.MapData(convert.MapToInterfaceMap(cfg.Labels), types.String),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectPubsubServiceSubscriptionConfig), nil
}

func (g *mqlGcpProjectPubsubService) snapshots() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(pubsub.ScopePubSub)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	pubsubSvc, err := pubsub.NewClient(ctx, projectId, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer pubsubSvc.Close()

	var subs []interface{}

	it := pubsubSvc.Snapshots(ctx)
	for {
		s, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		topic, err := CreateResource(g.MqlRuntime, "gcp.project.pubsubService.topic", map[string]*llx.RawData{
			"id":        llx.StringData(s.Topic.ID()),
			"projectId": llx.StringData(projectId),
			"name":      llx.StringData(s.Topic.ID()),
		})
		if err != nil {
			return nil, err
		}

		mqlSub, err := CreateResource(g.MqlRuntime, "gcp.project.pubsubService.snapshot", map[string]*llx.RawData{
			"id":         llx.StringData(s.ID()),
			"projectId":  llx.StringData(projectId),
			"name":       llx.StringData(s.ID()),
			"topic":      llx.ResourceData(topic, "gcp.project.pubsubService.topic"),
			"expiration": llx.TimeData(s.Expiration),
		})
		if err != nil {
			return nil, err
		}
		subs = append(subs, mqlSub)
	}

	return subs, nil
}

func pubsubConfigId(projectId, parentName string) string {
	return fmt.Sprintf("%s/%s/config", projectId, parentName)
}
