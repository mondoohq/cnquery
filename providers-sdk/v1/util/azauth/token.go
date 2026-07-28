// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package azauth

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

// CredentialMethod names one source in the default sign-in chain. A caller who
// already knows how its environment hands out credentials can name the source
// instead of paying for the whole chain: probing a source that cannot work
// costs real time (the managed identity probe alone retries for up to 15
// seconds before giving up) and it is paid again for every asset in a scan.
type CredentialMethod string

const (
	// CredentialMethodCLI signs in with the local Azure CLI session (`az login`).
	CredentialMethodCLI CredentialMethod = "cli"
	// CredentialMethodEnv signs in with the AZURE_* environment variables.
	CredentialMethodEnv CredentialMethod = "env"
	// CredentialMethodManagedIdentity signs in with the managed identity
	// assigned to the Azure host we are running on.
	CredentialMethodManagedIdentity CredentialMethod = "managed-identity"
	// CredentialMethodWorkloadIdentity exchanges an OIDC token for an Entra
	// access token via a federated identity credential.
	CredentialMethodWorkloadIdentity CredentialMethod = "workload-identity"
)

// DefaultCredentialMethods is the chain we try when the caller does not name
// any method, in order.
var DefaultCredentialMethods = []CredentialMethod{
	CredentialMethodCLI,
	CredentialMethodEnv,
	CredentialMethodManagedIdentity,
	CredentialMethodWorkloadIdentity,
}

// credentialMethodDescriptions renders each method the way we talk about it in
// user-facing sign-in guidance.
var credentialMethodDescriptions = map[CredentialMethod]string{
	CredentialMethodCLI:              "your local Azure CLI session",
	CredentialMethodEnv:              "Azure environment variables",
	CredentialMethodManagedIdentity:  "a managed identity",
	CredentialMethodWorkloadIdentity: "workload identity federation",
}

// ParseCredentialMethods reads a comma-separated method list, e.g.
// "workload-identity" or "workload-identity,managed-identity". An empty value
// (as well as the explicit "default" and "auto") returns nil, meaning try every
// method. Values are normalized, so "Workload_Identity" is accepted.
//
// An unrecognized value is an error rather than a silent fallback to the full
// chain: the whole point of naming a method is to avoid the slow probing, so a
// typo that quietly reinstates it would be invisible in exactly the deployment
// that asked not to have it.
func ParseCredentialMethods(s string) ([]CredentialMethod, error) {
	var methods []CredentialMethod
	for part := range strings.SplitSeq(s, ",") {
		name := strings.ToLower(strings.TrimSpace(part))
		name = strings.ReplaceAll(name, "_", "-")
		switch name {
		case "", "default", "auto":
			continue
		}

		method := CredentialMethod(name)
		if _, ok := credentialMethodDescriptions[method]; !ok {
			return nil, fmt.Errorf("unknown Azure credential method %q, expected one of %s",
				strings.TrimSpace(part), strings.Join(credentialMethodNames(), ", "))
		}
		if !slices.Contains(methods, method) {
			methods = append(methods, method)
		}
	}
	return methods, nil
}

func credentialMethodNames() []string {
	names := make([]string, 0, len(DefaultCredentialMethods))
	for _, m := range DefaultCredentialMethods {
		names = append(names, string(m))
	}
	return names
}

// AllowsMethod reports whether m is permitted by the selection. An empty
// selection permits everything.
func AllowsMethod(methods []CredentialMethod, m CredentialMethod) bool {
	return len(methods) == 0 || slices.Contains(methods, m)
}

// ChainedTokenOptions configures the sign-in chain we fall back to when no
// explicit credential (client secret or certificate) is available.
type ChainedTokenOptions struct {
	azidentity.DefaultAzureCredentialOptions

	// ClientID of the service principal to sign in as. Workload identity needs
	// it and errors out without one; empty falls back to AZURE_CLIENT_ID. The
	// tenant comes from the embedded DefaultAzureCredentialOptions.TenantID.
	ClientID string

	// FederatedTokenFile is the path to the OIDC token that workload identity
	// federation exchanges for an access token. Empty falls back to
	// AZURE_FEDERATED_TOKEN_FILE.
	FederatedTokenFile string

	// Methods restricts the chain to these sources, tried in the given order.
	// Empty means DefaultCredentialMethods.
	Methods []CredentialMethod
}

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

// BuildChainedToken assembles the credentials that could be constructed into a
// single chain. A resolver whose constructor fails is left out, because most of
// them fail simply because that way of signing in is not configured here. When
// *every* resolver fails we surface the collected errors instead: with an
// explicit method selection that is the caller being told why the source they
// asked for is unusable, which the SDK's bare "at least one credential
// required" hides.
func BuildChainedToken(opts ...TokenResolverFn) (*azidentity.ChainedTokenCredential, error) {
	chain := []azcore.TokenCredential{}
	errs := []error{}
	for _, fn := range opts {
		cred, err := fn()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		chain = append(chain, cred)
	}
	if len(chain) == 0 && len(errs) > 0 {
		return nil, errors.Wrap(errors.Join(errs...), "no Azure credential source could be configured")
	}
	return azidentity.NewChainedTokenCredential(chain, nil)
}

