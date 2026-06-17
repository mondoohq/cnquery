// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package upstream

import (
	"context"
	"sync"

	"github.com/cockroachdb/errors"
)

// TokenProvider fetches an identity token from a cloud provider, used by
// external token exchange (workload identity federation).
type TokenProvider interface {
	GetToken(ctx context.Context, audience string) (string, error)
}

// TokenResolver returns the TokenProvider matching a given issuer URI.
type TokenResolver func(issuerURI string) (TokenProvider, error)

var (
	tokenResolverMu sync.RWMutex
	tokenResolver   TokenResolver
)

// RegisterTokenResolver installs the resolver used by external token exchange.
//
// The cloud token providers depend on cloud SDKs (e.g. the AWS SigV4 signer),
// so they live in a separate implementation package that registers itself from
// init(). Blank-import go.mondoo.com/mql/v13/providers-sdk/v1/upstream/tokenauth
// to enable external token exchange. Keeping the registration in that package
// keeps the cloud SDKs out of the import graph of this package — and therefore
// out of inventory, which embeds upstream's protobuf types and is imported by
// any consumer of the llx.Runtime interface.
func RegisterTokenResolver(r TokenResolver) {
	tokenResolverMu.Lock()
	defer tokenResolverMu.Unlock()
	tokenResolver = r
}

// resolveTokenProvider returns the registered TokenProvider for the issuer, or
// an error explaining how to enable external token exchange.
func resolveTokenProvider(issuerURI string) (TokenProvider, error) {
	tokenResolverMu.RLock()
	r := tokenResolver
	tokenResolverMu.RUnlock()
	if r == nil {
		return nil, errors.New("external token exchange is unavailable: blank-import go.mondoo.com/mql/v13/providers-sdk/v1/upstream/tokenauth to enable it")
	}
	return r(issuerURI)
}
