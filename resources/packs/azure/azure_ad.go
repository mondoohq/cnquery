package azure

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/azure"
	"go.mondoo.com/cnquery/resources/packs/core"
)

var azureGraphAudience = azure.PublicCloud.ResourceIdentifiers.Graph

func (a *mqlAzuread) id() (string, error) {
	return "azuread", nil
}

func (a *mqlAzuread) GetUsers() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	authorizer, err := at.AuthorizerWithAudience(azureGraphAudience)
	if err != nil {
		return nil, err
	}

	usersClient := graphrbac.NewUsersClient(at.TenantID())
	usersClient.Authorizer = authorizer

	ctx := context.Background()
	userList, err := usersClient.List(ctx, "", "")
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range userList.Values() {
		usr := userList.Values()[i]

		properties := make(map[string](interface{}))

		data, err := json.Marshal(usr.AdditionalProperties)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(data), &properties)
		if err != nil {
			return nil, err
		}

		mqlAzureAdUser, err := a.MotorRuntime.CreateResource("azuread.user",
			"id", core.ToString(usr.ObjectID),
			"displayName", core.ToString(usr.DisplayName),
			"givenName", core.ToString(usr.GivenName),
			"surname", core.ToString(usr.Surname),
			"userPrincipalName", core.ToString(usr.UserPrincipalName),
			"accountEnabled", core.ToBool(usr.AccountEnabled),
			"mailNickname", core.ToString(usr.MailNickname),
			"mail", core.ToString(usr.Mail),
			"objectType", string(usr.ObjectType),
			"userType", string(usr.UserType),
			"properties", properties,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAzureAdUser)
	}

	return res, nil
}

func (a *mqlAzuread) GetGroups() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	authorizer, err := at.AuthorizerWithAudience(azureGraphAudience)
	if err != nil {
		return nil, err
	}

	groupsClient := graphrbac.NewGroupsClient(at.TenantID())
	groupsClient.Authorizer = authorizer

	ctx := context.Background()
	grpList, err := groupsClient.List(ctx, "")
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range grpList.Values() {
		grp := grpList.Values()[i]

		properties := make(map[string](interface{}))

		data, err := json.Marshal(grp.AdditionalProperties)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(data), &properties)
		if err != nil {
			return nil, err
		}

		mqlAzureAdGroup, err := a.MotorRuntime.CreateResource("azuread.group",
			"id", core.ToString(grp.ObjectID),
			"displayName", core.ToString(grp.DisplayName),
			"securityEnabled", core.ToBool(grp.SecurityEnabled),
			"mailEnabled", core.ToBool(grp.MailEnabled),
			"mailNickname", core.ToString(grp.MailNickname),
			"mail", core.ToString(grp.Mail),
			"mailNickname", core.ToString(grp.MailNickname),
			"mail", core.ToString(grp.Mail),
			"objectType", string(grp.ObjectType),
			"properties", properties,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAzureAdGroup)
	}

	return res, nil
}

func (a *mqlAzuread) GetDomains() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	authorizer, err := at.AuthorizerWithAudience(azureGraphAudience)
	if err != nil {
		return nil, err
	}

	domainClient := graphrbac.NewDomainsClient(at.TenantID())
	domainClient.Authorizer = authorizer

	ctx := context.Background()
	domainList, err := domainClient.List(ctx, "")
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	if domainList.Value == nil {
		return res, nil
	}

	list := *domainList.Value
	for i := range list {
		domain := list[i]

		properties := make(map[string](interface{}))

		data, err := json.Marshal(domain.AdditionalProperties)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(data), &properties)
		if err != nil {
			return nil, err
		}

		mqlAzureAdDomain, err := a.MotorRuntime.CreateResource("azuread.domain",
			"name", core.ToString(domain.Name),
			"isVerified", core.ToBool(domain.IsVerified),
			"isDefault", core.ToBool(domain.IsDefault),
			"authenticationType", core.ToString(domain.AuthenticationType),
			"properties", properties,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAzureAdDomain)
	}

	return res, nil
}

func (a *mqlAzuread) GetApplications() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *mqlAzuread) GetServicePrincipals() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *mqlAzureadUser) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureadGroup) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureadGroup) GetMembers() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *mqlAzureadDomain) id() (string, error) {
	return a.Name()
}

func (a *mqlAzureadApplication) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureadServiceprincipal) id() (string, error) {
	return a.Id()
}
