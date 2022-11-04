package okta

import (
	"context"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
)

func (o *mqlOkta) GetTrustedOrigins() ([]interface{}, error) {
	op, err := oktaProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client := op.Client()

	slice, resp, err := client.TrustedOrigin.ListOrigins(
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
	appendEntry := func(datalist []*okta.TrustedOrigin) error {
		for i := range datalist {
			entry := datalist[i]
			r, err := newMqlOktaTrustedOrigin(o.MotorRuntime, entry)
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
		var slice []*okta.TrustedOrigin
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

func newMqlOktaTrustedOrigin(runtime *resources.Runtime, entry *okta.TrustedOrigin) (interface{}, error) {
	scopes, err := core.JsonToDict(entry.Scopes)
	if err != nil {
		return nil, err
	}

	return runtime.CreateResource("okta.trustedOrigin",
		"id", entry.Id,
		"name", entry.Name,
		"origin", entry.Origin,
		"created", entry.Created,
		"createdBy", entry.CreatedBy,
		"lastUpdated", entry.LastUpdated,
		"lastUpdatedBy", entry.LastUpdatedBy,
		"scopes", scopes,
		"status", entry.Status,
	)
}

func (o *mqlOktaTrustedOrigin) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "okta.trustedOriogin/" + id, nil
}
