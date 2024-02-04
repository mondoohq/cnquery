// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	azcore "github.com/Azure/azure-sdk-for-go/sdk/azcore"
	errors "github.com/cockroachdb/errors"
	"github.com/microsoft/kiota-abstractions-go/authentication"
	a "github.com/microsoft/kiota-authentication-azure-go"
	msgraphsdkgo "github.com/microsoftgraph/msgraph-sdk-go"
)

const DefaultMSGraphScope = "https://graph.microsoft.com/.default"

var DefaultMSGraphScopes = []string{DefaultMSGraphScope}

func newGraphRequestAdapterWithFn(providerFn func() (authentication.AuthenticationProvider, error)) (*msgraphsdkgo.GraphRequestAdapter, error) {
	auth, err := providerFn()
	if err != nil {
		return nil, errors.Wrap(err, "authentication provider error")
	}

	return msgraphsdkgo.NewGraphRequestAdapter(auth)
}

func graphClient(token azcore.TokenCredential) (*msgraphsdkgo.GraphServiceClient, error) {
	providerFunc := func() (authentication.AuthenticationProvider, error) {
		return a.NewAzureIdentityAuthenticationProviderWithScopes(token, DefaultMSGraphScopes)
	}
	adapter, err := newGraphRequestAdapterWithFn(providerFunc)
	if err != nil {
		return nil, err
	}
	graphClient := msgraphsdkgo.NewGraphServiceClient(adapter)
	return graphClient, nil
}

func (conn *Ms365Connection) GraphClient() (*msgraphsdkgo.GraphServiceClient, error) {
	return graphClient(conn.Token())
}
