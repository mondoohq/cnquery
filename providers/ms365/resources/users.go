// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"time"

	betamodels "github.com/microsoftgraph/msgraph-beta-sdk-go/models"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/reports"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/ms365/connection"
	"go.mondoo.com/cnquery/v11/types"
)

var userSelectFields = []string{
	"id", "accountEnabled", "city", "companyName", "country", "createdDateTime", "department", "displayName", "employeeId", "givenName",
	"jobTitle", "mail", "mobilePhone", "otherMails", "officeLocation", "postalCode", "state", "streetAddress", "surname", "userPrincipalName", "userType",
}

// users reads all users from Entra ID
// Permissions: User.Read.All, Directory.Read.All
// see https://learn.microsoft.com/en-us/graph/api/user-list?view=graph-rest-1.0&tabs=http
func (a *mqlMicrosoft) users() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	betaClient, err := conn.BetaGraphClient()
	if err != nil {
		return nil, err
	}
	// fetch user data
	ctx := context.Background()
	top := int32(999)
	resp, err := graphClient.Users().Get(
		ctx, &users.UsersRequestBuilderGetRequestConfiguration{
			QueryParameters: &users.UsersRequestBuilderGetQueryParameters{
				Select: userSelectFields,
				Top:    &top,
			},
		},
	)
	if err != nil {
		return nil, transformError(err)
	}
	users, err := iterate[*models.User](ctx, resp, graphClient.GetAdapter(), users.CreateDeltaGetResponseFromDiscriminatorValue)
	if err != nil {
		return nil, transformError(err)
	}

	detailsResp, err := betaClient.Reports().AuthenticationMethods().UserRegistrationDetails().Get(
		ctx,
		&reports.AuthenticationMethodsUserRegistrationDetailsRequestBuilderGetRequestConfiguration{
			QueryParameters: &reports.AuthenticationMethodsUserRegistrationDetailsRequestBuilderGetQueryParameters{
				Top: &top,
			},
		})
	// we do not want to fail the user fetching here, this likely means the tenant does not have the right license
	if err != nil {
		a.mfaResp = mfaResp{err: err}
	} else {
		userRegistrationDetails, err := iterate[*betamodels.UserRegistrationDetails](ctx, detailsResp, betaClient.GetAdapter(), betamodels.CreateUserRegistrationDetailsCollectionResponseFromDiscriminatorValue)
		// we do not want to fail the user fetching here, this likely means the tenant does not have the right license
		if err != nil {
			a.mfaResp = mfaResp{err: err}
		} else {
			mfaMap := map[string]bool{}
			for _, u := range userRegistrationDetails {
				if u.GetId() == nil || u.GetIsMfaRegistered() == nil {
					continue
				}
				mfaMap[*u.GetId()] = *u.GetIsMfaRegistered()
			}
			a.mfaResp = mfaResp{mfaMap: mfaMap}
		}
	}

	// construct the result
	res := []interface{}{}
	for _, u := range users {
		graphUser, err := newMqlMicrosoftUser(a.MqlRuntime, u)
		if err != nil {
			return nil, err
		}
		// index users by id and principal name
		a.index(graphUser)
		res = append(res, graphUser)
	}

	return res, nil
}

func initMicrosoftUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// we only look up the user if we have been supplied by id, displayName or userPrincipalName
	if len(args) > 1 {
		return args, nil, nil
	}

	rawId, okId := args["id"]
	rawDisplayName, okDisplayName := args["displayName"]
	rawPrincipalName, okPrincipalName := args["userPrincipalName"]

	if !okId && !okDisplayName && !okPrincipalName {
		return args, nil, nil
	}

	var filter *string
	if okId {
		idFilter := fmt.Sprintf("id eq '%s'", rawId.Value.(string))
		filter = &idFilter
	} else if okPrincipalName {
		principalNameFilter := fmt.Sprintf("userPrincipalName eq '%s'", rawPrincipalName.Value.(string))
		filter = &principalNameFilter
	} else if okDisplayName {
		displayNameFilter := fmt.Sprintf("displayName eq '%s'", rawDisplayName.Value.(string))
		filter = &displayNameFilter
	}
	if filter == nil {
		return nil, nil, errors.New("no filter found")
	}

	conn := runtime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.Users().Get(ctx, &users.UsersRequestBuilderGetRequestConfiguration{
		QueryParameters: &users.UsersRequestBuilderGetQueryParameters{
			Filter: filter,
		},
	})
	if err != nil {
		return nil, nil, transformError(err)
	}

	val := resp.GetValue()
	if len(val) == 0 {
		return nil, nil, errors.New("user not found")
	}

	userId := val[0].GetId()
	if userId == nil {
		return nil, nil, errors.New("user id not found")
	}

	// fetch user by id
	user, err := graphClient.Users().ByUserId(*userId).Get(ctx, &users.UserItemRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, nil, transformError(err)
	}
	mqlMsApp, err := newMqlMicrosoftUser(runtime, user)
	if err != nil {
		return nil, nil, err
	}

	return nil, mqlMsApp, nil
}

func newMqlMicrosoftUser(runtime *plugin.Runtime, u models.Userable) (*mqlMicrosoftUser, error) {
	graphUser, err := CreateResource(runtime, "microsoft.user",
		map[string]*llx.RawData{
			"__id":              llx.StringDataPtr(u.GetId()),
			"id":                llx.StringDataPtr(u.GetId()),
			"accountEnabled":    llx.BoolDataPtr(u.GetAccountEnabled()),
			"city":              llx.StringDataPtr(u.GetCity()),        // deprecated
			"companyName":       llx.StringDataPtr(u.GetCompanyName()), // deprecated
			"country":           llx.StringDataPtr(u.GetCountry()),     // deprecated
			"createdDateTime":   llx.TimeDataPtr(u.GetCreatedDateTime()),
			"department":        llx.StringDataPtr(u.GetDepartment()),
			"displayName":       llx.StringDataPtr(u.GetDisplayName()),
			"employeeId":        llx.StringDataPtr(u.GetEmployeeId()), // deprecated
			"givenName":         llx.StringDataPtr(u.GetGivenName()),
			"jobTitle":          llx.StringDataPtr(u.GetJobTitle()), // deprecated
			"mail":              llx.StringDataPtr(u.GetMail()),
			"mobilePhone":       llx.StringDataPtr(u.GetMobilePhone()),                        // deprecated
			"otherMails":        llx.ArrayData(llx.TArr2Raw(u.GetOtherMails()), types.String), // deprecated
			"officeLocation":    llx.StringDataPtr(u.GetOfficeLocation()),                     // deprecated
			"postalCode":        llx.StringDataPtr(u.GetPostalCode()),                         // deprecated
			"state":             llx.StringDataPtr(u.GetState()),                              // deprecated
			"streetAddress":     llx.StringDataPtr(u.GetStreetAddress()),                      // deprecated
			"surname":           llx.StringDataPtr(u.GetSurname()),
			"userPrincipalName": llx.StringDataPtr(u.GetUserPrincipalName()),
			"userType":          llx.StringDataPtr(u.GetUserType()),
		})
	if err != nil {
		return nil, err
	}
	return graphUser.(*mqlMicrosoftUser), nil
}

// https://learn.microsoft.com/en-us/graph/api/resources/user?view=graph-rest-1.0#properties
var userJobContactFields = []string{
	"jobTitle", "companyName", "department", "employeeId", "employeeType", "employeeHireDate",
	"officeLocation", "streetAddress", "city", "state", "postalCode", "country", "businessPhones", "mobilePhone", "mail", "otherMails", "faxNumber", "mailNickname",
}

func (a *mqlMicrosoftUser) mfaEnabled() (bool, error) {
	mql, err := CreateResource(a.MqlRuntime, "microsoft", map[string]*llx.RawData{})
	if err != nil {
		return false, err
	}

	microsoft := mql.(*mqlMicrosoft)
	if microsoft.mfaResp.mfaMap == nil {
		microsoft.mfaResp.mfaMap = make(map[string]bool)
	}
	if microsoft.mfaResp.err != nil {
		a.MfaEnabled.Error = microsoft.mfaResp.err
		a.MfaEnabled.State = plugin.StateIsSet
		return false, a.MfaEnabled.Error
	}

	a.MfaEnabled.Data = microsoft.mfaResp.mfaMap[a.Id.Data]
	a.MfaEnabled.State = plugin.StateIsSet
	return a.MfaEnabled.Data, nil
}

