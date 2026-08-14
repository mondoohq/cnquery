// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// Every typed reference in this provider resolves through NewResource, which
// runs the target's init before the runtime cache is consulted and therefore
// issues one GET per reference. Okta rate limits per endpoint per minute and
// the plural resolvers fan out over id lists, so a resource-set binding with
// 200 members costs 200 GetUser calls against the same endpoint the org would
// serve as a handful of paged list requests.
//
// The helpers here look a reference up in the collection the root resource
// holds first. That collection is fetched at most once per scan and is shared
// by every reference that reads it, so resolving the same id twice, or
// resolving an id that the scan has already listed, costs nothing.
//
// A miss is never an answer. What the org serves from a list endpoint is not
// the same set as what it serves by id: the user list omits deprovisioned
// users, the application list omits Okta's internal apps, and a licensed
// feature the org does not have answers the list endpoint with 401 E0000015
// and reports nothing at all. Reading any of those as "no such object" would
// turn a call-count fix into a confident wrong answer, so an id that is absent
// from the collection, a collection that failed to read, and a collection the
// org will not serve all fall through to the NewResource path that was there
// before. The result set is unchanged; only the number of calls it takes to
// reach it goes down.

// getOkta returns the root resource, whose collections the reference lookups
// read. It is the runtime-cached singleton the query itself is rooted at, so
// this neither fetches nor allocates once the query is under way.
func getOkta(runtime *plugin.Runtime) (*mqlOkta, error) {
	res, err := CreateResource(runtime, "okta", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	root, ok := res.(*mqlOkta)
	if !ok {
		return nil, fmt.Errorf("okta root resource has unexpected type %T", res)
	}
	return root, nil
}

// oktaCollectionLock serializes reaching the root resource and reading one of
// its collections. Reference accessors are called from blocks the executor
// runs in parallel, and neither step is synchronized on its own: CreateResource
// checks the runtime cache and then writes it, so two accessors arriving at
// once can each end up holding their own root, and the field memo they read
// through (plugin.GetOrCompute) has no lock either. Between them, an unguarded
// pair of accessors lists the same collection twice and races on the field
// holding the first result.
//
// The lock is held across the read, which means it is held across the list
// request the first accessor triggers. That is safe only because no root
// collection lister resolves a reference itself; were one to start doing so it
// would deadlock here rather than fail quietly.
var oktaCollectionLock sync.Mutex

// cachedOktaCollection reads one of the root resource's collections, returning
// nil for every state that is not a readable list: an unreachable root, a list
// that errored, and a list the org would not serve. Each of those is a miss,
// and a miss falls back to the direct lookup.
func cachedOktaCollection(runtime *plugin.Runtime, read func(*mqlOkta) *plugin.TValue[[]any]) []any {
	oktaCollectionLock.Lock()
	defer oktaCollectionLock.Unlock()

	root, err := getOkta(runtime)
	if err != nil {
		return nil
	}
	return readableOktaList(read(root))
}

// readableOktaList returns the entries of a collection that was read, and nil
// for one that was not. A list that errored and a list the org reported
// nothing for are both absences rather than emptiness: answering a reference
// out of either would report "no such object" for something that was never
// looked at.
func readableOktaList(list *plugin.TValue[[]any]) []any {
	if list == nil || list.Error != nil || list.State&plugin.StateIsNull != 0 {
		return nil
	}
	return list.Data
}

// findCachedOktaResource returns the entry a reference names, or reports a miss.
// The type assertion is checked: a panic in an accessor takes down the whole
// scan rather than the one field, and a list is not worth that risk.
func findCachedOktaResource[T any](list []any, id func(T) string, want string) (T, bool) {
	var zero T
	if want == "" {
		return zero, false
	}
	for _, entry := range list {
		res, ok := entry.(T)
		if ok && id(res) == want {
			return res, true
		}
	}
	return zero, false
}

// indexCachedOktaResources keys the entries a list of references names by id,
// so a plural resolver walks the collection once rather than once per
// reference. Ids the collection does not carry are simply absent from the
// result, which is the miss the caller falls back on.
func indexCachedOktaResources[T any](list []any, id func(T) string, want []string) map[string]T {
	if len(list) == 0 || len(want) == 0 {
		return nil
	}

	wanted := make(map[string]struct{}, len(want))
	for _, w := range want {
		if w != "" {
			wanted[w] = struct{}{}
		}
	}
	if len(wanted) == 0 {
		return nil
	}

	index := make(map[string]T, len(wanted))
	for _, entry := range list {
		res, ok := entry.(T)
		if !ok {
			continue
		}
		key := id(res)
		if _, ok := wanted[key]; !ok {
			continue
		}
		index[key] = res
	}
	return index
}

// The id of a resource that failed to build is the empty string, which no
// reference asks for, so a broken entry can never be mistaken for a match.

func oktaUserID(r *mqlOktaUser) string { return r.Id.Data }

func oktaGroupID(r *mqlOktaGroup) string { return r.Id.Data }

func oktaApplicationID(r *mqlOktaApplication) string { return r.Id.Data }

func oktaCustomRoleID(r *mqlOktaCustomRole) string { return r.Id.Data }

func oktaUserTypeID(r *mqlOktaUserType) string { return r.Id.Data }

func oktaRealmID(r *mqlOktaRealm) string { return r.Id.Data }

func cachedOktaUsers(runtime *plugin.Runtime) []any {
	return cachedOktaCollection(runtime, func(root *mqlOkta) *plugin.TValue[[]any] { return root.GetUsers() })
}

func cachedOktaGroups(runtime *plugin.Runtime) []any {
	return cachedOktaCollection(runtime, func(root *mqlOkta) *plugin.TValue[[]any] { return root.GetGroups() })
}

func cachedOktaApplications(runtime *plugin.Runtime) []any {
	return cachedOktaCollection(runtime, func(root *mqlOkta) *plugin.TValue[[]any] { return root.GetApplications() })
}

func cachedOktaCustomRoles(runtime *plugin.Runtime) []any {
	return cachedOktaCollection(runtime, func(root *mqlOkta) *plugin.TValue[[]any] { return root.GetCustomRoles() })
}

func cachedOktaUserTypes(runtime *plugin.Runtime) []any {
	return cachedOktaCollection(runtime, func(root *mqlOkta) *plugin.TValue[[]any] { return root.GetUserTypes() })
}

func cachedOktaRealms(runtime *plugin.Runtime) []any {
	return cachedOktaCollection(runtime, func(root *mqlOkta) *plugin.TValue[[]any] { return root.GetRealms() })
}
