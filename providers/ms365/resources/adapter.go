// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/cockroachdb/errors"
	"github.com/microsoft/kiota-abstractions-go/authentication"
	absauth "github.com/microsoft/kiota-abstractions-go/authentication"
	a "github.com/microsoft/kiota-authentication-azure-go"
	msgraphsdkgo "github.com/microsoftgraph/msgraph-sdk-go"
	"go.mondoo.com/cnquery/v9/providers/ms365/connection"

	msgraphclient "github.com/microsoftgraph/msgraph-sdk-go"
)

const DefaultMSGraphScope = "https://graph.microsoft.com/.default"

var DefaultMSGraphScopes = []string{DefaultMSGraphScope}

func newGraphRequestAdapterWithFn(providerFn func() (absauth.AuthenticationProvider, error)) (*msgraphsdkgo.GraphRequestAdapter, error) {
	auth, err := providerFn()
	if err != nil {
		return nil, errors.Wrap(err, "authentication provider error")
	}

	return msgraphsdkgo.NewGraphRequestAdapter(auth)
}

func graphClient(conn *connection.Ms365Connection) (*msgraphclient.GraphServiceClient, error) {
	token := conn.Token()

	providerFunc := func() (authentication.AuthenticationProvider, error) {
		return a.NewAzureIdentityAuthenticationProviderWithScopes(token, DefaultMSGraphScopes)
	}
	adapter, err := newGraphRequestAdapterWithFn(providerFunc)
	if err != nil {
		return nil, err
	}
	graphClient := msgraphclient.NewGraphServiceClient(adapter)
	return graphClient, nil
}
