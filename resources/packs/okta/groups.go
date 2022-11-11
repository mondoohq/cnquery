package okta

import (
	"context"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (o *mqlOkta) GetGroups() ([]interface{}, error) {
	op, err := oktaProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client := op.Client()

	slice, resp, err := client.Group.ListGroups(
		ctx,
		query.NewQueryParams(
			query.WithLimit(queryLimit),
		),
	)
	if err != nil {
		return nil, err
	}

	if len(slice) == 0 {
		return nil, nil
	}

	list := []interface{}{}
	appendEntry := func(datalist []*okta.Group) error {
		for i := range datalist {
			entry := datalist[i]
			r, err := newMqlOktaGroup(o.MotorRuntime, entry)
			if err != nil {
				return err
			}
			list = append(list, r)
		}

		return nil
	}

	err = appendEntry(slice)
	if err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var slice []*okta.Group
		resp, err = resp.Next(ctx, &slice)
		if err != nil {
			return nil, err
		}
		err = appendEntry(slice)
		if err != nil {
			return nil, err
		}
	}
	return list, nil
}

func newMqlOktaGroup(runtime *resources.Runtime, entry *okta.Group) (interface{}, error) {
	profile, err := core.JsonToDict(entry.Profile)
	if err != nil {
		return nil, err
	}

	return runtime.CreateResource("okta.group",
		"id", entry.Id,
		"type", entry.Type,
		"created", entry.Created,
		"lastMembershipUpdated", entry.LastMembershipUpdated,
		"lastUpdated", entry.LastUpdated,
		"profile", profile,
	)
}

func (o *mqlOktaGroup) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "okta.group/" + id, nil
}
