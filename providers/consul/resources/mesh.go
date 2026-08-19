// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"net/url"
	"strings"

	consulapi "github.com/hashicorp/consul/api"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/types"
)

const (
	// intentionPolicyAllow lets a connection no intention matches through.
	intentionPolicyAllow = "allow"
	// intentionPolicyDeny stops a connection no intention matches.
	intentionPolicyDeny = "deny"
	// intentionWildcard is the source or destination name matching every
	// service.
	intentionWildcard = "*"
)

// errNoAclSystem reports a grant that has no ACL system to resolve against,
// which means the resource was built outside the path that supplies one.
var errNoAclSystem = errors.New("no Consul ACL system to resolve grants against")

// defaultIntentionPolicy reports what happens to a mesh connection no intention
// matches. Consul Community Edition takes this from the ACL default policy
// rather than from the mesh configuration, so an agent with ACLs switched off,
// or switched on in "allow" mode, lets every service reach every other one.
func defaultIntentionPolicy(aclEnabled bool, aclDefaultPolicy string) string {
	if aclDefaultDeny(aclEnabled, aclDefaultPolicy) {
		return intentionPolicyDeny
	}
	return intentionPolicyAllow
}

// intentionID builds the cache key of an intention. An intention has no
// identifier of its own, and one pair of service names repeats across
// namespaces, admin partitions and cluster peers, so every one of those has to
// be part of the key or the second intention silently reports the first one's
// action. Each part is escaped, because a value carrying the separator would
// otherwise produce a key that reads as a different pairing.
func intentionID(ixn *consulapi.Intention) string {
	if ixn == nil {
		return ""
	}
	source := strings.Join([]string{
		url.PathEscape(ixn.SourcePeer),
		url.PathEscape(ixn.SourcePartition),
		url.PathEscape(ixn.SourceNS),
		url.PathEscape(ixn.SourceName),
	}, "/")
	destination := strings.Join([]string{
		url.PathEscape(ixn.DestinationPartition),
		url.PathEscape(ixn.DestinationNS),
		url.PathEscape(ixn.DestinationName),
	}, "/")
	return source + "=>" + destination
}

// intentionHasWildcard reports whether either side of the intention matches
// every service.
func intentionHasWildcard(ixn *consulapi.Intention) bool {
	if ixn == nil {
		return false
	}
	return ixn.SourceName == intentionWildcard || ixn.DestinationName == intentionWildcard
}

func (r *mqlConsulServiceMesh) intentions() ([]any, error) {
	client, err := consulClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	// The intention listing is not paged: Consul answers it out of the local
	// state store and returns every intention in the datacenter at once.
	intentions, _, err := client.Connect().Intentions(nil)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(intentions))
	for _, ixn := range intentions {
		if ixn == nil {
			continue
		}

		permissions, err := convert.JsonToDictSlice(ixn.Permissions)
		if err != nil {
			return nil, err
		}

		mqlIntention, err := CreateResource(r.MqlRuntime, "consul.mesh.intention", map[string]*llx.RawData{
			"__id":                 llx.StringData(intentionID(ixn)),
			"sourceName":           llx.StringData(ixn.SourceName),
			"sourceNamespace":      llx.StringData(ixn.SourceNS),
			"sourcePartition":      llx.StringData(ixn.SourcePartition),
			"sourcePeer":           llx.StringData(ixn.SourcePeer),
			"sourceType":           llx.StringData(string(ixn.SourceType)),
			"destinationName":      llx.StringData(ixn.DestinationName),
			"destinationNamespace": llx.StringData(ixn.DestinationNS),
			"destinationPartition": llx.StringData(ixn.DestinationPartition),
			"action":               llx.StringData(string(ixn.Action)),
			"description":          llx.StringData(ixn.Description),
			"precedence":           llx.IntData(int64(ixn.Precedence)),
			"hasWildcard":          llx.BoolData(intentionHasWildcard(ixn)),
			"hasL7Permissions":     llx.BoolData(len(ixn.Permissions) > 0),
			"permissions":          llx.ArrayData(permissions, types.Dict),
			"meta":                 llx.MapData(convert.MapToInterfaceMap(ixn.Meta), types.String),
			"createdAt":            llx.TimeDataPtr(nullableTime(ixn.CreatedAt)),
			"updatedAt":            llx.TimeDataPtr(nullableTime(ixn.UpdatedAt)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlIntention)
	}
	return res, nil
}
