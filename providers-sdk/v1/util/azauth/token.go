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
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
)

type TokenResolverFn (func() (azcore.TokenCredential, error))

func WithCliCredentials(opts *azidentity.AzureCLICredentialOptions) TokenResolverFn {
	return func() (azcore.TokenCredential, error) {
		return azidentity.NewAzureCLICredential(opts)
	}
}

func WithEnvCredentials(opts *azidentity.EnvironmentCredentialOptions) TokenResolverFn {
	return func() (azcore.TokenCredential, error) {
		return azidentity.NewEnvironmentCredential(opts)
	}
}

// sometimes we run into a 'managed identity timed out' error when using a managed identity.
// This function mimics the behavior of the NewManagedIdentityCredential, but with a higher timeout and retries
func WithRetryableManagedIdentityCredentials(timeout time.Duration, attempts int, opts *azidentity.ManagedIdentityCredentialOptions) TokenResolverFn {
	return func() (azcore.TokenCredential, error) {
		mic, err := azidentity.NewManagedIdentityCredential(opts)
		if err != nil {
			return nil, err
		}
		return &retryableManagedIdentityCredential{mic: *mic, timeout: timeout, attempts: attempts}, nil
	}
}

func WithWorkloadIdentityCredentials(opts *azidentity.WorkloadIdentityCredentialOptions) TokenResolverFn {
	return func() (azcore.TokenCredential, error) {
		return azidentity.NewWorkloadIdentityCredential(opts)
	}
}

func BuildChainedToken(opts ...TokenResolverFn) (*azidentity.ChainedTokenCredential, error) {
	chain := []azcore.TokenCredential{}
	for _, fn := range opts {
		cred, err := fn()
		if err == nil {
			chain = append(chain, cred)
		}
	}
	return azidentity.NewChainedTokenCredential(chain, nil)
}

func GetChainedToken(options *azidentity.DefaultAzureCredentialOptions) (*azidentity.ChainedTokenCredential, error) {
	opts := []TokenResolverFn{
		WithCliCredentials(&azidentity.AzureCLICredentialOptions{AdditionallyAllowedTenants: []string{"*"}}),
		WithEnvCredentials(&azidentity.EnvironmentCredentialOptions{ClientOptions: options.ClientOptions}),
		WithRetryableManagedIdentityCredentials(5*time.Second, 3, &azidentity.ManagedIdentityCredentialOptions{ClientOptions: options.ClientOptions}),
		WithWorkloadIdentityCredentials(&azidentity.WorkloadIdentityCredentialOptions{
			ClientOptions:            options.ClientOptions,
			DisableInstanceDiscovery: options.DisableInstanceDiscovery,
			TenantID:                 options.TenantID,
		}),
	}
	return BuildChainedToken(opts...)
}

type retryableManagedIdentityCredential struct {
	mic      azidentity.ManagedIdentityCredential
	attempts int
	timeout  time.Duration
}

func (t *retryableManagedIdentityCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	// sanity check to ensure we get at least one attempt
	if t.attempts < 1 {
		t.attempts = 1
	}

	errs := []error{}
	for i := 0; i < t.attempts; i++ {
		tk, err := t.tryGetToken(ctx, opts)
		if err == nil {
			return tk, nil
		}
		log.Debug().
			Err(err).
			Int("attempt", i+1).
			Int("max_attempts", t.attempts).
			Msg("failed to get managed identity token (may retry)")
		errs = append(errs, err)
	}

	log.Error().
		Int("num_attempts", t.attempts).
		Msg("failed to get managed identity token (max retries reached)")
	return azcore.AccessToken{}, errors.Join(errs...)
}

func (t *retryableManagedIdentityCredential) tryGetToken(ctx context.Context, opts policy.TokenRequestOptions) (tk azcore.AccessToken, err error) {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
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
	return
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
