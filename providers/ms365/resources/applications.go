// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/base64"
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

// expiredCredentials returns true if any of the credentials of the application are expired
func (a *mqlMicrosoftApplication) expiredCredentials() (bool, error) {
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
