// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	abstractions "github.com/microsoft/kiota-abstractions-go"
	"github.com/microsoft/kiota-abstractions-go/serialization"
	msgraphgocore "github.com/microsoftgraph/msgraph-sdk-go-core"
	"github.com/rs/zerolog/log"
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
		code, hasCode := statusCodes[id]
		if hasCode {
			out.statuses[key] = code
		}
		// A non-2xx sub-response carries an error body, not a T. The kiota core
		// helper GetBatchResponseById panics trying to deserialize that body as
		// T (e.g. a 403 when the app lacks AuditLog.Read.All for signInActivity).
		// Skip the parse when we already know the sub-request failed...
		if hasCode && (code < 200 || code >= 300) {
			out.errs[key] = fmt.Errorf("batch request failed with status %d", code)
			continue
		}
		// ...and recover defensively around the parse itself, since the helper
		// can still panic on malformed/error bodies the status map does not flag.
		val, err := safeGetBatchResponseByID[T](resp, id, constructor)
		if err != nil {
			out.errs[key] = transformError(err)
			continue
		}
		out.results[key] = val
	}
	return out, nil
}

// safeGetBatchResponseByID wraps msgraphgocore.GetBatchResponseById, which can
// panic (rather than return an error) when a batch sub-response holds an error
// body that does not deserialize into T -- e.g. a 403 error object parsed as a
// User. Recovering keeps one failed sub-request from crashing the whole batched
// query; the caller records it as a per-item error like any other failure.
func safeGetBatchResponseByID[T serialization.Parsable](
	resp msgraphgocore.BatchResponse,
	id string,
	constructor serialization.ParsableFactory,
) (val T, err error) {
	defer func() {
		if r := recover(); r != nil {
			var zero T
			val = zero
			err = fmt.Errorf("batch sub-response %q could not be parsed (recovered panic): %v", id, r)
			// Surface the recovered panic so it is not lost during debugging --
			// the status-code guard handles the expected non-2xx case, so a panic
			// reaching here means an unexpected/malformed body the SDK mishandled.
			log.Warn().Str("id", id).Msgf("recovered panic parsing batch sub-response: %v", r)
		}
	}()
	return msgraphgocore.GetBatchResponseById[T](resp, id, constructor)
}
