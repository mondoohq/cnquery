// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/bitwarden/connection"
)

// mqlBitwardenCollectionInternal caches the group IDs embedded in the
// collection's Public API response, so groups() can resolve typed
// references without an extra list call.
type mqlBitwardenCollectionInternal struct {
	cacheGroupIds []string
}

// newMqlBitwardenCollection maps a single Public API collection to its MQL
// resource.
func newMqlBitwardenCollection(runtime *plugin.Runtime, c connection.Collection) (plugin.Resource, error) {
	res, err := CreateResource(runtime, "bitwarden.collection", map[string]*llx.RawData{
		"__id":       llx.StringData(c.Id),
		"id":         llx.StringData(c.Id),
		"name":       llx.StringData(c.Name),
		"externalId": llx.StringDataPtr(c.ExternalId),
	})
	if err != nil {
		return nil, err
	}
	mqlCollection := res.(*mqlBitwardenCollection)
	mqlCollection.cacheGroupIds = collectionIds(c.Groups)
	return mqlCollection, nil
}

// initBitwardenCollection resolves a collection by its ID on demand, for
// typed references (e.g. bitwarden.member.collections,
// bitwarden.group.collections) and direct lookups.
func initBitwardenCollection(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// fast path: the caller already provided a fully populated resource
	if len(args) > 2 {
		return args, nil, nil
	}

	idArg, ok := args["id"]
	if !ok {
		return nil, nil, errors.New("bitwarden.collection requires an id argument")
	}
	id, ok := idArg.Value.(string)
	if !ok || id == "" {
		return nil, nil, errors.New("bitwarden.collection requires a valid id")
	}

	conn := runtime.Connection.(*connection.BitwardenConnection)
	c, err := conn.Client().GetCollection(context.Background(), id)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "bitwarden.collection with id %q not found", id)
	}

	res, err := newMqlBitwardenCollection(runtime, *c)
	if err != nil {
		return nil, nil, err
	}
	return args, res, nil
}

// groups resolves the groups granted access to this collection, as typed
// bitwarden.group references. The collection's own Public API response
// already embeds the group IDs, so this never issues an extra list call.
func (c *mqlBitwardenCollection) groups() ([]any, error) {
	if len(c.cacheGroupIds) == 0 {
		return nil, nil
	}

	var all []any
	for _, id := range c.cacheGroupIds {
		r, err := NewResource(c.MqlRuntime, "bitwarden.group", map[string]*llx.RawData{"id": llx.StringData(id)})
		if err != nil {
			log.Warn().Err(err).Str("id", id).Msg("bitwarden: failed to resolve group")
			continue
		}
		all = append(all, r)
	}
	return all, nil
}

// members resolves the members granted explicit access to this collection,
// as typed bitwarden.member references. The Public API has no dedicated
// "members of a collection" endpoint; member-to-collection access is only
// exposed on the member record, so this lists every member once and filters
// in memory (Pattern C) rather than making a per-member call.
func (c *mqlBitwardenCollection) members() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.BitwardenConnection)

	members, err := conn.Client().ListMembers(context.Background())
	if err != nil {
		return nil, err
	}

	var all []any
	for _, m := range members {
		if !hasCollectionId(m.Collections, c.Id.Data) {
			continue
		}
		r, err := newMqlBitwardenMember(c.MqlRuntime, m)
		if err != nil {
			return nil, err
		}
		all = append(all, r)
	}
	return all, nil
}

// hasCollectionId reports whether sel contains an entry for the given
// collection ID.
func hasCollectionId(sel []connection.SelectionReadOnly, id string) bool {
	for _, s := range sel {
		if s.Id == id {
			return true
		}
	}
	return false
}
