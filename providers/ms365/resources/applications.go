// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/url"
	"time"

	"github.com/microsoftgraph/msgraph-sdk-go/applications"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/serviceprincipals"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/ms365/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func (a *mqlMicrosoft) applications() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	top := int32(999)
	resp, err := graphClient.Applications().Get(ctx, &applications.ApplicationsRequestBuilderGetRequestConfiguration{
		QueryParameters: &applications.ApplicationsRequestBuilderGetQueryParameters{
			Top: &top,
		},
	})
	if err != nil {
		return nil, transformError(err)
	}

	res := []interface{}{}
	apps := resp.GetValue()
	for _, app := range apps {
		mqlResource, err := newMqlMicrosoftApplication(a.MqlRuntime, app)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

// newMqlMicrosoftApplication creates a new mqlMicrosoftApplication resource
// see https://learn.microsoft.com/en-us/entra/identity-platform/reference-microsoft-graph-app-manifest for a
// better description of the fields
func newMqlMicrosoftApplication(runtime *plugin.Runtime, app models.Applicationable) (*mqlMicrosoftApplication, error) {
	// certificates
	var certificates []interface{}
	keycredentials := app.GetKeyCredentials()
	for _, keycredential := range keycredentials {
		cert, err := newMqlMicrosoftKeyCredential(runtime, keycredential)
		if err != nil {
			return nil, err
		}
		certificates = append(certificates, cert)
	}
	// secrets
	var secrets []interface{}
	clientSecrets := app.GetPasswordCredentials()
	for _, clientSecret := range clientSecrets {
		secret, err := newMqlMicrosoftPasswordCredential(runtime, clientSecret)
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, secret)
	}

	info, err := convert.JsonToDict(newAppInformationUrl(app.GetInfo()))
	// https://learn.microsoft.com/en-us/entra/identity-platform/reference-microsoft-graph-app-manifest#api-attribute
	apiInfo, err := convert.JsonToDict(newApiApplication(app.GetApi()))
	// https://learn.microsoft.com/en-us/entra/identity-platform/reference-microsoft-graph-app-manifest#web-attribute
	webInfo, err := convert.JsonToDict(newWebApplication(app.GetWeb()))
	// https://learn.microsoft.com/en-us/entra/identity-platform/reference-microsoft-graph-app-manifest#spa-attribute
	spaInfo, err := convert.JsonToDict(newSpaApplication(app.GetSpa()))

	certification, err := convert.JsonToDict(newCertificationable(app.GetCertification()))
	optionalClaims, err := convert.JsonToDict(newOptionalClaimsable(app.GetOptionalClaims()))
	servicePrincipalLockConfiguration, err := convert.JsonToDict(newServicePrincipalLockConfiguration(app.GetServicePrincipalLockConfiguration()))
	requestSignatureVerification, err := convert.JsonToDict(newRequestSignatureVerification(app.GetRequestSignatureVerification()))
	parentalControlSettings, err := convert.JsonToDict(newParentalControlSettings(app.GetParentalControlSettings()))
	publicClient, err := convert.JsonToDict(newPublicClientApplication(app.GetPublicClient()))

	var nativeAuthenticationApisEnabled *string
	if app.GetNativeAuthenticationApisEnabled() != nil {
		val := app.GetNativeAuthenticationApisEnabled().String()
		nativeAuthenticationApisEnabled = &val
	}

	mqlAppRoleList := []interface{}{}
	appRoles := app.GetAppRoles()
	for i := range appRoles {
		appRole := appRoles[i]

		uuid := appRole.GetId()
		if uuid == nil {
			log.Debug().Msg("appRole ID is nil")
			continue
		}

		mqlAppRoleResource, err := CreateResource(runtime, "microsoft.application.role",
			map[string]*llx.RawData{
				"__id":               llx.StringData(uuid.String()),
				"id":                 llx.StringData(uuid.String()),
				"name":               llx.StringDataPtr(appRole.GetDisplayName()),
				"description":        llx.StringDataPtr(appRole.GetDescription()),
				"value":              llx.StringDataPtr(appRole.GetValue()),
				"allowedMemberTypes": llx.ArrayData(convert.SliceAnyToInterface(appRole.GetAllowedMemberTypes()), types.String),
				"isEnabled":          llx.BoolDataPtr(appRole.GetIsEnabled()),
			})
		if err != nil {
			return nil, err
		}
		mqlAppRoleList = append(mqlAppRoleList, mqlAppRoleResource)
	}

	mqlResource, err := CreateResource(runtime, "microsoft.application",
		map[string]*llx.RawData{
			"__id":                              llx.StringDataPtr(app.GetId()),
			"id":                                llx.StringDataPtr(app.GetId()),
			"appId":                             llx.StringDataPtr(app.GetAppId()),
			"applicationTemplateId":             llx.StringDataPtr(app.GetApplicationTemplateId()),
			"createdDateTime":                   llx.TimeDataPtr(app.GetCreatedDateTime()),
			"createdAt":                         llx.TimeDataPtr(app.GetCreatedDateTime()),
			"displayName":                       llx.StringDataPtr(app.GetDisplayName()),
			"disabledByMicrosoftStatus":         llx.StringDataPtr(app.GetDisabledByMicrosoftStatus()),
			"groupMembershipClaims":             llx.StringDataPtr(app.GetGroupMembershipClaims()),
			"name":                              llx.StringDataPtr(app.GetDisplayName()),
			"description":                       llx.StringDataPtr(app.GetDescription()),
			"notes":                             llx.StringDataPtr(app.GetNotes()),
			"publisherDomain":                   llx.StringDataPtr(app.GetPublisherDomain()),
			"signInAudience":                    llx.StringDataPtr(app.GetSignInAudience()),
			"tags":                              llx.ArrayData(convert.SliceAnyToInterface(app.GetTags()), types.String),
			"identifierUris":                    llx.ArrayData(convert.SliceAnyToInterface(app.GetIdentifierUris()), types.String),
			"info":                              llx.DictData(info),
			"api":                               llx.DictData(apiInfo),
			"web":                               llx.DictData(webInfo),
			"spa":                               llx.DictData(spaInfo),
			"secrets":                           llx.ArrayData(secrets, types.Resource("microsoft.passwordCredential")),
			"certificates":                      llx.ArrayData(certificates, types.Resource("microsoft.keyCredential")),
			"isDeviceOnlyAuthSupported":         llx.BoolDataPtr(app.GetIsDeviceOnlyAuthSupported()),
			"isFallbackPublicClient":            llx.BoolDataPtr(app.GetIsFallbackPublicClient()),
			"nativeAuthenticationApisEnabled":   llx.StringDataPtr(nativeAuthenticationApisEnabled),
			"serviceManagementReference":        llx.StringDataPtr(app.GetServiceManagementReference()),
			"tokenEncryptionKeyId":              llx.StringDataPtr(newUuidString(app.GetTokenEncryptionKeyId())),
			"samlMetadataUrl":                   llx.StringDataPtr(app.GetSamlMetadataUrl()),
			"defaultRedirectUri":                llx.StringDataPtr(app.GetDefaultRedirectUri()),
			"certification":                     llx.DictData(certification),
			"optionalClaims":                    llx.DictData(optionalClaims),
			"servicePrincipalLockConfiguration": llx.DictData(servicePrincipalLockConfiguration),
			"requestSignatureVerification":      llx.DictData(requestSignatureVerification),
			"parentalControlSettings":           llx.DictData(parentalControlSettings),
			"publicClient":                      llx.DictData(publicClient),
			"appRoles":                          llx.ArrayData(mqlAppRoleList, types.Resource("microsoft.application.role")),
		})
	if err != nil {
		return nil, err
	}
	return mqlResource.(*mqlMicrosoftApplication), nil
}

