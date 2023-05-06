package oci

import (
	"context"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/rs/zerolog/log"
	oci_provider "go.mondoo.com/cnquery/motor/providers/oci"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	corePack "go.mondoo.com/cnquery/resources/packs/core"
)

func (o *mqlOciIdentity) id() (string, error) {
	return "oci.identity", nil
}

func (o *mqlOciIdentity) GetUsers() ([]interface{}, error) {
	provider, err := ociProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(o.getUsers(provider), 5)
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

func (s *mqlOciIdentity) getUsers(provider *oci_provider.Provider) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions(ctx)
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionVal)

			svc, err := provider.IdentityClientWithRegion(*regionVal.RegionKey)
			if err != nil {
				return nil, err
			}

			var res []interface{}
			users, err := s.getUsersForRegion(ctx, svc, provider.TenantID())
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
					capabilities["canUseConsolePassword"] = corePack.ToBool(user.Capabilities.CanUseConsolePassword)
					capabilities["canUseApiKeys"] = corePack.ToBool(user.Capabilities.CanUseApiKeys)
					capabilities["canUseAuthTokens"] = corePack.ToBool(user.Capabilities.CanUseAuthTokens)
					capabilities["canUseSmtpCredentials"] = corePack.ToBool(user.Capabilities.CanUseSmtpCredentials)
					capabilities["canUseCustomerSecretKeys"] = corePack.ToBool(user.Capabilities.CanUseCustomerSecretKeys)
					capabilities["canUseOAuth2ClientCredentials"] = corePack.ToBool(user.Capabilities.CanUseOAuth2ClientCredentials)
				}

				mqlInstance, err := s.MotorRuntime.CreateResource("oci.identity.user",
					"id", corePack.ToString(user.Id),
					"name", corePack.ToString(user.Name),
					"description", corePack.ToString(user.Description),
					"created", created,
					"lifecycleState", string(user.LifecycleState),
					"mfaActivated", corePack.ToBool(user.IsMfaActivated),
					"compartmentID", corePack.ToString(user.CompartmentId),
					"email", corePack.ToString(user.Email),
					"emailVerified", corePack.ToBool(user.EmailVerified),
					"capabilities", capabilities,
					"lastLogin", lastLogin,
					"previousLogin", previousLogin,
				)
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
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "oci.identity.user/" + id, nil
}

func (o *mqlOciIdentityUser) GetApiKeys() ([]interface{}, error) {
	provider, err := ociProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	userId, err := o.Id()

	client, err := provider.IdentityClient()
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

		mqlInstance, err := o.MotorRuntime.CreateResource("oci.identity.apiKey",
			"id", corePack.ToString(apikey.KeyId),
			"value", corePack.ToString(apikey.KeyValue),
			"fingerprint", corePack.ToString(apikey.Fingerprint),
			"created", created,
			"lifecycleState", string(apikey.LifecycleState),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (o *mqlOciIdentityApiKey) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "oci.identity.apiKey/" + id, nil
}

func (o *mqlOciIdentityUser) GetCustomerSecretKeys() ([]interface{}, error) {
	provider, err := ociProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	userId, err := o.Id()

	client, err := provider.IdentityClient()
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

		mqlInstance, err := o.MotorRuntime.CreateResource("oci.identity.customerSecretKey",
			"id", corePack.ToString(secretKey.Id),
			"name", corePack.ToString(secretKey.DisplayName),
			"created", created,
			"lifecycleState", string(secretKey.LifecycleState),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (o *mqlOciIdentityCustomerSecretKey) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "oci.identity.customerSecretKey/" + id, nil
}

func (o *mqlOciIdentityUser) GetAuthTokens() ([]interface{}, error) {
	provider, err := ociProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	userId, err := o.Id()

	client, err := provider.IdentityClient()
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

		mqlInstance, err := o.MotorRuntime.CreateResource("oci.identity.authToken",
			"id", corePack.ToString(authToken.Id),
			"description", corePack.ToString(authToken.Description),
			"created", created,
			"expires", expires,
			"lifecycleState", string(authToken.LifecycleState),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (o *mqlOciIdentityAuthToken) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "oci.identity.authToken/" + id, nil
}

func (o *mqlOciIdentityUser) GetGroups() ([]interface{}, error) {
	provider, err := ociProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	userId, err := o.Id()
	if err != nil {
		return nil, err
	}

	compartmentID, err := o.CompartmentID()
	if err != nil {
		return nil, err
	}

	client, err := provider.IdentityClient()
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
	obj, err := o.MotorRuntime.CreateResource("oci.identity")
	if err != nil {
		return nil, err
	}
	ociIdentity := obj.(OciIdentity)
	groups, err := ociIdentity.Groups()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range groups {
		grp := groups[i].(OciIdentityGroup)
		id, err := grp.Id()
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

func (o *mqlOciIdentity) GetGroups() ([]interface{}, error) {
	provider, err := ociProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(o.getGroups(provider), 5)
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

func (s *mqlOciIdentity) getGroups(provider *oci_provider.Provider) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions(ctx)
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionVal)

			svc, err := provider.IdentityClientWithRegion(*regionVal.RegionKey)
			if err != nil {
				return nil, err
			}

			var res []interface{}
			groups, err := s.getGroupsForRegion(ctx, svc, provider.TenantID())
			if err != nil {
				return nil, err
			}

			for i := range groups {
				grp := groups[i]

				var created *time.Time
				if grp.TimeCreated != nil {
					created = &grp.TimeCreated.Time
				}

				mqlInstance, err := s.MotorRuntime.CreateResource("oci.identity.group",
					"id", corePack.ToString(grp.Id),
					"name", corePack.ToString(grp.Name),
					"description", corePack.ToString(grp.Description),
					"created", created,
					"lifecycleState", string(grp.LifecycleState),
					"compartmentID", corePack.ToString(grp.CompartmentId),
				)
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
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "oci.identity.group/" + id, nil
}
