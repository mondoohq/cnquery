// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"
	"sync/atomic"

	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
)

// tagsToStringMap converts a slice of AWS SDK tag structs into a
// map[string]string, using the supplied key and value accessors. The accessors
// absorb the only variation across the ~100 AWS tag shapes: the field names
// (Key/Value vs KMS-style TagKey/TagValue).
//
// Nil policy: an entry whose key is nil is skipped (a tag key is never
// legitimately empty), while a nil value is normalized to the empty string so a
// tag key present on a resource is always represented.
func tagsToStringMap[T any](tags []T, key func(T) *string, value func(T) *string) map[string]string {
	m := make(map[string]string, len(tags))
	for i := range tags {
		k := key(tags[i])
		if k == nil {
			continue
		}
		m[*k] = convert.ToValue(value(tags[i]))
	}
	return m
}

// lazyTags caches a tag lookup that costs its own API call. Most AWS services
// leave tags out of their list and describe responses, so those resources
// declare tags as a computed field and resolve it through this handler on first
// read. Embed it in the resource's Internal struct:
//
//	type mqlAwsRdsParameterGroupInternal struct {
//		lazyTags
//	}
//
//	func (a *mqlAwsRdsParameterGroup) tags() (map[string]any, error) {
//		return a.resolveTags(func() (map[string]any, error) { ... })
//	}
//
// A resource with no tags caches an empty map, so an untagged resource costs
// one call rather than one per read. Errors are not cached: a throttled or
// briefly denied call is worth retrying, and caching the failure would turn it
// into a permanent empty tag set.
type lazyTags struct {
	cacheTags   map[string]any
	tagsFetched atomic.Bool
	tagsLock    sync.Mutex
}

// resolveTags runs fetch once and caches the result. fetch returning a nil map
// is normalized to an empty one; to report tags that genuinely could not be
// read, the caller should set the field's null state instead of routing through
// here, so an unreadable tag set is never published as an authoritative empty
// one.
func (h *lazyTags) resolveTags(fetch func() (map[string]any, error)) (map[string]any, error) {
	if h.tagsFetched.Load() {
		return h.cacheTags, nil
	}
	h.tagsLock.Lock()
	defer h.tagsLock.Unlock()
	if h.tagsFetched.Load() {
		return h.cacheTags, nil
	}

	tags, err := fetch()
	if err != nil {
		return nil, err
	}
	if tags == nil {
		tags = map[string]any{}
	}

	h.cacheTags = tags
	h.tagsFetched.Store(true)
	return tags, nil
}

// tagsToMap is tagsToStringMap with a map[string]any result, the shape MQL
// expects for tag fields. It builds the map directly rather than converting a
// map[string]string, avoiding a second allocation and copy.
func tagsToMap[T any](tags []T, key func(T) *string, value func(T) *string) map[string]any {
	m := make(map[string]any, len(tags))
	for i := range tags {
		k := key(tags[i])
		if k == nil {
			continue
		}
		m[*k] = convert.ToValue(value(tags[i]))
	}
	return m
}
