package gcp

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectPubsubService) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/gcp.project.pubsubService", projectId), nil
}

func (g *mqlGcpProjectPubsubService) init(args *resources.Args) (*resources.Args, GcpProjectPubsubService, error) {
	if len(*args) > 0 {
		return args, nil, nil
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	projectId := provider.ResourceID()
	(*args)["projectId"] = projectId

	return args, nil, nil
}

func (g *mqlGcpProject) GetPubsub() (interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.project.pubsubService",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectPubsubServiceTopic) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", projectId, name), nil
}

func (g *mqlGcpProjectPubsubServiceTopicConfig) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	topicName, err := g.TopicName()
	if err != nil {
		return "", err
	}
	return pubsubConfigId(projectId, topicName), nil
}

func (g *mqlGcpProjectPubsubServiceTopicConfigMessagestoragepolicy) id() (string, error) {
	configId, err := g.ConfigId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/messageStoragePolicy", configId), nil
}

func (g *mqlGcpProjectPubsubServiceSubscription) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", projectId, name), nil
}

func (g *mqlGcpProjectPubsubServiceSubscriptionConfig) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	subscriptionName, err := g.SubscriptionName()
	if err != nil {
		return "", err
	}
	return pubsubConfigId(projectId, subscriptionName), nil
}

func (g *mqlGcpProjectPubsubServiceSubscriptionConfigPushconfig) id() (string, error) {
	configId, err := g.ConfigId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/pushConfig", configId), nil
}

func (g *mqlGcpProjectPubsubServiceSnapshot) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", projectId, name), nil
}

func (g *mqlGcpProjectPubsubService) GetTopics() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(pubsub.ScopePubSub)
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
		mqlTopic, err := g.MotorRuntime.CreateResource("gcp.project.pubsubService.topic",
			"projectId", projectId,
			"name", t.ID(),
		)
		if err != nil {
			return nil, err
		}
		topics = append(topics, mqlTopic)
	}

	return topics, nil
}

func (g *mqlGcpProjectPubsubServiceTopic) GetConfig() (interface{}, error) {
	name, err := g.Name()
	if err != nil {
		return nil, err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(pubsub.ScopePubSub)
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

	messageStoragePolicy, err := g.MotorRuntime.CreateResource("gcp.project.pubsubService.topic.config.messagestoragepolicy",
		"configId", pubsubConfigId(projectId, t.ID()),
		"allowedPersistenceRegions", core.SliceToInterfaceSlice(cfg.MessageStoragePolicy.AllowedPersistenceRegions),
	)
	if err != nil {
		return nil, err
	}
	return g.MotorRuntime.CreateResource("gcp.project.pubsubService.topic.config",
		"projectId", projectId,
		"topicName", t.ID(),
		"labels", core.StrMapToInterface(cfg.Labels),
		"kmsKeyName", cfg.KMSKeyName,
		"messageStoragePolicy", messageStoragePolicy,
	)
}

func (g *mqlGcpProjectPubsubService) GetSubscriptions() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(pubsub.ScopePubSub)
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
		mqlSub, err := g.MotorRuntime.CreateResource("gcp.project.pubsubService.subscription",
			"projectId", projectId,
			"name", s.ID(),
		)
		if err != nil {
			return nil, err
		}
		subs = append(subs, mqlSub)
	}

	return subs, nil
}

func (g *mqlGcpProjectPubsubServiceSubscription) GetConfig() (interface{}, error) {
	name, err := g.Name()
	if err != nil {
		return nil, err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(pubsub.ScopePubSub)
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

	topic, err := g.MotorRuntime.CreateResource("gcp.project.pubsubService.topic",
		"projectId", projectId,
		"name", cfg.Topic.ID(),
	)

	pushConfig, err := g.MotorRuntime.CreateResource("gcp.project.pubsubService.subscription.config.pushconfig",
		"configId", pubsubConfigId(projectId, s.ID()),
		"endpoint", cfg.PushConfig.Endpoint,
		"attributes", core.StrMapToInterface(cfg.PushConfig.Attributes),
	)
	if err != nil {
		return nil, err
	}
	var expPolicy *time.Time
	if exp, ok := cfg.ExpirationPolicy.(time.Duration); ok {
		expPolicy = core.MqlTime(llx.DurationToTime(int64(exp.Seconds())))
	}
	return g.MotorRuntime.CreateResource("gcp.project.pubsubService.subscription.config",
		"projectId", projectId,
		"subscriptionName", s.ID(),
		"topic", topic,
		"pushConfig", pushConfig,
		"ackDeadline", core.MqlTime(llx.DurationToTime(int64(cfg.AckDeadline.Seconds()))),
		"retainAckedMessages", cfg.RetainAckedMessages,
		"retentionDuration", core.MqlTime(llx.DurationToTime(int64(cfg.RetentionDuration.Seconds()))),
		"expirationPolicy", expPolicy,
		"labels", core.StrMapToInterface(cfg.Labels),
	)
}

func (g *mqlGcpProjectPubsubService) GetSnapshots() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(pubsub.ScopePubSub)
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

		topic, err := g.MotorRuntime.CreateResource("gcp.project.pubsubService.topic",
			"id", s.Topic.ID(),
			"projectId", projectId,
			"name", s.Topic.ID(),
		)
		if err != nil {
			return nil, err
		}

		mqlSub, err := g.MotorRuntime.CreateResource("gcp.project.pubsubService.snapshot",
			"id", s.ID(),
			"projectId", projectId,
			"name", s.ID(),
			"topic", topic,
			"expirtaion", core.MqlTime(s.Expiration),
		)
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