func GetDefaultChainedToken(options *ChainedTokenOptions) (*azidentity.ChainedTokenCredential, error) {
	if options == nil {
		options = &ChainedTokenOptions{}
	}

	resolvers := map[CredentialMethod]TokenResolverFn{
		CredentialMethodCLI: WithCliCredentials(&azidentity.AzureCLICredentialOptions{AdditionallyAllowedTenants: []string{"*"}}),
		CredentialMethodEnv: WithEnvCredentials(&azidentity.EnvironmentCredentialOptions{ClientOptions: options.ClientOptions}),
		CredentialMethodManagedIdentity: WithRetryableManagedIdentityCredentials(5*time.Second, 3,
			&azidentity.ManagedIdentityCredentialOptions{ClientOptions: options.ClientOptions}),
		CredentialMethodWorkloadIdentity: WithWorkloadIdentityCredentials(&azidentity.WorkloadIdentityCredentialOptions{
			ClientOptions:            options.ClientOptions,
			DisableInstanceDiscovery: options.DisableInstanceDiscovery,
			TenantID:                 options.TenantID,
			ClientID:                 options.ClientID,
			TokenFilePath:            options.FederatedTokenFile,
		}),
	}

	methods := options.Methods
	if len(methods) == 0 {
		methods = DefaultCredentialMethods
	}

	// iterate the selection, not the map, so the chain keeps the caller's order
	opts := make([]TokenResolverFn, 0, len(methods))
	for _, method := range methods {
		resolver, ok := resolvers[method]
		if !ok {
			return nil, fmt.Errorf("unknown Azure credential method %q, expected one of %s",
				method, strings.Join(credentialMethodNames(), ", "))
		}
		opts = append(opts, resolver)
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

// GetTokenFromCredential builds a credential from an explicit client secret or
// certificate. When there is none, it falls back to the sign-in chain described
// by options (nil means the full default chain). tenantId and clientId are the
// authoritative values for the connection and override anything options carries.
func GetTokenFromCredential(credential *vault.Credential, tenantId, clientId string, options *ChainedTokenOptions) (azcore.TokenCredential, error) {
	var azCred azcore.TokenCredential
	var err error
	usedDefaultChain := credential == nil
	chainOpts := ChainedTokenOptions{}
	if options != nil {
		chainOpts = *options
	}
	if tenantId != "" {
		chainOpts.TenantID = tenantId
	}
	if clientId != "" {
		chainOpts.ClientID = clientId
	}
	// fallback to default authorizer if no credentials are specified
	if credential == nil {
		log.Info().Msg("no Azure credentials were provided; trying to sign in with " + describeMethods(chainOpts.Methods))
		azCred, err = GetDefaultChainedToken(&chainOpts)
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
	return &guidedCredential{inner: azCred, usedDefaultChain: usedDefaultChain, methods: chainOpts.Methods}, nil
}

// describeMethods renders a method selection as an English list, so sign-in
// guidance names what we actually tried rather than the full chain. An empty
// selection describes the default chain.
func describeMethods(methods []CredentialMethod) string {
	if len(methods) == 0 {
		methods = DefaultCredentialMethods
	}

	parts := make([]string, 0, len(methods))
	for _, m := range methods {
		desc, ok := credentialMethodDescriptions[m]
		if !ok {
			desc = string(m)
		}
		parts = append(parts, desc)
	}

	switch len(parts) {
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " or " + parts[1]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + ", or " + parts[len(parts)-1]
	}
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
	// methods records which sign-in sources the chain was restricted to, so the
	// guidance names what we actually tried. Empty means the full chain.
	methods []CredentialMethod
}

func (c *guidedCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	tk, err := c.inner.GetToken(ctx, opts)
	if err != nil {
		return azcore.AccessToken{}, enrichTokenError(err, c.usedDefaultChain, c.methods)
	}
	return tk, nil
}

// enrichTokenError wraps a sign-in failure with guidance tailored to how the
// credential was configured. The original error is always preserved so no
// diagnostic detail is lost.
func enrichTokenError(err error, usedDefaultChain bool, methods []CredentialMethod) error {
	if !usedDefaultChain {
		return errors.Wrap(err, "Azure sign-in with the provided credentials failed; double-check the tenant ID, client ID, and the certificate or client secret")
	}

	msg := "Azure sign-in failed. No credentials were provided, so we tried to sign in using " +
		describeMethods(methods) + ", and none of them worked. "
	if AllowsMethod(methods, CredentialMethodCLI) {
		msg += "Run `az login` and confirm `az account get-access-token` returns a token, or provide credentials "
	} else {
		msg += "Provide credentials "
	}
	msg += "directly with --tenant-id and --client-id plus a certificate or client secret"

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
