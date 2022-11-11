package okta

import (
	"context"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (o *mqlOkta) GetUsers() ([]interface{}, error) {
	op, err := oktaProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client := op.Client()
	userSetSlice, resp, err := client.User.ListUsers(
		ctx,
		query.NewQueryParams(
			query.WithLimit(queryLimit),
		),
	)
	if err != nil {
		return nil, err
	}

	if len(userSetSlice) == 0 {
		return nil, nil
	}

	list := []interface{}{}
	appendEntry := func(datalist []*okta.User) error {
		for i := range datalist {
			user := datalist[i]
			r, err := newMqlOktaUser(o.MotorRuntime, user)
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	err = appendEntry(userSetSlice)
	if err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var userSetSlice []*okta.User
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

func newMqlOktaUser(runtime *resources.Runtime, user *okta.User) (interface{}, error) {
	userType, err := core.JsonToDict(user.Type)
	if err != nil {
		return nil, err
	}
	credentials, err := core.JsonToDict(user.Credentials)
	if err != nil {
		return nil, err
	}

	return runtime.CreateResource("okta.user",
		"id", user.Id,
		"type", userType,
		"credentials", credentials,
		"activated", user.Activated,
		"created", user.Created,
		"lastLogin", user.LastLogin,
		"lastUpdated", user.LastUpdated,
		"passwordChanged", user.PasswordChanged,
		"profile", user.Profile,
		"status", user.Status,
		"statusChanged", user.StatusChanged,
		"transitioningToStatus", user.TransitioningToStatus,
	)
}

func (o *mqlOktaUser) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "okta.user/" + id, nil
}