func (a *mqlMicrosoftUser) populateJobContactData() error {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return err
	}

	userID := a.Id.Data
	ctx := context.Background()
	userData, err := graphClient.Users().ByUserId(userID).Get(ctx, &users.UserItemRequestBuilderGetRequestConfiguration{
		QueryParameters: &users.UserItemRequestBuilderGetQueryParameters{
			Select: userJobContactFields,
		},
	})
	if err != nil {
		return transformError(err)
	}

	jobDesc, _ := convert.JsonToDict(userJob{
		JobTitle:    userData.GetJobTitle(),
		CompanyName: userData.GetCompanyName(),
		Department:  userData.GetDepartment(),
		EmployeeId:  userData.GetEmployeeId(),
		// EmployeeType:     userData.GetEmployeeType(),
		// EmployeeHireDate: userData.GetEmployeeHireDate(),
		OfficeLocation: userData.GetOfficeLocation(),
	})
	a.Job = plugin.TValue[interface{}]{Data: jobDesc, State: plugin.StateIsSet}

	userContact, _ := convert.JsonToDict(userContact{
		StreetAddress:  userData.GetStreetAddress(),
		City:           userData.GetCity(),
		State:          userData.GetState(),
		PostalCode:     userData.GetPostalCode(),
		Country:        userData.GetCountry(),
		BusinessPhones: userData.GetBusinessPhones(),
		MobilePhone:    userData.GetMobilePhone(),
		Email:          userData.GetMail(),
		OtherMails:     userData.GetOtherMails(),
		FaxNumber:      userData.GetFaxNumber(),
		MailNickname:   userData.GetMailNickname(),
	})
	a.Contact = plugin.TValue[interface{}]{Data: userContact, State: plugin.StateIsSet}

	return nil
}

type userJob struct {
	CompanyName      *string    `json:"companyName"`
	JobTitle         *string    `json:"jobTitle"`
	Department       *string    `json:"department"`
	EmployeeId       *string    `json:"employeeId"`
	EmployeeType     *string    `json:"employeeType"`
	EmployeeHireDate *time.Time `json:"employeeHireDate"`
	OfficeLocation   *string    `json:"officeLocation"`
}

type userContact struct {
	StreetAddress  *string  `json:"streetAddress"`
	City           *string  `json:"city"`
	State          *string  `json:"state"`
	PostalCode     *string  `json:"postalCode"`
	Country        *string  `json:"country"`
	BusinessPhones []string `json:"BusinessPhones"`
	MobilePhone    *string  `json:"mobilePhone"`
	Email          *string  `json:"email"`
	OtherMails     []string `json:"otherMails"`
	FaxNumber      *string  `json:"faxNumber"`
	MailNickname   *string  `json:"mailNickname"`
}

func (a *mqlMicrosoftUser) job() (interface{}, error) {
	return nil, a.populateJobContactData()
}

func (a *mqlMicrosoftUser) contact() (interface{}, error) {
	return nil, a.populateJobContactData()
}

func (a *mqlMicrosoftUser) settings() (interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	id := a.Id.Data
	userSettings, err := graphClient.Users().ByUserId(id).Settings().Get(ctx, &users.ItemSettingsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}

	return convert.JsonToDict(newUserSettings(userSettings))
}

type authMethod struct {
	Id string `json:"id"`
}

type phoneMethod struct {
	authMethod
	Type           string  `json:"type"`
	PhoneNumber    *string `json:"phoneNumber"`
	SsmSignInState string  `json:"ssmSignInState"`
}

type fido2Method struct {
	authMethod
	Name                    *string  `json:"name"`
	AttestationLevel        string   `json:"attestationLevel"`
	Model                   *string  `json:"model"`
	AttestationCertificates []string `json:"attestationCertificates"`
}

type emailMethod struct {
	authMethod
	EmailAddress *string `json:"emailAddress"`
}

