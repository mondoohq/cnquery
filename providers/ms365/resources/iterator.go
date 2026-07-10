// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"runtime"

	abstractions "github.com/microsoft/kiota-abstractions-go"
	"github.com/microsoft/kiota-abstractions-go/serialization"
	msgraphgocore "github.com/microsoftgraph/msgraph-sdk-go-core"
	"github.com/rs/zerolog/log"
)

// iterate walks every page of a Microsoft Graph collection response and returns
// the flattened list of items.
//
// It recovers from the failed type assertion raised inside the SDK's page
// iterator. The SDK's convertToPage (msgraph-sdk-go-core page_iterator.go)
// type-asserts every element of a page's `value` collection to T without a nil
// check. When Graph returns a null entry in the collection, the SDK's own
// deserializer stores it as a nil interface (see e.g.
// ServicePrincipalCollectionResponse: nil elements are left as the zero value),
// and the subsequent assertion panics with "interface conversion: interface {}
// is nil" (a *runtime.TypeAssertionError). A single malformed page must not
// crash the whole scan, so we recover from that specific panic and return
// whatever was collected before it, following the provider's "continue with
// accessible resources" error-handling convention. Any other panic is
// re-raised unchanged so genuine bugs are not silently swallowed.
func iterate[T any](ctx context.Context, res any, adapter abstractions.RequestAdapter, constructorFunc serialization.ParsableFactory) (result []T, err error) {
	resp := []T{}
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(*runtime.TypeAssertionError); !ok {
				panic(r) // not the known SDK nil-element panic; preserve normal panic semantics
			}
			log.Warn().
				Interface("recover", r).
				Int("collected", len(resp)).
				Msg("ms365> recovered from nil-element panic while paging Graph results; returning partial results")
			result = resp
			err = nil
		}
	}()

	iterator, err := msgraphgocore.NewPageIterator[T](res, adapter, constructorFunc)
	if err != nil {
		return nil, transformError(err)
	}
	err = iterator.Iterate(ctx, func(u T) bool {
		resp = append(resp, u)
		return true
	})
	if err != nil {
		return nil, transformError(err)
	}
	return resp, nil
}
