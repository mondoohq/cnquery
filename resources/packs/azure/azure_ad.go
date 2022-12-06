package azure

import (
	"github.com/cockroachdb/errors"
	"github.com/microsoft/kiota-abstractions-go/authentication"
	a "github.com/microsoft/kiota-authentication-azure-go"
	microsoft_provider "go.mondoo.com/cnquery/motor/providers/microsoft"
	"go.mondoo.com/cnquery/motor/providers/microsoft/msgraph/msgraphclient"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func graphBetaClient(t *microsoft_provider.Provider) (*msgraphclient.GraphServiceClient, error) {
	auth, err := t.GetTokenCredential()
	if err != nil {
		return nil, err
	}
	providerFunc := func() (authentication.AuthenticationProvider, error) {
		return a.NewAzureIdentityAuthenticationProviderWithScopes(auth, msgraphclient.DefaultMSGraphScopes)
	}
	adapter, err := msgraphclient.NewGraphRequestAdapterWithFn(providerFunc)
	if err != nil {
		return nil, err
	}
	graphBetaClient := msgraphclient.NewGraphServiceClient(adapter)
	return graphBetaClient, nil
}

func (a *mqlAzuread) id() (string, error) {
	return "azuread", nil
}

func (a *mqlAzuread) GetUsers() ([]interface{}, error) {
	at, err := msGraphTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	usersClient, err := graphBetaClient(at)
	if err != nil {
		return nil, err
	}
	userList, err := usersClient.Users().Get()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for _, usr := range userList.GetValue() {
		mqlAzureAdUser, err := a.MotorRuntime.CreateResource("azuread.user",
			"id", core.ToString(usr.GetId()),
			"displayName", core.ToString(usr.GetDisplayName()),
			"givenName", core.ToString(usr.GetGivenName()),
			"surname", core.ToString(usr.GetSurname()),
			"userPrincipalName", core.ToString(usr.GetUserPrincipalName()),
			"accountEnabled", core.ToBool(usr.GetAccountEnabled()),
			"mailNickname", core.ToString(usr.GetMailNickname()),
			"mail", core.ToString(usr.GetMail()),
			"userType", core.ToString(usr.GetUserType()),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAzureAdUser)
	}

	return res, nil
}

func (a *mqlAzuread) GetGroups() ([]interface{}, error) {
	at, err := msGraphTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	groupsClient, err := graphBetaClient(at)
	if err != nil {
		return nil, err
	}
	groupsList, err := groupsClient.Groups().Get()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for _, grp := range groupsList.GetValue() {
		mqlAzureAdGroup, err := a.MotorRuntime.CreateResource("azuread.group",
			"id", core.ToString(grp.GetId()),
			"displayName", core.ToString(grp.GetDisplayName()),
			"securityEnabled", core.ToBool(grp.GetSecurityEnabled()),
			"mailEnabled", core.ToBool(grp.GetMailEnabled()),
			"mailNickname", core.ToString(grp.GetMailNickname()),
			"mail", core.ToString(grp.GetMail()),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAzureAdGroup)
	}

	return res, nil
}

func (a *mqlAzuread) GetDomains() ([]interface{}, error) {
	at, err := msGraphTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := graphBetaClient(at)
	if err != nil {
		return nil, err
	}
	domains, err := client.Domains().Get()
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for _, domain := range domains.GetValue() {
		mqlAzureAdDomain, err := a.MotorRuntime.CreateResource("azuread.domain",
			"name", core.ToString(domain.GetId()),
			"isVerified", core.ToBool(domain.GetIsVerified()),
			"isDefault", core.ToBool(domain.GetIsDefault()),
			"authenticationType", core.ToString(domain.GetAuthenticationType()),
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

func (a *mqlAzureadUser) GetObjectType() (interface{}, error) {
	return nil, errors.New("object type no longer supported")
}

func (a *mqlAzureadUser) GetProperties() ([]interface{}, error) {
	return nil, errors.New("properties no longer supported")
}

func (a *mqlAzureadGroup) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureadGroup) GetMembers() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *mqlAzureadGroup) GetObjectType() (interface{}, error) {
	return nil, errors.New("object type no longer supported")
}

func (a *mqlAzureadGroup) GetProperties() ([]interface{}, error) {
	return nil, errors.New("properties no longer supported")
}

func (a *mqlAzureadDomain) id() (string, error) {
	return a.Name()
}

func (a *mqlAzureadDomain) GetProperties() ([]interface{}, error) {
	return nil, errors.New("properties no longer supported")
}

func (a *mqlAzureadApplication) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureadServiceprincipal) id() (string, error) {
	return a.Id()
}
