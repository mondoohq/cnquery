// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package aksauth

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	// AKS Microsoft Entra server application ID used by AKS for user authentication. The bearer token needs to be issued
	// for this application (aud claim in JWT).
	// https://learn.microsoft.com/en-us/azure/aks/kubelogin-authentication#how-to-use-kubelogin-with-aks
	serverAppId  = "6dae42f8-4368-4678-94ff-3960e28e3630"
	defaultScope = "/.default"
)

// attempt to get a bearer token using the kubelogin flow and attach it to the rest config
func GetKubeloginBearerToken(token azcore.TokenCredential) (string, error) {
	log.Debug().Msg("aks kubelogin> attempting to get a bearer token using the kubelogin flow")
	scope := serverAppId + defaultScope
	rawToken, err := token.GetToken(context.Background(), policy.TokenRequestOptions{
		Scopes: []string{scope},
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to get access token for Azure AKS authentication")
	}

	log.Debug().Msg("aks kubelogin> got access token")
	return rawToken.Token, nil
}
