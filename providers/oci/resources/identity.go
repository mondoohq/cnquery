// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v10/providers/oci/connection"
	"go.mondoo.com/cnquery/v10/types"
)

func (o *mqlOciIdentity) id() (string, error) {
	return "oci.identity", nil
}

func (o *mqlOciIdentity) users() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(o.getUsers(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
	}

	return res, nil
}

func (s *mqlOciIdentity) getUsersForRegion(ctx context.Context, identityClient *identity.IdentityClient, compartmentID string) ([]identity.User, error) {
	users := []identity.User{}
	var page *string
	for {
		request := identity.ListUsersRequest{
			CompartmentId: common.String(compartmentID),
			Page:          page,
		}

		response, err := identityClient.ListUsers(ctx, request)
		if err != nil {
			return nil, err
		}

		users = append(users, response.Items...)

		if response.OpcNextPage == nil {
			break
		}

		page = response.OpcNextPage
	}

	return users, nil
}

func (o *mqlOciIdentity) getUsers(conn *connection.OciConnection) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.GetRegions(ctx)
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionVal)

			svc, err := conn.IdentityClientWithRegion(*regionVal.RegionKey)
			if err != nil {
				return nil, err
			}

			var res []interface{}
			users, err := o.getUsersForRegion(ctx, svc, conn.TenantID())
			if err != nil {
				return nil, err
			}

			for i := range users {
				user := users[i]

				var created *time.Time
				if user.TimeCreated != nil {
					created = &user.TimeCreated.Time
				}

				var lastLogin *time.Time
				if user.LastSuccessfulLoginTime != nil {
					lastLogin = &user.LastSuccessfulLoginTime.Time
				}

				var previousLogin *time.Time
				if user.PreviousSuccessfulLoginTime != nil {
					previousLogin = &user.PreviousSuccessfulLoginTime.Time
				}

				capabilities := map[string]interface{}{}
				if user.Capabilities != nil {
					capabilities["canUseConsolePassword"] = boolValue(user.Capabilities.CanUseConsolePassword)
					capabilities["canUseApiKeys"] = boolValue(user.Capabilities.CanUseApiKeys)
					capabilities["canUseAuthTokens"] = boolValue(user.Capabilities.CanUseAuthTokens)
					capabilities["canUseSmtpCredentials"] = boolValue(user.Capabilities.CanUseSmtpCredentials)
					capabilities["canUseCustomerSecretKeys"] = boolValue(user.Capabilities.CanUseCustomerSecretKeys)
					capabilities["canUseOAuth2ClientCredentials"] = boolValue(user.Capabilities.CanUseOAuth2ClientCredentials)
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.identity.user", map[string]*llx.RawData{
					"id":            llx.StringDataPtr(user.Id),
					"name":          llx.StringDataPtr(user.Name),
					"description":   llx.StringDataPtr(user.Description),
					"created":       llx.TimeDataPtr(created),
					"state":         llx.StringData(string(user.LifecycleState)),
					"mfaActivated":  llx.BoolData(boolValue(user.IsMfaActivated)),
					"compartmentID": llx.StringDataPtr(user.CompartmentId),
					"email":         llx.StringDataPtr(user.Email),
					"emailVerified": llx.BoolData(boolValue(user.EmailVerified)),
					"capabilities":  llx.MapData(capabilities, types.Bool),
					"lastLogin":     llx.TimeDataPtr(lastLogin),
					"previousLogin": llx.TimeDataPtr(previousLogin),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (o *mqlOciIdentityUser) id() (string, error) {
	return "oci.identity.user/" + o.Id.Data, nil
}

func (o *mqlOciIdentityUser) apiKeys() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	userId := o.Id.Data

	client, err := conn.IdentityClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := client.ListApiKeys(ctx, identity.ListApiKeysRequest{
		UserId: common.String(userId),
	})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range resp.Items {
		apikey := resp.Items[i]

		var created *time.Time
		if apikey.TimeCreated != nil {
			created = &apikey.TimeCreated.Time
		}

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.identity.apiKey", map[string]*llx.RawData{
			"id":          llx.StringDataPtr(apikey.KeyId),
			"value":       llx.StringDataPtr(apikey.KeyValue),
			"fingerprint": llx.StringDataPtr(apikey.Fingerprint),
			"created":     llx.TimeDataPtr(created),
			"state":       llx.StringData(string(apikey.LifecycleState)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (o *mqlOciIdentityApiKey) id() (string, error) {
	return "oci.identity.apiKey/" + o.Id.Data, nil
}

func (o *mqlOciIdentityUser) customerSecretKeys() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	userId := o.Id.Data

	client, err := conn.IdentityClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := client.ListCustomerSecretKeys(ctx, identity.ListCustomerSecretKeysRequest{
		UserId: common.String(userId),
	})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range resp.Items {
		secretKey := resp.Items[i]

		var created *time.Time
		if secretKey.TimeCreated != nil {
			created = &secretKey.TimeCreated.Time
		}

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.identity.customerSecretKey", map[string]*llx.RawData{
			"id":      llx.StringDataPtr(secretKey.Id),
			"name":    llx.StringDataPtr(secretKey.DisplayName),
			"created": llx.TimeDataPtr(created),
			"state":   llx.StringData(string(secretKey.LifecycleState)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (o *mqlOciIdentityCustomerSecretKey) id() (string, error) {
	return "oci.identity.customerSecretKey/" + o.Id.Data, nil
}

func (o *mqlOciIdentityUser) authTokens() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	userId := o.Id.Data

	client, err := conn.IdentityClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := client.ListAuthTokens(ctx, identity.ListAuthTokensRequest{
		UserId: common.String(userId),
	})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range resp.Items {
		authToken := resp.Items[i]

		var created *time.Time
		if authToken.TimeCreated != nil {
			created = &authToken.TimeCreated.Time
		}
		var expires *time.Time
		if authToken.TimeExpires != nil {
			created = &authToken.TimeExpires.Time
		}

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.identity.authToken", map[string]*llx.RawData{
			"id":          llx.StringDataPtr(authToken.Id),
			"description": llx.StringDataPtr(authToken.Description),
			"created":     llx.TimeDataPtr(created),
			"expires":     llx.TimeDataPtr(expires),
			"state":       llx.StringData(string(authToken.LifecycleState)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (o *mqlOciIdentityAuthToken) id() (string, error) {
	return "oci.identity.authToken/" + o.Id.Data, nil
}

func (o *mqlOciIdentityUser) groups() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	userId := o.Id.Data
	compartmentID := o.CompartmentID.Data

	client, err := conn.IdentityClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	memberships, err := client.ListUserGroupMemberships(ctx, identity.ListUserGroupMembershipsRequest{
		CompartmentId: common.String(compartmentID),
		UserId:        common.String(userId),
	})
	if err != nil {
		return nil, err
	}

	grpMember := map[string]bool{}
	for i := range memberships.Items {
		m := memberships.Items[i]
		if m.GroupId != nil {
			grpMember[*m.GroupId] = true
		}
	}

	// fetch all groups and filter the groups
	obj, err := NewResource(o.MqlRuntime, "oci.identity", nil)
	if err != nil {
		return nil, err
	}
	ociIdentity := obj.(*mqlOciIdentity)
	list := ociIdentity.GetGroups()
	if list.Error != nil {
		return nil, list.Error
	}

	res := []interface{}{}
	for i := range list.Data {
		grp := list.Data[i].(*mqlOciIdentityGroup)
		id := grp.Id.Data
		if err != nil {
			return nil, err
		}
		_, ok := grpMember[id]
		if ok {
			res = append(res, grp)
		}
	}

	return res, nil
}

func (o *mqlOciIdentity) groups() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(o.getGroups(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
	}

	return res, nil
}

func (s *mqlOciIdentity) getGroupsForRegion(ctx context.Context, identityClient *identity.IdentityClient, compartmentID string) ([]identity.Group, error) {
	groups := []identity.Group{}
	var page *string
	for {
		request := identity.ListGroupsRequest{
			CompartmentId: common.String(compartmentID),
			Page:          page,
		}

		response, err := identityClient.ListGroups(ctx, request)
		if err != nil {
			return nil, err
		}

		groups = append(groups, response.Items...)

		if response.OpcNextPage == nil {
			break
		}

		page = response.OpcNextPage
	}

	return groups, nil
}

func (o *mqlOciIdentity) getGroups(conn *connection.OciConnection) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.GetRegions(ctx)
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionVal)

			svc, err := conn.IdentityClientWithRegion(*regionVal.RegionKey)
			if err != nil {
				return nil, err
			}

			var res []interface{}
			groups, err := o.getGroupsForRegion(ctx, svc, conn.TenantID())
			if err != nil {
				return nil, err
			}

			for i := range groups {
				grp := groups[i]

				var created *time.Time
				if grp.TimeCreated != nil {
					created = &grp.TimeCreated.Time
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.identity.group", map[string]*llx.RawData{
					"id":            llx.StringDataPtr(grp.Id),
					"name":          llx.StringDataPtr(grp.Name),
					"description":   llx.StringDataPtr(grp.Description),
					"created":       llx.TimeDataPtr(created),
					"state":         llx.StringData(string(grp.LifecycleState)),
					"compartmentID": llx.StringDataPtr(grp.CompartmentId),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (o *mqlOciIdentityGroup) id() (string, error) {
	return "oci.identity.group/" + o.Id.Data, nil
}

func (o *mqlOciIdentity) policies() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(o.getPolicies(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
	}

	return res, nil
}

func (s *mqlOciIdentity) getPoliciesForRegion(ctx context.Context, identityClient *identity.IdentityClient, compartmentID string) ([]identity.Policy, error) {
	policies := []identity.Policy{}
	var page *string
	for {
		request := identity.ListPoliciesRequest{
			CompartmentId: common.String(compartmentID),
			Page:          page,
		}

		response, err := identityClient.ListPolicies(ctx, request)
		if err != nil {
			return nil, err
		}

		policies = append(policies, response.Items...)

		if response.OpcNextPage == nil {
			break
		}

		page = response.OpcNextPage
	}

	return policies, nil
}

func (o *mqlOciIdentity) getPolicies(conn *connection.OciConnection) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.GetRegions(ctx)
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionVal)

			svc, err := conn.IdentityClientWithRegion(*regionVal.RegionKey)
			if err != nil {
				return nil, err
			}

			var res []interface{}
			policies, err := o.getPoliciesForRegion(ctx, svc, conn.TenantID())
			if err != nil {
				return nil, err
			}

			for i := range policies {
				policy := policies[i]

				var created *time.Time
				if policy.TimeCreated != nil {
					created = &policy.TimeCreated.Time
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.identity.policy", map[string]*llx.RawData{
					"id":            llx.StringDataPtr(policy.Id),
					"name":          llx.StringDataPtr(policy.Name),
					"description":   llx.StringDataPtr(policy.Description),
					"created":       llx.TimeDataPtr(created),
					"state":         llx.StringData(string(policy.LifecycleState)),
					"compartmentID": llx.StringDataPtr(policy.CompartmentId),
					"statements":    llx.ArrayData(convert.SliceAnyToInterface(policy.Statements), types.String),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (o *mqlOciIdentityPolicy) id() (string, error) {
	return "oci.identity.policy/" + o.Id.Data, nil
}