func initMicrosoftApplication(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// we only look up the application if we have been supplied by its name and nothing else
	raw, ok := args["name"]
	if !ok || len(args) != 1 {
		return args, nil, nil
	}
	name := raw.Value.(string)

	conn := runtime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, nil, err
	}

	// https://graph.microsoft.com/v1.0/servicePrincipals?$count=true&$search="displayName:teams"&$select=id,displayName
	filter := fmt.Sprintf("displayName eq '%s'", url.QueryEscape(name))
	ctx := context.Background()
	resp, err := graphClient.Applications().Get(ctx, &applications.ApplicationsRequestBuilderGetRequestConfiguration{
		QueryParameters: &applications.ApplicationsRequestBuilderGetQueryParameters{
			Filter: &filter,
		},
	})
	if err != nil {
		return nil, nil, transformError(err)
	}

	val := resp.GetValue()
	if len(val) == 0 {
		return nil, nil, errors.New("application not found")
	}

	applicationId := val[0].GetId()
	if applicationId == nil {
		return nil, nil, errors.New("application id not found")
	}

	// https://graph.microsoft.com/v1.0/applications/{application-id}
	app, err := graphClient.Applications().ByApplicationId(*applicationId).Get(ctx, &applications.ApplicationItemRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, nil, transformError(err)
	}
	mqlMsApp, err := newMqlMicrosoftApplication(runtime, app)
	if err != nil {
		return nil, nil, err
	}

	return nil, mqlMsApp, nil
}

