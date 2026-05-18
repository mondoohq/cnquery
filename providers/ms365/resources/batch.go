// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	abstractions "github.com/microsoft/kiota-abstractions-go"
	"github.com/microsoft/kiota-abstractions-go/serialization"
	msgraphgocore "github.com/microsoftgraph/msgraph-sdk-go-core"
)

// batchItemRequest pairs a caller-defined key with the Graph request to issue
// for it. The key is how the caller correlates a batched response back to the
// item it asked for (typically a user, group, or service principal id).
type batchItemRequest struct {
	key     string
	reqInfo *abstractions.RequestInformation
}

// batchResult holds the per-key outcome of a batched Graph request. A failure
// on a single item (e.g. a 403 on one user) is recorded in errs without
// failing the whole batch, mirroring the per-resource error handling the
// provider already uses for serial calls.
type batchResult[T serialization.Parsable] struct {
	results  map[string]T
	errs     map[string]error
	statuses map[string]int32
}

// batchGet issues every request in reqs against the Microsoft Graph $batch
// endpoint and returns, per caller key, the deserialized response of type T.
//
// Graph caps a single $batch payload at 20 sub-requests; the SDK chunks at 19
// and sends the chunks sequentially. This collapses an N-call N+1 pattern into
// ceil(N/19) HTTP round-trips. Concurrency across chunks is intentionally left
// out here (tracked separately).
//
// The adapter selects the service root, so v1.0 and beta requests must not be
// mixed in one call -- pass the matching client's adapter for each.
func batchGet[T serialization.Parsable](
	ctx context.Context,
	adapter abstractions.RequestAdapter,
	reqs []batchItemRequest,
	constructor serialization.ParsableFactory,
) (batchResult[T], error) {
	out := batchResult[T]{
		results:  make(map[string]T, len(reqs)),
		errs:     make(map[string]error, len(reqs)),
		statuses: make(map[string]int32, len(reqs)),
	}
	if len(reqs) == 0 {
		return out, nil
	}

	// NewBatchRequestCollection defaults to a 4-chunk limit; size it to the
	// actual request count so large collections (e.g. 1,000 users) are not
	// rejected.
	chunkLimit := len(reqs)/19 + 1
	collection := msgraphgocore.NewBatchRequestCollectionWithLimit(adapter, chunkLimit)

	// the SDK assigns each step a generated id; map it back to the caller key
	idToKey := make(map[string]string, len(reqs))
	for _, r := range reqs {
		item, err := collection.AddBatchRequestStep(*r.reqInfo)
		if err != nil {
			return out, err
		}
		idToKey[*item.GetId()] = r.key
	}

	resp, err := collection.Send(ctx, adapter)
	if err != nil {
		return out, transformError(err)
	}

	statusCodes := resp.GetStatusCodes()
	for id, key := range idToKey {
		if code, ok := statusCodes[id]; ok {
			out.statuses[key] = code
		}
		val, err := msgraphgocore.GetBatchResponseById[T](resp, id, constructor)
		if err != nil {
			out.errs[key] = transformError(err)
			continue
		}
		out.results[key] = val
	}
	return out, nil
}
