package resources

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/azure"
)

var azureGraphAudience = azure.PublicCloud.ResourceIdentifiers.Graph

func (a *lumiAzuread) id() (string, error) {
	return "azuread", nil
}

func (a *lumiAzuread) GetUsers() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Transport)
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

		lumiAzureAdUser, err := a.MotorRuntime.CreateResource("azuread.user",
			"id", toString(usr.ObjectID),
			"displayName", toString(usr.DisplayName),
			"givenName", toString(usr.GivenName),
			"surname", toString(usr.Surname),
			"userPrincipalName", toString(usr.UserPrincipalName),
			"accountEnabled", toBool(usr.AccountEnabled),
			"mailNickname", toString(usr.MailNickname),
			"mail", toString(usr.Mail),
			"objectType", string(usr.ObjectType),
			"userType", string(usr.UserType),
			"properties", properties,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzureAdUser)
	}

	return res, nil
}

func (a *lumiAzuread) GetGroups() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Transport)
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

		lumiAzureAdGroup, err := a.MotorRuntime.CreateResource("azuread.group",
			"id", toString(grp.ObjectID),
			"displayName", toString(grp.DisplayName),
			"securityEnabled", toBool(grp.SecurityEnabled),
			"mailEnabled", toBool(grp.MailEnabled),
			"mailNickname", toString(grp.MailNickname),
			"mail", toString(grp.Mail),
			"mailNickname", toString(grp.MailNickname),
			"mail", toString(grp.Mail),
			"objectType", string(grp.ObjectType),
			"properties", properties,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzureAdGroup)
	}

	return res, nil
}

func (a *lumiAzuread) GetDomains() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Transport)
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

		lumiAzureAdDomain, err := a.MotorRuntime.CreateResource("azuread.domain",
			"name", toString(domain.Name),
			"isVerified", toBool(domain.IsVerified),
			"isDefault", toBool(domain.IsDefault),
			"authenticationType", toString(domain.AuthenticationType),
			"properties", properties,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzureAdDomain)
	}

	return res, nil
}

func (a *lumiAzuread) GetApplications() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *lumiAzuread) GetServicePrincipals() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *lumiAzureadUser) id() (string, error) {
	return a.Id()
}

func (a *lumiAzureadGroup) id() (string, error) {
	return a.Id()
}

func (a *lumiAzureadGroup) GetMembers() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *lumiAzureadDomain) id() (string, error) {
	return a.Name()
}

func (a *lumiAzureadApplication) id() (string, error) {
	return a.Id()
}

func (a *lumiAzureadServiceprincipal) id() (string, error) {
	return a.Id()
}
