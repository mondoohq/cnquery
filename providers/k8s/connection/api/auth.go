// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"context"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/azauth"
	"go.mondoo.com/cnquery/v11/providers/k8s/connection/shared"
	"k8s.io/client-go/rest"
)

const (
	// AKS Microsoft Entra server application ID used by AKS for user authentication. The bearer token needs to be issued
	// for this application (aud claim in JWT).
	// https://learn.microsoft.com/en-us/azure/aks/kubelogin-authentication#how-to-use-kubelogin-with-aks
	serverAppId  = "6dae42f8-4368-4678-94ff-3960e28e3630"
	defaultScope = "/.default"
)

// attempt to get a bearer token using the kubelogin flow and attach it to the rest config
func attemptKubeloginAuthFlow(asset *inventory.Asset, config *rest.Config) error {
	var err error
	kubeloginAuth := false
	if val, ok := asset.Connections[0].Options[shared.OPTION_KUBELOGIN]; ok {
		kubeloginAuth, err = strconv.ParseBool(val)
		if err != nil {
			return errors.Wrap(err, "could not parse boolean from the kubelogin option value")
		}
	}

	if !kubeloginAuth {
		return nil
	}

	log.Debug().Msg("attempting to get a bearer token using the kubelogin flow")

	// the managed identity token credential is used for AKS authentication
	chainedToken, err := azauth.GetChainedToken(&azidentity.DefaultAzureCredentialOptions{
		ClientOptions: azcore.ClientOptions{Cloud: cloud.AzurePublic},
	})
	if err != nil {
		return errors.Wrap(err, "failed to get chained token credential for Azure AKS authentication")
	}

	scope := serverAppId + defaultScope
	token, err := chainedToken.GetToken(context.Background(), policy.TokenRequestOptions{
		Scopes: []string{scope},
	})
	if err != nil {
		return errors.Wrap(err, "failed to get access token for Azure AKS authentication")
	}

	log.Debug().Str("bearer_token", token.Token).Msg("got access token")

	config.BearerToken = token.Token

	// clear the exec provider since the code above bypasses the need to run the command
	// `kubelogin get-token --server-id {serverAppId}` since that would require the kubelogin CLI tool to be present
	config.ExecProvider = nil

	return nil
}
