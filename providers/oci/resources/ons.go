// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/ons"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v12/providers/oci/connection"
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

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getTopics(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
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
			t := time.Unix(0, *sub.CreatedTime*int64(time.Millisecond))
			created = &t
		}

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.ons.subscription", map[string]*llx.RawData{
			"id":       llx.StringDataPtr(sub.Id),
			"topicId":  llx.StringDataPtr(sub.TopicId),
			"protocol": llx.StringDataPtr(sub.Protocol),
			"endpoint": llx.StringDataPtr(sub.Endpoint),
			"state":    llx.StringData(string(sub.LifecycleState)),
			"created":  llx.TimeDataPtr(created),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (o *mqlOciOnsSubscription) id() (string, error) {
	return "oci.ons.subscription/" + o.Id.Data, nil
}
