// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/cockroachdb/errors"
	"github.com/microsoft/kiota-abstractions-go/authentication"
	a "github.com/microsoft/kiota-authentication-azure-go"
	msgraphsdkgo "github.com/microsoftgraph/msgraph-sdk-go"
	"go.mondoo.com/cnquery/v10/providers/ms365/connection"
)

var DefaultMSGraphScopes = []string{connection.DefaultMSGraphScope}

func newGraphRequestAdapterWithFn(providerFn func() (authentication.AuthenticationProvider, error)) (*msgraphsdkgo.GraphRequestAdapter, error) {
	auth, err := providerFn()
	if err != nil {
		return nil, errors.Wrap(err, "authentication provider error")
	}

	return msgraphsdkgo.NewGraphRequestAdapter(auth)
}

func graphClient(conn *connection.Ms365Connection) (*msgraphsdkgo.GraphServiceClient, error) {
	token := conn.Token()

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
