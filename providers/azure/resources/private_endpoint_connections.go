// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
)

// azurePrivateEndpointConnectionsToMql converts a slice of Azure SDK private
// endpoint connection values into shared azure.subscription.privateEndpointConnection
// resources. Every Azure SDK models private endpoint connections with the same
// JSON shape (id, name, type, properties.privateEndpoint.id,
// properties.privateLinkServiceConnectionState, properties.provisioningState),
// so the values are normalized through a dict rather than accessed per SDK type.
// This lets one helper serve resources whose SDK connection type differs
// (armservicebus.PrivateEndpointConnection, armappconfiguration.PrivateEndpointConnectionReference,
// armapimanagement.RemotePrivateEndpointConnectionWrapper, and so on).
func azurePrivateEndpointConnectionsToMql[T any](runtime *plugin.Runtime, entries []T) ([]any, error) {
	res := make([]any, 0, len(entries))
	for _, entry := range entries {
		mqlConn, err := azurePrivateEndpointConnectionToMql(runtime, entry)
		if err != nil {
			return nil, err
		}
		if mqlConn != nil {
			res = append(res, mqlConn)
		}
	}
	return res, nil
}

// azurePrivateEndpointConnectionToMql builds a single shared private endpoint
// connection resource from any Azure SDK connection value. It returns nil when
// the value carries no usable data (e.g. a nil pointer in the slice).
func azurePrivateEndpointConnectionToMql(runtime *plugin.Runtime, entry any) (plugin.Resource, error) {
	dict, err := convert.JsonToDict(entry)
	if err != nil {
		return nil, err
	}
	if len(dict) == 0 {
		return nil, nil
	}

	// A connection with no ID has no stable cache key, and everything useful
	// about it (name, navigation) derives from that ID. Skip it rather than
	// letting multiple ID-less entries collide on an empty __id.
	id, _ := dict["id"].(string)
	if id == "" {
		return nil, nil
	}
	args := map[string]*llx.RawData{
		"__id": llx.StringData(id),
		"id":   llx.StringData(id),
	}

	// Prefer the SDK-provided name; most connection types leave it empty on
	// read, so fall back to deriving it from the resource ID.
	name, _ := dict["name"].(string)
	if name == "" && id != "" {
		if rid, err := ParseResourceID(id); err == nil {
			if comp, err := rid.Component("privateEndpointConnections"); err == nil {
				name = comp
			}
		}
		if name == "" {
			parts := strings.Split(id, "/")
			name = parts[len(parts)-1]
		}
	}
	if name != "" {
		args["name"] = llx.StringData(name)
	}
	if typ, _ := dict["type"].(string); typ != "" {
		args["type"] = llx.StringData(typ)
	}

	if props, ok := dict["properties"].(map[string]any); ok && props != nil {
		args["properties"] = llx.DictData(props)

		if pe, ok := props["privateEndpoint"].(map[string]any); ok {
			if peID, _ := pe["id"].(string); peID != "" {
				args["privateEndpointId"] = llx.StringData(peID)
			}
		}
		if provState, _ := props["provisioningState"].(string); provState != "" {
			args["provisioningState"] = llx.StringData(provState)
		}
		if cs, ok := props["privateLinkServiceConnectionState"].(map[string]any); ok && cs != nil {
			// Derive a unique cache key from the parent connection so multiple
			// connection states never collide on an empty __id.
			stateArgs := map[string]*llx.RawData{
				"__id": llx.StringData(id + "/privateLinkServiceConnectionState"),
			}
			if v, _ := cs["actionsRequired"].(string); v != "" {
				stateArgs["actionsRequired"] = llx.StringData(v)
			}
			if v, _ := cs["description"].(string); v != "" {
				stateArgs["description"] = llx.StringData(v)
			}
			if v, _ := cs["status"].(string); v != "" {
				stateArgs["status"] = llx.StringData(v)
			}
			stateRes, err := CreateResource(runtime, ResourceAzureSubscriptionPrivateEndpointConnectionConnectionState, stateArgs)
			if err != nil {
				return nil, err
			}
			args["privateLinkServiceConnectionState"] = llx.ResourceData(stateRes, ResourceAzureSubscriptionPrivateEndpointConnectionConnectionState)
		}
	}

	return CreateResource(runtime, ResourceAzureSubscriptionPrivateEndpointConnection, args)
}