// hasExpiredCredentials returns true if any of the credentials of the application are expired
func (a *mqlMicrosoftApplication) hasExpiredCredentials() (bool, error) {
	certificates := a.GetCertificates()
	for _, val := range certificates.Data {
		cert := val.(*mqlMicrosoftKeyCredential)
		if cert.GetExpired().Data {
			return true, nil
		}
	}

	secrets := a.GetSecrets()
	for _, val := range secrets.Data {
		secret := val.(*mqlMicrosoftPasswordCredential)
		if secret.GetExpired().Data {
			return true, nil
		}
	}
	return false, nil
}

func (a *mqlMicrosoftApplication) servicePrincipal() (*mqlMicrosoftServiceprincipal, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	filter := fmt.Sprintf("appId eq '%s'", a.GetAppId().Data)
	resp, err := graphClient.ServicePrincipals().Get(ctx, &serviceprincipals.ServicePrincipalsRequestBuilderGetRequestConfiguration{
		QueryParameters: &serviceprincipals.ServicePrincipalsRequestBuilderGetQueryParameters{
			Filter: &filter,
		},
	})
	servicePrincipals := resp.GetValue()
	if len(servicePrincipals) == 0 {
		return nil, errors.New("service principal not found")
	}
	if len(servicePrincipals) > 1 {
		return nil, errors.New("multiple service principals found")
	}
	return newMqlMicrosoftServicePrincipal(a.MqlRuntime, servicePrincipals[0])
}

func (a *mqlMicrosoftApplication) owners() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)

	msResource, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "microsoft", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	mqlMicrsoftResource := msResource.(*mqlMicrosoft)

	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.Applications().ByApplicationId(a.GetId().Data).Owners().Get(ctx, &applications.ItemOwnersRequestBuilderGetRequestConfiguration{
		QueryParameters: &applications.ItemOwnersRequestBuilderGetQueryParameters{
			Select: []string{"id"},
		},
	})
	if err != nil {
		return nil, transformError(err)
	}

	res := []interface{}{}
	for i := range resp.GetValue() {
		ownerId := resp.GetValue()[i].GetId()
		if ownerId == nil {
			continue
		}

		// if the user is already indexed, we can reuse it
		userResource, ok := mqlMicrsoftResource.userById(*ownerId)
		if ok {
			res = append(res, userResource)
			continue
		}

		// otherwise we create a new user resource
		newUserResource, err := a.MqlRuntime.NewResource(a.MqlRuntime, "microsoft.user", map[string]*llx.RawData{
			"id": llx.StringDataPtr(ownerId),
		})
		if err != nil {
			return nil, err
		}
		mqlMicrsoftResource.index(newUserResource.(*mqlMicrosoftUser))
		res = append(res, newUserResource)
	}
	return res, nil
}

// newMqlMicrosoftKeyCredential creates a new mqlMicrosoftKeyCredential resource
func newMqlMicrosoftKeyCredential(runtime *plugin.Runtime, app models.KeyCredentialable) (*mqlMicrosoftKeyCredential, error) {
	endDate := app.GetEndDateTime()
	expired := true
	if endDate != nil {
		expired = endDate.Before(time.Now())
	}

	mqlResource, err := CreateResource(runtime, "microsoft.keyCredential",
		map[string]*llx.RawData{
			"__id":        llx.StringData(app.GetKeyId().String()),
			"keyId":       llx.StringData(app.GetKeyId().String()),
			"description": llx.StringDataPtr(app.GetDisplayName()),
			"usage":       llx.StringDataPtr(app.GetUsage()),
			"thumbprint":  llx.StringData(base64.StdEncoding.EncodeToString(app.GetCustomKeyIdentifier())),
			"type":        llx.StringDataPtr(app.GetTypeEscaped()),
			"expires":     llx.TimeDataPtr(endDate),
			"expired":     llx.BoolData(expired),
		})
	if err != nil {
		return nil, err
	}
	return mqlResource.(*mqlMicrosoftKeyCredential), nil
}

// newMqlMicrosoftPasswordCredential creates a new mqlMicrosoftPasswordCredential resource
func newMqlMicrosoftPasswordCredential(runtime *plugin.Runtime, app models.PasswordCredentialable) (*mqlMicrosoftPasswordCredential, error) {
	endDate := app.GetEndDateTime()
	expired := true
	if endDate != nil {
		expired = endDate.Before(time.Now())
	}

	mqlResource, err := CreateResource(runtime, "microsoft.passwordCredential",
		map[string]*llx.RawData{
			"__id":        llx.StringData(app.GetKeyId().String()),
			"keyId":       llx.StringData(app.GetKeyId().String()),
			"description": llx.StringDataPtr(app.GetDisplayName()),
			"hint":        llx.StringDataPtr(app.GetHint()),
			"expires":     llx.TimeDataPtr(endDate),
			"expired":     llx.BoolData(expired),
		})
	if err != nil {
		return nil, err
	}
	return mqlResource.(*mqlMicrosoftPasswordCredential), nil
}
