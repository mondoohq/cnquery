package okta

import (
	"context"

	"go.mondoo.com/cnquery/resources/packs/core"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.mondoo.com/cnquery/resources"
)

func (o *mqlOkta) GetApplications() ([]interface{}, error) {
	op, err := oktaProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client := op.Client()
	appSetSlice, resp, err := client.Application.ListApplications(
		ctx,
		query.NewQueryParams(
			query.WithLimit(queryLimit),
		),
	)
	if err != nil {
		return nil, err
	}

	if len(appSetSlice) == 0 {
		return nil, nil
	}

	list := []interface{}{}
	appendEntry := func(datalist []okta.App) error {
		for i := range datalist {
			entry := datalist[i]
			if entry.IsApplicationInstance() {
				app := entry.(*okta.Application)
				r, err := newMqlOktaApplication(o.MotorRuntime, app)
				if err != nil {
					return err
				}
				list = append(list, r)
			}
		}
		return nil
	}

	err = appendEntry(appSetSlice)
	if err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var userSetSlice []okta.App
		resp, err = resp.Next(ctx, &userSetSlice)
		if err != nil {
			return nil, err
		}
		err = appendEntry(userSetSlice)
		if err != nil {
			return nil, err
		}
	}
	return list, nil
}

func newMqlOktaApplication(runtime *resources.Runtime, entry *okta.Application) (interface{}, error) {
	credentials, err := core.JsonToDict(entry.Credentials)
	if err != nil {
		return nil, err
	}

	licensing, err := core.JsonToDict(entry.Licensing)
	if err != nil {
		return nil, err
	}

	profile, err := core.JsonToDict(entry.Profile)
	if err != nil {
		return nil, err
	}

	settings, err := core.JsonToDict(entry.Settings)
	if err != nil {
		return nil, err
	}

	visibility, err := core.JsonToDict(entry.Visibility)
	if err != nil {
		return nil, err
	}

	return runtime.CreateResource("okta.application",
		"id", entry.Id,
		"name", entry.Name,
		"label", entry.Label,
		"created", entry.Created,
		"lastUpdated", entry.LastUpdated,
		"credentials", credentials,
		"features", core.StrSliceToInterface(entry.Features),
		"licensing", licensing,
		"profile", profile,
		"settings", settings,
		"signOnMode", entry.SignOnMode,
		"status", entry.Status,
		"visibility", visibility,
	)
}

func (o *mqlOktaApplication) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "okta.application/" + id, nil
}
