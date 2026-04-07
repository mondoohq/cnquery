// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tokenauth

import (
	"context"
	"fmt"
	"strings"
)

// TokenProvider fetches an identity token from a cloud provider.
type TokenProvider interface {
	GetToken(ctx context.Context, audience string) (string, error)
}

// providers maps issuer URI substrings to their TokenProvider implementation.
var providers = map[string]TokenProvider{
	"sts.amazonaws.com":                   &AWSTokenProvider{},
	"accounts.google.com":                 &GCPTokenProvider{},
	"token.actions.githubusercontent.com": &GitHubTokenProvider{},
	"login.microsoftonline.com":           &AzureTokenProvider{},
	"sts.windows.net":                     &AzureTokenProvider{},
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
