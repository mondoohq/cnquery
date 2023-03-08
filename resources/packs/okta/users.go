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

	profileDict := map[string]interface{}{}
	if user.Profile != nil {
		for k, v := range *user.Profile {
			profileDict[k] = v
		}
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
		"profile", profileDict,
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

func (o *mqlOktaRole) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "okta.role/" + id, nil
}

func (o *mqlOktaUser) GetRoles() ([]interface{}, error) {
	op, err := oktaProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client := op.Client()
	id, err := o.Id()
	if err != nil {
		return nil, err
	}
	roles, resp, err := client.User.ListAssignedRolesForUser(ctx, id, query.NewQueryParams(query.WithLimit(queryLimit)))
	if err != nil {
		return nil, err
	}
	res := []interface{}{}

	appendEntry := func(datalist []*okta.Role) error {
		for _, r := range datalist {
			mqlOktaRole, err := o.MotorRuntime.CreateResource("okta.role",
				"id", r.Id,
				"assignmentType", r.AssignmentType,
				"created", r.Created,
				"lastUpdated", r.LastUpdated,
				"label", r.Label,
				"status", r.Status,
				"description", r.Description,
				"type", r.Type)
			if err != nil {
				return err
			}
			res = append(res, mqlOktaRole)
		}
		return nil
	}
	err = appendEntry(roles)
	if err != nil {
		return nil, err
	}
	for resp != nil && resp.HasNextPage() {
		var userRoles []*okta.Role
		resp, err = resp.Next(ctx, &userRoles)
		if err != nil {
			return nil, err
		}
		err = appendEntry(userRoles)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}
