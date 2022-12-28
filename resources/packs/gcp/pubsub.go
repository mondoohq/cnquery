package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectPubsub) id() (string, error) {
	return "gcp.project.pubsub", nil
}

func (g *mqlGcpProjectPubsub) init(args *resources.Args) (*resources.Args, GcpProjectPubsub, error) {
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

	return g.MotorRuntime.CreateResource("gcp.project.pubsub",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectPubsubTopic) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpProjectPubsubTopicConfig) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpProjectPubsubTopicConfigMessagestoragepolicy) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpProjectPubsub) GetTopics() ([]interface{}, error) {
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

	var topics []interface{}

	it := pubsubSvc.Topics(ctx)
	for {
		t, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// TODO: Handle error.
			return nil, err
		}
		mqlTopic, err := g.MotorRuntime.CreateResource("gcp.project.pubsub.topic",
			"id", t.ID(),
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

func (g *mqlGcpProjectPubsubTopic) GetConfig() (interface{}, error) {
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

	t := pubsubSvc.Topic(name)
	cfg, err := t.Config(ctx)
	if err != nil {
		return nil, err
	}

	messageStoragePolicy, err := g.MotorRuntime.CreateResource("gcp.project.pubsub.topic.config.messagestoragepolicy",
		"id", fmt.Sprintf("%s/config/messagestoragepolicy", t.ID()),
		"allowedPersistenceRegions", core.StrSliceToInterface(cfg.MessageStoragePolicy.AllowedPersistenceRegions),
	)
	if err != nil {
		return nil, err
	}
	return g.MotorRuntime.CreateResource("gcp.project.pubsub.topic.config",
		"id", fmt.Sprintf("%s/config", t.ID()),
		"labels", core.StrMapToInterface(cfg.Labels),
		"kmsKeyName", cfg.KMSKeyName,
		"messageStoragePolicy", messageStoragePolicy,
	)
}
