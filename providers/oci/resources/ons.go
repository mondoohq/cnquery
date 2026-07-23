// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/ons"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
)

func (o *mqlOciOns) id() (string, error) {
	return "oci.ons", nil
}

func (o *mqlOciOns) topics() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	return ociRunRegionPool(o.getTopics(conn, list.Data))
}

func (o *mqlOciOns) getTopicsForRegion(ctx context.Context, client *ons.NotificationControlPlaneClient, compartmentID string) ([]ons.NotificationTopicSummary, error) {
	topics := []ons.NotificationTopicSummary{}
	var page *string
	for {
		request := ons.ListTopicsRequest{
			CompartmentId: common.String(compartmentID),
			Page:          page,
		}

		response, err := client.ListTopics(ctx, request)
		if err != nil {
			return nil, err
		}

		topics = append(topics, response.Items...)

		if response.OpcNextPage == nil {
			break
		}

		page = response.OpcNextPage
	}

	return topics, nil
}

func (o *mqlOciOns) getTopics(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionResource.Id.Data)

			svc, err := conn.NotificationControlPlaneClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var res []any
			topics, err := o.getTopicsForRegion(ctx, svc, conn.TenantID())
			if err != nil {
				return nil, err
			}

			for i := range topics {
				topic := topics[i]

				var created *time.Time
				if topic.TimeCreated != nil {
					created = &topic.TimeCreated.Time
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.ons.topic", map[string]*llx.RawData{
					"id":            llx.StringDataPtr(topic.TopicId),
					"name":          llx.StringDataPtr(topic.Name),
					"description":   llx.StringDataPtr(topic.Description),
					"compartmentID": llx.StringDataPtr(topic.CompartmentId),
					"state":         llx.StringData(string(topic.LifecycleState)),
					"created":       llx.TimeDataPtr(created),
				})
				if err != nil {
					return nil, err
				}

				mqlTopic := mqlInstance.(*mqlOciOnsTopic)
				mqlTopic.region = regionResource.Id.Data

				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciOnsTopicInternal struct {
	region string
}

func initOciOnsTopic(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	idVal := ociArgString(args, "id")
	if idVal == "" {
		return nil, nil, errors.New("id required to fetch oci.ons.topic")
	}

	obj, err := CreateResource(runtime, "oci.ons", nil)
	if err != nil {
		return nil, nil, err
	}
	o := obj.(*mqlOciOns)

	rawTopics := o.GetTopics()
	if rawTopics.Error != nil {
		return nil, nil, rawTopics.Error
	}

	for _, raw := range rawTopics.Data {
		topic := raw.(*mqlOciOnsTopic)
		if topic.Id.Data == idVal {
			return args, topic, nil
		}
	}

	return nil, nil, errors.New("oci.ons.topic not found: " + idVal)
}

func (o *mqlOciOnsTopic) id() (string, error) {
	return "oci.ons.topic/" + o.Id.Data, nil
}

func (o *mqlOciOnsTopic) subscriptions() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	client, err := conn.NotificationDataPlaneClient(o.region)
	if err != nil {
		return nil, err
	}

	topicId := o.Id.Data
	ctx := context.Background()

	subs := []ons.SubscriptionSummary{}
	var page *string
	for {
		request := ons.ListSubscriptionsRequest{
			CompartmentId: common.String(conn.TenantID()),
			TopicId:       common.String(topicId),
			Page:          page,
		}

		response, err := client.ListSubscriptions(ctx, request)
		if err != nil {
			return nil, err
		}

		subs = append(subs, response.Items...)

		if response.OpcNextPage == nil {
			break
		}

		page = response.OpcNextPage
	}

	res := make([]any, 0, len(subs))
	for i := range subs {
		sub := subs[i]

		var created *time.Time
		if sub.CreatedTime != nil {
			t := time.UnixMilli(*sub.CreatedTime)
			created = &t
		}

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.ons.subscription", map[string]*llx.RawData{
			"id":       llx.StringDataPtr(sub.Id),
			"protocol": llx.StringDataPtr(sub.Protocol),
			"endpoint": llx.StringDataPtr(sub.Endpoint),
			"state":    llx.StringData(string(sub.LifecycleState)),
			"created":  llx.TimeDataPtr(created),
		})
		if err != nil {
			return nil, err
		}
		mqlInstance.(*mqlOciOnsSubscription).cacheTopicId = stringValue(sub.TopicId)
		res = append(res, mqlInstance)
	}

	return res, nil
}

type mqlOciOnsSubscriptionInternal struct {
	cacheTopicId string
}

func (o *mqlOciOnsSubscription) id() (string, error) {
	return "oci.ons.subscription/" + o.Id.Data, nil
}

func (o *mqlOciOnsSubscription) topic() (*mqlOciOnsTopic, error) {
	if o.cacheTopicId == "" {
		o.Topic.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlTopic, err := NewResource(o.MqlRuntime, "oci.ons.topic", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheTopicId),
	})
	if err != nil {
		return nil, err
	}
	return mqlTopic.(*mqlOciOnsTopic), nil
}