type windowsHelloMethod struct {
	authMethod
	Name        *string `json:"name"`
	DeviceId    *string `json:"deviceId"`
	KeyStrength string  `json:"keyStrength"`
}

type softwareMethod struct {
	authMethod
}

type passwordMethod struct {
	authMethod
}

type microsoftAuthenticatorMethod struct {
	authMethod
	Name            *string `json:"name"`
	PhoneAppVersion *string `json:"phoneAppVersion"`
	DeviceTag       *string `json:"deviceTag"`
}

type temporaryAccessPassMethod struct {
	authMethod
	IsUsable          *bool  `json:"isUsable"`
	IsUsableOnce      *bool  `json:"isUsableOnce"`
	LifetimeInMinutes *int32 `json:"lifetimeInMinutes"`
}

type userAuthentication struct {
	userID                     string                         `json:"userId"`
	methodCount                int                            `json:"methodCount"`
	PhoneMethods               []phoneMethod                  `json:"phoneMethods"`
	Fido2Methods               []fido2Method                  `json:"fido2Methods"`
	SoftwareMethods            []softwareMethod               `json:"softwareMethods"`
	MicrosoftAuthenticator     []microsoftAuthenticatorMethod `json:"microsoftAuthenticator"`
	PasswordMethods            []passwordMethod               `json:"passwordMethods"`
	TemporaryAccessPassMethods []temporaryAccessPassMethod    `json:"temporaryAccessPassMethods"`
	WindowsHelloMethods        []windowsHelloMethod           `json:"windowsHelloMethods"`
	EmailMethods               []emailMethod                  `json:"emailMethods"`
}

// needs the permission UserAuthenticationMethod.Read.All
func (a *mqlMicrosoftUser) authMethods() (*mqlMicrosoftUserAuthenticationMethods, error) {
	runtime := a.MqlRuntime
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	userID := a.Id.Data
	ua := userAuthentication{
		userID: userID,
	}

	ctx := context.Background()

	authMethods, err := graphClient.Users().ByUserId(userID).Authentication().Methods().Get(ctx, &users.ItemAuthenticationMethodsRequestBuilderGetRequestConfiguration{})
	if oErr, ok := isOdataError(err); ok {
		if oErr.ResponseStatusCode == 403 {
			return nil, errors.New("UserAuthenticationMethod.Read.All permission is required")
		}
		return nil, transformError(err)
	} else if err != nil {
		return nil, transformError(err)
	}

	methods := authMethods.GetValue()
	ua.methodCount = len(methods)
	for i := range methods {
		entry := methods[i]
		switch x := entry.(type) {
		case *models.PhoneAuthenticationMethod:
			if x.GetId() == nil {
				continue
			}

			m := phoneMethod{
				authMethod: authMethod{
					Id: *x.GetId(),
				},
				PhoneNumber: x.GetPhoneNumber(),
			}

			if x.GetPhoneType() != nil {
				m.Type = x.GetPhoneType().String()
			}

			if x.GetSmsSignInState() != nil {
				m.SsmSignInState = x.GetSmsSignInState().String()
			}

			ua.PhoneMethods = append(ua.PhoneMethods, m)
		case *models.Fido2AuthenticationMethod:
			if x.GetId() == nil {
				continue
			}
			m := fido2Method{
				authMethod: authMethod{
					Id: *x.GetId(),
				},
				Name:                    x.GetDisplayName(),
				Model:                   x.GetModel(),
				AttestationCertificates: x.GetAttestationCertificates(),
			}
			if x.GetAttestationLevel() != nil {
				m.AttestationLevel = x.GetAttestationLevel().String()
			}
			ua.Fido2Methods = append(ua.Fido2Methods, m)
		case *models.SoftwareOathAuthenticationMethod:
			if x.GetId() == nil {
				continue
			}
			m := softwareMethod{
				authMethod: authMethod{
					Id: *x.GetId(),
				},
			}

			ua.SoftwareMethods = append(ua.SoftwareMethods, m)
		case *models.MicrosoftAuthenticatorAuthenticationMethod:
			if x.GetId() == nil {
				continue
			}
			m := microsoftAuthenticatorMethod{
				authMethod: authMethod{
					Id: *x.GetId(),
				},
				Name:            x.GetDisplayName(),
				PhoneAppVersion: x.GetPhoneAppVersion(),
				DeviceTag:       x.GetDeviceTag(),
			}

			ua.MicrosoftAuthenticator = append(ua.MicrosoftAuthenticator, m)
		case *models.PasswordAuthenticationMethod:
			if x.GetId() == nil {
				continue
			}
			m := passwordMethod{
				authMethod: authMethod{
					Id: *x.GetId(),
				},
			}

			ua.PasswordMethods = append(ua.PasswordMethods, m)
		case *models.TemporaryAccessPassAuthenticationMethod:
			if x.GetId() == nil {
				continue
			}
			m := temporaryAccessPassMethod{
				authMethod: authMethod{
					Id: *x.GetId(),
				},
				IsUsable:          x.GetIsUsable(),
				IsUsableOnce:      x.GetIsUsableOnce(),
				LifetimeInMinutes: x.GetLifetimeInMinutes(),
			}
			ua.TemporaryAccessPassMethods = append(ua.TemporaryAccessPassMethods, m)
		case *models.WindowsHelloForBusinessAuthenticationMethod:
			if x.GetId() == nil {
				continue
			}
			m := windowsHelloMethod{
				authMethod: authMethod{
					Id: *x.GetId(),
				},
				Name: x.GetDisplayName(),
			}
			if x.GetDevice() != nil {
				m.DeviceId = x.GetDevice().GetDeviceId()
			}

			if x.GetKeyStrength() != nil {
				m.KeyStrength = x.GetKeyStrength().String()
			}

			ua.WindowsHelloMethods = append(ua.WindowsHelloMethods, m)
		case *models.EmailAuthenticationMethod:
			if x.GetId() == nil {
				continue
			}

			m := emailMethod{
				authMethod: authMethod{
					Id: *x.GetId(),
				},
				EmailAddress: x.GetEmailAddress(),
			}
			ua.EmailMethods = append(ua.EmailMethods, m)
		default:

		}
	}

	return newMqlMicrosoftUserAuthentication(runtime, ua)
}

