// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/events"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v12/providers/oci/connection"
)

func (o *mqlOciEvents) id() (string, error) {
	return "oci.events", nil
}

func (o *mqlOciEvents) rules() ([]any, error) {
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
	poolOfJobs := jobpool.CreatePool(o.getEventRules(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciEvents) getEventRulesForRegion(ctx context.Context, client *events.EventsClient, compartmentID string) ([]events.RuleSummary, error) {
	rules := []events.RuleSummary{}
	var page *string
	for {
		request := events.ListRulesRequest{
			CompartmentId: common.String(compartmentID),
			Page:          page,
		}

		response, err := client.ListRules(ctx, request)
		if err != nil {
			return nil, err
		}

		rules = append(rules, response.Items...)

		if response.OpcNextPage == nil {
			break
		}

		page = response.OpcNextPage
	}

	return rules, nil
}

func (o *mqlOciEvents) getEventRules(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionResource.Id.Data)

			svc, err := conn.EventsClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var res []any
			rules, err := o.getEventRulesForRegion(ctx, svc, conn.TenantID())
			if err != nil {
				return nil, err
			}

			for i := range rules {
				rule := rules[i]

				var created *time.Time
				if rule.TimeCreated != nil {
					created = &rule.TimeCreated.Time
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.events.rule", map[string]*llx.RawData{
					"id":            llx.StringDataPtr(rule.Id),
					"name":          llx.StringDataPtr(rule.DisplayName),
					"description":   llx.StringDataPtr(rule.Description),
					"compartmentID": llx.StringDataPtr(rule.CompartmentId),
					"condition":     llx.StringDataPtr(rule.Condition),
					"isEnabled":     llx.BoolData(boolValue(rule.IsEnabled)),
					"state":         llx.StringData(string(rule.LifecycleState)),
					"created":       llx.TimeDataPtr(created),
				})
				if err != nil {
					return nil, err
				}

				mqlRule := mqlInstance.(*mqlOciEventsRule)
				mqlRule.region = regionResource.Id.Data

				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciEventsRuleInternal struct {
	rule   *events.Rule
	region string
}

func (o *mqlOciEventsRule) id() (string, error) {
	return "oci.events.rule/" + o.Id.Data, nil
}

func (o *mqlOciEventsRule) getRuleDetails() (*events.Rule, error) {
	if o.rule != nil {
		return o.rule, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	client, err := conn.EventsClient(o.region)
	if err != nil {
		return nil, err
	}

	response, err := client.GetRule(context.Background(), events.GetRuleRequest{
		RuleId: common.String(o.Id.Data),
	})
	if err != nil {
		return nil, err
	}

	o.rule = &response.Rule
	return o.rule, nil
}

func (o *mqlOciEventsRule) actions() ([]any, error) {
	rule, err := o.getRuleDetails()
	if err != nil {
		return nil, err
	}

	if rule.Actions == nil {
		return []any{}, nil
	}

	res := make([]any, 0, len(rule.Actions.Actions))
	for _, action := range rule.Actions.Actions {
		m := map[string]any{
			"id":        stringValue(action.GetId()),
			"isEnabled": boolValue(action.GetIsEnabled()),
			"state":     string(action.GetLifecycleState()),
		}

		switch a := action.(type) {
		case events.NotificationServiceAction:
			m["actionType"] = "ONS"
			m["topicId"] = stringValue(a.TopicId)
			m["description"] = stringValue(a.Description)
		case events.StreamingServiceAction:
			m["actionType"] = "OSS"
			m["streamId"] = stringValue(a.StreamId)
			m["description"] = stringValue(a.Description)
		case events.FaaSAction:
			m["actionType"] = "FAAS"
			m["functionId"] = stringValue(a.FunctionId)
			m["description"] = stringValue(a.Description)
		}

		res = append(res, m)
	}

	return res, nil
}
