// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package azauth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

type TokenResolverFn (func() (azcore.TokenCredential, error))

func WithStaticToken(t azcore.TokenCredential) TokenResolverFn {
	return func() (azcore.TokenCredential, error) {
		return t, nil
	}
}

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

func GetDefaultChainedToken(options *azidentity.DefaultAzureCredentialOptions) (*azidentity.ChainedTokenCredential, error) {
	if options == nil {
		options = &azidentity.DefaultAzureCredentialOptions{}
	}
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

// GetWorkloadIdentityToken builds a keyless credential that exchanges a
// Mondoo-issued OIDC token (written to federatedTokenFile) for an Entra access
// token via the federated identity credential on the app registration.
func GetWorkloadIdentityToken(tenantId, clientId, federatedTokenFile string) (azcore.TokenCredential, error) {
	return azidentity.NewWorkloadIdentityCredential(&azidentity.WorkloadIdentityCredentialOptions{
		TenantID:      tenantId,
		ClientID:      clientId,
		TokenFilePath: federatedTokenFile,
	})
}

func GetTokenFromCredential(credential *vault.Credential, tenantId, clientId string) (azcore.TokenCredential, error) {
	var azCred azcore.TokenCredential
	var err error
	usedDefaultChain := credential == nil
	// fallback to default authorizer if no credentials are specified
	if credential == nil {
		log.Info().Msg("no Azure credentials were provided; trying to sign in with your local Azure CLI session, Azure environment variables, or a managed identity")
		azCred, err = GetDefaultChainedToken(&azidentity.DefaultAzureCredentialOptions{})
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
	return &guidedCredential{inner: azCred, usedDefaultChain: usedDefaultChain}, nil
}

// guidedCredential decorates an azcore.TokenCredential so that a failed sign-in
// surfaces a plain-language, actionable message instead of the raw error from
// deep inside the Azure SDK (for example a JSON decode error when the Azure CLI
// returns something other than a token). usedDefaultChain records whether we
// fell back to signing in with whatever Azure login is available on the machine
// because no credentials were supplied, which changes the guidance we give.
type guidedCredential struct {
	inner            azcore.TokenCredential
	usedDefaultChain bool
}

func (c *guidedCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	tk, err := c.inner.GetToken(ctx, opts)
	if err != nil {
		return azcore.AccessToken{}, enrichTokenError(err, c.usedDefaultChain)
	}
	return tk, nil
}

// enrichTokenError wraps a sign-in failure with guidance tailored to how the
// credential was configured. The original error is always preserved so no
// diagnostic detail is lost.
func enrichTokenError(err error, usedDefaultChain bool) error {
	if !usedDefaultChain {
		return errors.Wrap(err, "Azure sign-in with the provided credentials failed; double-check the tenant ID, client ID, and the certificate or client secret")
	}

	msg := "Azure sign-in failed. No credentials were provided, so we tried to sign in using your local Azure CLI session, Azure environment variables, and a managed identity, and none of them worked. " +
		"Run `az login` and confirm `az account get-access-token` returns a token, or provide credentials directly with --tenant-id and --client-id plus a certificate or client secret"

	// azidentity's Azure CLI credential reports output that isn't a token (for
	// example a message asking you to sign in again) as a JSON decode error.
	// Match the typed decode error first, falling back to the message substring
	// in case the SDK stringifies the error and drops its type.
	var jsonSyntaxErr *json.SyntaxError
	if errors.As(err, &jsonSyntaxErr) || strings.Contains(err.Error(), "looking for beginning of value") {
		msg = "Your Azure CLI returned something other than a sign-in token, which usually means it needs you to sign in again (run `az login`) or is printing a notice. " + msg
	}

	return errors.Wrap(err, msg)
}
