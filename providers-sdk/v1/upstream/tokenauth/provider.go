// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tokenauth

import (
	"fmt"
	"strings"

	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
)

// TokenProvider fetches an identity token from a cloud provider.
//
// It is an alias for upstream.TokenProvider: this package implements the
// cloud-specific providers (which pull in cloud SDKs) and registers them with
// upstream from init(), so upstream itself stays free of those dependencies.
type TokenProvider = upstream.TokenProvider

// providers maps issuer URI substrings to their TokenProvider implementation.
var providers = map[string]TokenProvider{
	"sts.amazonaws.com":                   &AWSTokenProvider{},
	"accounts.google.com":                 &GCPTokenProvider{},
	"token.actions.githubusercontent.com": &GitHubTokenProvider{},
	"login.microsoftonline.com":           &AzureTokenProvider{},
	"sts.windows.net":                     &AzureTokenProvider{},
}

// init registers Resolve with upstream so external token exchange can find the
// cloud token providers without upstream importing this package (and its cloud
// SDKs). Binaries enable token exchange by blank-importing this package.
func init() {
	upstream.RegisterTokenResolver(Resolve)
}

// Resolve returns the TokenProvider matching the given issuer URI.
func Resolve(issuerURI string) (TokenProvider, error) {
	for key, provider := range providers {
		if strings.Contains(strings.ToLower(issuerURI), key) {
			return provider, nil
		}
	}
	return nil, fmt.Errorf(
		"issuer %q not supported yet - open an issue %s or see how to exchange tokens manually %s",
		issuerURI,
		"https://github.com/mondoohq/mql/issues",
		"https://mondoo.com/docs/maintain/access/non-human/wif#exchange-tokens-manually")
}
