// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azauth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// sometimes we run into a 'managed identity timed out' error when using a managed identity.
// according to https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/azidentity/TROUBLESHOOTING.md#troubleshoot-defaultazurecredential-authentication-issues
// we should instead use the NewManagedIdentityCredential directly.
// This function mimics the behavior of the DefaultAzureCredential, but with a higher timeout on the managed identity
func GetTokenChain(options *azidentity.DefaultAzureCredentialOptions) (*azidentity.ChainedTokenCredential, error) {
	if options == nil {
		options = &azidentity.DefaultAzureCredentialOptions{}
	}

	chain := []azcore.TokenCredential{}

	cli, err := azidentity.NewAzureCLICredential(&azidentity.AzureCLICredentialOptions{})
	if err == nil {
		chain = append(chain, cli)
	}
	envCred, err := azidentity.NewEnvironmentCredential(&azidentity.EnvironmentCredentialOptions{ClientOptions: options.ClientOptions})
	if err == nil {
		chain = append(chain, envCred)
	}
	mic, err := azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{ClientOptions: options.ClientOptions})
	if err == nil {
		timedMic := &timedManagedIdentityCredential{mic: *mic, timeout: 5 * time.Second}
		chain = append(chain, timedMic)
	}
	wic, err := azidentity.NewWorkloadIdentityCredential(&azidentity.WorkloadIdentityCredentialOptions{
		ClientOptions:            options.ClientOptions,
		DisableInstanceDiscovery: options.DisableInstanceDiscovery,
		TenantID:                 options.TenantID,
	})
	if err == nil {
		chain = append(chain, wic)
	}

	return azidentity.NewChainedTokenCredential(chain, nil)
}

type timedManagedIdentityCredential struct {
	mic     azidentity.ManagedIdentityCredential
	timeout time.Duration
}

func (t *timedManagedIdentityCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	var tk azcore.AccessToken
	var err error
	if t.timeout > 0 {
		c, cancel := context.WithTimeout(ctx, t.timeout)
		defer cancel()
		tk, err = t.mic.GetToken(c, opts)
		if err != nil {
			var authFailedErr *azidentity.AuthenticationFailedError
			if errors.As(err, &authFailedErr) && strings.Contains(err.Error(), "context deadline exceeded") {
				err = azidentity.NewCredentialUnavailableError("managed identity request timed out")
			}
		} else {
			// some managed identity implementation is available, so don't apply the timeout to future calls
			t.timeout = 0
		}
	} else {
		tk, err = t.mic.GetToken(ctx, opts)
	}
	return tk, err
}
