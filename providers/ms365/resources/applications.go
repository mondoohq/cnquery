// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/microsoftgraph/msgraph-sdk-go/applications"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
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
	resp, err := graphClient.Applications().Get(ctx, &applications.ApplicationsRequestBuilderGetRequestConfiguration{})
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

func initMicrosoftApplication(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// we only look up the package, if we have been supplied by its name and nothing else
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

// newMqlMicrosoftApplication creates a new mqlMicrosoftApplication resource
func newMqlMicrosoftApplication(runtime *plugin.Runtime, app models.Applicationable) (*mqlMicrosoftApplication, error) {
	info, _ := convert.JsonToDictSlice(app.GetInfo())

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

	mqlResource, err := CreateResource(runtime, "microsoft.application",
		map[string]*llx.RawData{
			"__id":            llx.StringDataPtr(app.GetId()),
			"id":              llx.StringDataPtr(app.GetId()),
			"appId":           llx.StringDataPtr(app.GetAppId()),
			"createdDateTime": llx.TimeDataPtr(app.GetCreatedDateTime()),
			"createdAt":       llx.TimeDataPtr(app.GetCreatedDateTime()),
			"displayName":     llx.StringDataPtr(app.GetDisplayName()),
			"name":            llx.StringDataPtr(app.GetDisplayName()),
			"description":     llx.StringDataPtr(app.GetDescription()),
			"notes":           llx.StringDataPtr(app.GetNotes()),
			"publisherDomain": llx.StringDataPtr(app.GetPublisherDomain()),
			"signInAudience":  llx.StringDataPtr(app.GetSignInAudience()),
			"tags":            llx.ArrayData(convert.SliceAnyToInterface(app.GetTags()), types.String),
			"identifierUris":  llx.ArrayData(convert.SliceAnyToInterface(app.GetIdentifierUris()), types.String),
			"info":            llx.DictData(info),
			"secrets":         llx.ArrayData(secrets, types.Resource("microsoft.passwordCredential")),
			"certificates":    llx.ArrayData(certificates, types.Resource("microsoft.keyCredential")),
		})
	if err != nil {
		return nil, err
	}
	return mqlResource.(*mqlMicrosoftApplication), nil
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
