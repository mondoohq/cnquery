// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"time"

	abstractions "github.com/microsoft/kiota-abstractions-go"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/auditlogs"
	betamodels "github.com/microsoftgraph/msgraph-beta-sdk-go/models"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/reports"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers/ms365/connection"
	"go.mondoo.com/cnquery/v12/types"
)

var userSelectFields = []string{
	"id", "accountEnabled", "city", "companyName", "country", "createdDateTime",
	"department", "displayName", "employeeId", "givenName", "jobTitle", "mail",
	"mobilePhone", "otherMails", "officeLocation", "postalCode", "state", "identities",
	"streetAddress", "surname", "userPrincipalName", "userType", "creationType",
	"assignedLicenses",
}

func (a *mqlMicrosoft) users() (*mqlMicrosoftUsers, error) {
	resource, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "microsoft.users", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}

	return resource.(*mqlMicrosoftUsers), nil
}

func initMicrosoftUsers(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	args["__id"] = newListResourceIdFromArguments("microsoft.users", args)
	resource, err := runtime.CreateResource(runtime, "microsoft.users", args)
	if err != nil {
		return args, nil, err
	}

	return args, resource.(*mqlMicrosoftUsers), nil
}

// list fetches users from Entra ID and allows the user provide a filter to retrieve
// a subset of users
//
// Permissions: User.Read.All, Directory.Read.All
// see https://learn.microsoft.com/en-us/graph/api/user-list?view=graph-rest-1.0&tabs=http
func (a *mqlMicrosoftUsers) list() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	betaClient, err := conn.BetaGraphClient()
	if err != nil {
		return nil, err
	}

	// Index of users are stored inside the top level resource `microsoft`, just like
	// MFA response. Here we create or get the resource to access those internals
	mainResource, err := CreateResource(a.MqlRuntime, "microsoft", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	microsoft := mainResource.(*mqlMicrosoft)

	// fetch user data
	ctx := context.Background()
	top := int32(999)
	opts := &users.UsersRequestBuilderGetRequestConfiguration{
		QueryParameters: &users.UsersRequestBuilderGetQueryParameters{
			Select: userSelectFields,
			Top:    &top,
		},
	}

	if a.Search.State == plugin.StateIsSet || a.Filter.State == plugin.StateIsSet {
		// search and filter requires this header
		headers := abstractions.NewRequestHeaders()
		headers.Add("ConsistencyLevel", "eventual")
		opts.Headers = headers

		if a.Search.State == plugin.StateIsSet {
			log.Debug().
				Str("search", a.Search.Data).
				Msg("microsoft.users.list.search set")
			search, err := parseSearch(a.Search.Data)
			if err != nil {
				return nil, err
			}
			opts.QueryParameters.Search = &search
		}
		if a.Filter.State == plugin.StateIsSet {
			log.Debug().
				Str("filter", a.Filter.Data).
				Msg("microsoft.users.list.filter set")
			opts.QueryParameters.Filter = &a.Filter.Data
			count := true
			opts.QueryParameters.Count = &count
		}
	}

	resp, err := graphClient.Users().Get(ctx, opts)
	if err != nil {
		return nil, transformError(err)
	}
	users, err := iterate[*models.User](ctx,
		resp,
		graphClient.GetAdapter(),
		users.CreateDeltaGetResponseFromDiscriminatorValue,
	)
	if err != nil {
		return nil, transformError(err)
	}

	detailsResp, err := betaClient.
		Reports().
		AuthenticationMethods().
		UserRegistrationDetails().
		Get(ctx,
			&reports.AuthenticationMethodsUserRegistrationDetailsRequestBuilderGetRequestConfiguration{
				QueryParameters: &reports.AuthenticationMethodsUserRegistrationDetailsRequestBuilderGetQueryParameters{
					Top: &top,
				},
			})
	// we do not want to fail the user fetching here, this likely means the tenant does not have the right license
	if err != nil {
		microsoft.mfaResp = mfaResp{err: err}
	} else {
		userRegistrationDetails, err := iterate[*betamodels.UserRegistrationDetails](ctx, detailsResp, betaClient.GetAdapter(), betamodels.CreateUserRegistrationDetailsCollectionResponseFromDiscriminatorValue)
		// we do not want to fail the user fetching here, this likely means the tenant does not have the right license
		if err != nil {
			microsoft.mfaResp = mfaResp{err: err}
		} else {
			mfaMap := map[string]bool{}
			for _, u := range userRegistrationDetails {
				if u.GetId() == nil || u.GetIsMfaRegistered() == nil {
					continue
				}
				mfaMap[*u.GetId()] = *u.GetIsMfaRegistered()
			}
			microsoft.mfaResp = mfaResp{mfaMap: mfaMap}
		}
	}

	// construct the result
	res := []any{}
	for _, u := range users {
		graphUser, err := newMqlMicrosoftUser(a.MqlRuntime, u)
		if err != nil {
			return nil, err
		}
		// index users by id and principal name
		microsoft.indexUser(graphUser)
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
	identities := []any{}
	for idx, userId := range u.GetIdentities() {
		id := fmt.Sprintf("%s-%d", *u.GetId(), idx)
		identity, err := CreateResource(runtime, "microsoft.user.identity", map[string]*llx.RawData{
			"signInType":       llx.StringDataPtr(userId.GetSignInType()),
			"issuer":           llx.StringDataPtr(userId.GetIssuer()),
			"issuerAssignedId": llx.StringDataPtr(userId.GetIssuerAssignedId()),
			"__id":             llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		identities = append(identities, identity)
	}

	mqlAssignedLicensesList := []any{}

	if u.GetAssignedLicenses() != nil {
		for _, license := range u.GetAssignedLicenses() {
			if license == nil {
				continue
			}

			var disabledPlanStrings []string
			if license.GetDisabledPlans() != nil {
				for _, planUUID := range license.GetDisabledPlans() {
					disabledPlanStrings = append(disabledPlanStrings, planUUID.String())
				}
			}

			mqlAssignedLicenses, err := CreateResource(runtime, "microsoft.user.assignedLicense",
				map[string]*llx.RawData{
					"__id":          llx.StringData(license.GetSkuId().String()),
					"disabledPlans": llx.ArrayData(convert.SliceAnyToInterface(disabledPlanStrings), types.String),
					"skuId":         llx.StringData(license.GetSkuId().String()),
				})
			if err != nil {
				return nil, err
			}
			mqlAssignedLicensesList = append(mqlAssignedLicensesList, mqlAssignedLicenses)
		}
	}

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
			"creationType":      llx.StringDataPtr(u.GetCreationType()),
			"identities":        llx.ArrayData(identities, types.ResourceLike),
			"assignedLicenses":  llx.ArrayData(mqlAssignedLicensesList, types.ResourceLike),
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
	a.Job = plugin.TValue[any]{Data: jobDesc, State: plugin.StateIsSet}

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
	a.Contact = plugin.TValue[any]{Data: userContact, State: plugin.StateIsSet}

	return nil
}

func (a *mqlMicrosoftUser) auditlog() (*mqlMicrosoftUserAuditlog, error) {
	res, err := CreateResource(a.MqlRuntime, "microsoft.user.auditlog", map[string]*llx.RawData{
		"__id":   llx.StringData(a.Id.Data),
		"userId": llx.StringData(a.Id.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMicrosoftUserAuditlog), nil
}

func (a *mqlMicrosoftUserAuditlog) signins() ([]any, error) {
	ctx := context.Background()
	now := time.Now()
	dayAgo := now.AddDate(0, 0, -1)
	filter := fmt.Sprintf(
		"createdDateTime ge %s and createdDateTime lt %s and (userId eq '%s' or contains(tolower(userDisplayName), '%s'))",
		dayAgo.Format(time.RFC3339),
		now.Format(time.RFC3339),
		a.UserId.Data,
		a.UserId.Data)
	top := int32(50)
	res := []any{}
	signIns, err := fetchUserSignins(ctx, a.MqlRuntime, filter, top)
	if err != nil {
		return nil, err
	}
	for _, s := range signIns {
		res = append(res, s)
	}
	return res, nil
}

func fetchUserSignins(ctx context.Context, runtime *plugin.Runtime, filter string, top int32) ([]*mqlMicrosoftUserSignin, error) {
	conn := runtime.Connection.(*connection.Ms365Connection)
	betaClient, err := conn.BetaGraphClient()
	if err != nil {
		return nil, err
	}
	orderBy := "createdDateTime desc"
	req := &auditlogs.SignInsRequestBuilderGetRequestConfiguration{
		QueryParameters: &auditlogs.SignInsRequestBuilderGetQueryParameters{
			Top:     &top,
			Filter:  &filter,
			Orderby: []string{orderBy},
		},
	}
	resp, err := betaClient.AuditLogs().SignIns().Get(ctx, req)
	if err != nil {
		return nil, transformError(err)
	}

	res := []*mqlMicrosoftUserSignin{}
	for _, s := range resp.GetValue() {
		signIn, err := newMqlMicrosoftSignIn(runtime, s)
		if err != nil {
			return nil, err
		}
		res = append(res, signIn)
	}
	return res, nil
}

func (a *mqlMicrosoftUserAuditlog) lastInteractiveSignIn() (*mqlMicrosoftUserSignin, error) {
	signIns := a.GetSignins()
	if signIns.Error != nil {
		return nil, signIns.Error
	}
	if len(signIns.Data) == 0 {
		a.LastInteractiveSignIn.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	latest := signIns.Data[0].(*mqlMicrosoftUserSignin)
	return latest, nil
}

// Note: the audit log API by default excludes the non-interactive sign-ins. This is a workaround to fetch the last non-interactive sign-in.
// We could also grab those as part of the `sign-ins` query but then the amount of data would be much larger as non-interactive logins are much more frequent.
func (a *mqlMicrosoftUserAuditlog) lastNonInteractiveSignIn() (*mqlMicrosoftUserSignin, error) {
	ctx := context.Background()
	now := time.Now()
	dayAgo := now.AddDate(0, 0, -1)
	filter := fmt.Sprintf(
		"signInEventTypes/any(t: t ne 'interactiveUser') and createdDateTime ge %s and createdDateTime lt %s and (userId eq '%s' or contains(tolower(userDisplayName), '%s'))",
		dayAgo.Format(time.RFC3339),
		now.Format(time.RFC3339),
		a.UserId.Data,
		a.UserId.Data)
	top := int32(1)
	signIns, err := fetchUserSignins(ctx, a.MqlRuntime, filter, top)
	if err != nil {
		return nil, err
	}
	if len(signIns) == 0 {
		a.LastNonInteractiveSignIn.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return signIns[0], nil
}

func newMqlMicrosoftSignIn(runtime *plugin.Runtime, signIn betamodels.SignInable) (*mqlMicrosoftUserSignin, error) {
	mqlSignIn, err := CreateResource(runtime, "microsoft.user.signin",
		map[string]*llx.RawData{
			"__id":                llx.StringDataPtr(signIn.GetId()),
			"id":                  llx.StringDataPtr(signIn.GetId()),
			"createdDateTime":     llx.TimeDataPtr(signIn.GetCreatedDateTime()),
			"userId":              llx.StringDataPtr(signIn.GetUserId()),
			"clientAppUsed":       llx.StringDataPtr(signIn.GetClientAppUsed()),
			"resourceDisplayName": llx.StringDataPtr(signIn.GetResourceDisplayName()),
			"userDisplayName":     llx.StringDataPtr(signIn.GetUserDisplayName()),
			"appDisplayName":      llx.StringDataPtr(signIn.GetAppDisplayName()),
			"interactive":         llx.BoolDataPtr(signIn.GetIsInteractive()),
		})
	if err != nil {
		return nil, err
	}

	return mqlSignIn.(*mqlMicrosoftUserSignin), nil
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

func (a *mqlMicrosoftUser) job() (any, error) {
	return nil, a.populateJobContactData()
}

func (a *mqlMicrosoftUser) contact() (any, error) {
	return nil, a.populateJobContactData()
}

func (a *mqlMicrosoftUser) settings() (any, error) {
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

func (a *mqlMicrosoftUser) authenticationRequirements() (*mqlMicrosoftUserAuthenticationRequirements, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.BetaGraphClient()
	if err != nil {
		return nil, err
	}

	userID := a.Id.Data

	authRequirements, err := graphClient.Users().ByUserId(userID).Authentication().Requirements().Get(context.Background(), nil)
	if err != nil {
		return nil, transformError(err)
	}

	mqlAuthRequirements, err := CreateResource(a.MqlRuntime, "microsoft.userAuthenticationRequirements",
		map[string]*llx.RawData{
			"__id":            llx.StringData(userID),
			"perUserMfaState": llx.StringData(authRequirements.GetPerUserMfaState().String()),
		})
	if err != nil {
		return nil, err
	}

	return mqlAuthRequirements.(*mqlMicrosoftUserAuthenticationRequirements), nil
}

// Needs the permission AuditLog.Read.All
func (a *mqlMicrosoftUserAuthenticationMethods) registrationDetails() (*mqlMicrosoftUserAuthenticationMethodsUserRegistrationDetails, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	userID := a.__id
	if userID == "" {
		return nil, errors.New("cannot fetch user registration details without a user ID")
	}

	ctx := context.Background()
	userRegistrationDetails, err := graphClient.Reports().
		AuthenticationMethods().
		UserRegistrationDetails().
		ByUserRegistrationDetailsId(userID).
		Get(ctx, nil)
	if err != nil {
		return nil, transformError(err)
	}

	return newMqlUserRegistrationDetails(a.MqlRuntime, userRegistrationDetails)
}

func newMqlUserRegistrationDetails(runtime *plugin.Runtime, details models.UserRegistrationDetailsable) (*mqlMicrosoftUserAuthenticationMethodsUserRegistrationDetails, error) {
	if details.GetId() == nil {
		return nil, errors.New("user registration details response is missing an ID")
	}

	var userPrefMethodStr, userTypeStr string
	if details.GetUserPreferredMethodForSecondaryAuthentication() != nil {
		userPrefMethodStr = details.GetUserPreferredMethodForSecondaryAuthentication().String()
	}
	if details.GetUserType() != nil {
		userTypeStr = details.GetUserType().String()
	}

	data := map[string]*llx.RawData{
		"__id":                  llx.StringDataPtr(details.GetId()),
		"id":                    llx.StringDataPtr(details.GetId()),
		"isAdmin":               llx.BoolDataPtr(details.GetIsAdmin()),
		"isMfaCapable":          llx.BoolDataPtr(details.GetIsMfaCapable()),
		"isMfaRegistered":       llx.BoolDataPtr(details.GetIsMfaRegistered()),
		"isPasswordlessCapable": llx.BoolDataPtr(details.GetIsPasswordlessCapable()),
		"isSsprCapable":         llx.BoolDataPtr(details.GetIsSsprCapable()),
		"isSsprEnabled":         llx.BoolDataPtr(details.GetIsSsprEnabled()),
		"isSsprRegistered":      llx.BoolDataPtr(details.GetIsSsprRegistered()),
		"isSystemPreferredAuthenticationMethodEnabled":  llx.BoolDataPtr(details.GetIsSystemPreferredAuthenticationMethodEnabled()),
		"lastUpdatedDateTime":                           llx.TimeDataPtr(details.GetLastUpdatedDateTime()),
		"methodsRegistered":                             llx.ArrayData(convert.SliceAnyToInterface(details.GetMethodsRegistered()), types.String),
		"systemPreferredAuthenticationMethods":          llx.ArrayData(convert.SliceAnyToInterface(details.GetSystemPreferredAuthenticationMethods()), types.String),
		"userDisplayName":                               llx.StringDataPtr(details.GetUserDisplayName()),
		"userPreferredMethodForSecondaryAuthentication": llx.StringData(userPrefMethodStr),
		"userPrincipalName":                             llx.StringDataPtr(details.GetUserPrincipalName()),
		"userType":                                      llx.StringData(userTypeStr),
	}

	resource, err := CreateResource(runtime, "microsoft.user.authenticationMethods.userRegistrationDetails", data)
	if err != nil {
		return nil, err
	}

	return resource.(*mqlMicrosoftUserAuthenticationMethodsUserRegistrationDetails), nil
}

func (a *mqlMicrosoftUser) licenseDetails() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	userID := a.Id.Data
	ctx := context.Background()

	// Permissions: User.Read.All, Directory.Read.All
	details, err := graphClient.Users().ByUserId(userID).LicenseDetails().Get(ctx, nil)
	if err != nil {
		return nil, transformError(err)
	}

	results := []any{}
	for _, d := range details.GetValue() {
		mqlDetail, err := newMqlMicrosoftUserLicenseDetail(a.MqlRuntime, d)
		if err != nil {
			return nil, err
		}
		results = append(results, mqlDetail)
	}

	return results, nil
}

func newMqlMicrosoftUserLicenseDetail(runtime *plugin.Runtime, d models.LicenseDetailsable) (*mqlMicrosoftUserLicenseDetail, error) {
	if d.GetId() == nil {
		return nil, errors.New("license detail response is missing an ID")
	}

	var skuId, skuPartNumber string
	if d.GetSkuId() != nil {
		skuId = d.GetSkuId().String()
	}
	if d.GetSkuPartNumber() != nil {
		skuPartNumber = *d.GetSkuPartNumber()
	}

	servicePlans := []any{}
	for i, sp := range d.GetServicePlans() {
		planId := fmt.Sprintf("%s-service-plans-%d", *d.GetId(), +i)

		servicePlan, err := CreateResource(runtime, "microsoft.user.licenseDetail.servicePlanInfo",
			map[string]*llx.RawData{
				"__id":               llx.StringData(planId),
				"appliesTo":          llx.StringDataPtr(sp.GetAppliesTo()),
				"provisioningStatus": llx.StringDataPtr(sp.GetProvisioningStatus()),
				"servicePlanId":      llx.StringData(sp.GetServicePlanId().String()),
				"servicePlanName":    llx.StringDataPtr(sp.GetServicePlanName()),
			})
		if err != nil {
			return nil, err
		}
		servicePlans = append(servicePlans, servicePlan)
	}

	data := map[string]*llx.RawData{
		"__id":          llx.StringDataPtr(d.GetId()),
		"id":            llx.StringDataPtr(d.GetId()),
		"skuId":         llx.StringData(skuId),
		"skuPartNumber": llx.StringData(skuPartNumber),
		"servicePlans":  llx.ArrayData(servicePlans, types.ResourceLike),
	}

	resource, err := CreateResource(runtime, "microsoft.user.licenseDetail", data)
	if err != nil {
		return nil, err
	}

	return resource.(*mqlMicrosoftUserLicenseDetail), nil
}