func newMqlMicrosoftUserAuthentication(runtime *plugin.Runtime, u userAuthentication) (*mqlMicrosoftUserAuthenticationMethods, error) {
	if u.userID == "" {
		return nil, errors.New("user id is required")
	}
	phoneMethods, _ := convert.JsonToDictSlice(u.PhoneMethods)
	emailMethods, _ := convert.JsonToDictSlice(u.EmailMethods)
	fido2Methods, _ := convert.JsonToDictSlice(u.Fido2Methods)
	softwareMethods, _ := convert.JsonToDictSlice(u.SoftwareMethods)
	microsoftAuthenticator, _ := convert.JsonToDictSlice(u.MicrosoftAuthenticator)
	passwordMethods, _ := convert.JsonToDictSlice(u.PasswordMethods)
	temporaryAccessPassMethods, _ := convert.JsonToDictSlice(u.TemporaryAccessPassMethods)
	windowsHelloMethods, _ := convert.JsonToDictSlice(u.WindowsHelloMethods)

	graphUser, err := CreateResource(runtime, "microsoft.user.authenticationMethods",
		map[string]*llx.RawData{
			"__id":                       llx.StringData(u.userID),
			"count":                      llx.IntData(u.methodCount),
			"phoneMethods":               llx.DictData(phoneMethods),
			"emailMethods":               llx.DictData(emailMethods),
			"fido2Methods":               llx.DictData(fido2Methods),
			"softwareMethods":            llx.DictData(softwareMethods),
			"microsoftAuthenticator":     llx.DictData(microsoftAuthenticator),
			"passwordMethods":            llx.DictData(passwordMethods),
			"temporaryAccessPassMethods": llx.DictData(temporaryAccessPassMethods),
			"windowsHelloMethods":        llx.DictData(windowsHelloMethods),
		})
	if err != nil {
		return nil, err
	}
	return graphUser.(*mqlMicrosoftUserAuthenticationMethods), nil
}
