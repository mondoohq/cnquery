// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azauth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"
)

// sometimes we run into a 'managed identity timed out' error when using a managed identity.
// according to https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/azidentity/TROUBLESHOOTING.md#troubleshoot-defaultazurecredential-authentication-issues
// we should instead use the NewManagedIdentityCredential directly.
// This function mimics the behavior of the DefaultAzureCredential, but with a higher timeout on the managed identity
func GetChainedToken(options *azidentity.DefaultAzureCredentialOptions) (*azidentity.ChainedTokenCredential, error) {
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

func GetTokenFromCredential(credential *vault.Credential, tenantId, clientId string) (azcore.TokenCredential, error) {
	var azCred azcore.TokenCredential
	var err error
	// fallback to default authorizer if no credentials are specified
	if credential == nil {
		log.Debug().Msg("using default azure token chain resolver")
		azCred, err = GetChainedToken(&azidentity.DefaultAzureCredentialOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "error creating CLI credentials")
		}
	} else {
		switch credential.Type {
		case vault.CredentialType_pkcs12:
			certs, privateKey, err := azidentity.ParseCertificates(credential.Secret, []byte(credential.Password))
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("could not parse provided certificate at %s", credential.PrivateKeyPath))
			}
			azCred, err = azidentity.NewClientCertificateCredential(tenantId, clientId, certs, privateKey, &azidentity.ClientCertificateCredentialOptions{})
			if err != nil {
				return nil, errors.Wrap(err, "error creating credentials from a certificate")
			}
		case vault.CredentialType_password:
			azCred, err = azidentity.NewClientSecretCredential(tenantId, clientId, string(credential.Secret), &azidentity.ClientSecretCredentialOptions{})
			if err != nil {
				return nil, errors.Wrap(err, "error creating credentials from a secret")
			}
		default:
			return nil, errors.New("invalid secret configuration for microsoft transport: " + credential.Type.String())
		}
	}
	return azCred, nil
}
