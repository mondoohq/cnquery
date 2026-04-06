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
	"login.microsoftonline.com":           &AzureTokenProvider{},
	"token.actions.githubusercontent.com": &GitHubTokenProvider{},
}

// Resolve returns the TokenProvider matching the given issuer URI.
func Resolve(issuerURI string) (TokenProvider, error) {
	for key, provider := range providers {
		if strings.Contains(issuerURI, key) {
			return provider, nil
		}
	}
	return nil, fmt.Errorf("unsupported WIF issuer URI: %s", issuerURI)
}
