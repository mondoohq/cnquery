// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	abstractions "github.com/microsoft/kiota-abstractions-go"
	"github.com/microsoft/kiota-abstractions-go/serialization"
	msgraphgocore "github.com/microsoftgraph/msgraph-sdk-go-core"
)

func iterate[T interface{}](ctx context.Context, res interface{}, adapter abstractions.RequestAdapter, constructorFunc serialization.ParsableFactory) ([]T, error) {
	resp := []T{}
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
