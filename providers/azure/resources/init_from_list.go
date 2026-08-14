// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/azure/connection"
)

// azureListedResource is any resource that a service-level list returns and
// that carries the ARM id its asset is discovered under.
type azureListedResource interface {
	plugin.Resource
	GetId() *plugin.TValue[string]
}

// azureListService is any of the azure.subscription.*Service resources, all of
// which are keyed on the subscription alone.
type azureListService interface {
	plugin.Resource
}

// initFromServiceList resolves a resource by its ARM id out of the list its
// parent service has already fetched, instead of asking ARM for that one
// resource.
//
// Every discovered asset resolves its own resource through its init, once per
// asset, so a per-resource Get costs one API call per asset of that type: a
// subscription with 150 VMs pays 150 of them. The service's list is a single
// call that already contains every one of those resources, and during a scan it
// has usually been fetched already, because discovery itself walks the same
// list to find the assets. Worse, NewResource runs the init *before* it
// consults the resource cache, so a Get here is spent even when the resource is
// already built, and its result is then thrown away in favour of the cached
// one.
//
// This mirrors the k8s provider's initNamespacedResource.
//
// The trade is deliberate: resolving a single asset in isolation now pulls the
// whole subscription's list rather than one record. During a scan that list is
// wanted anyway and is very likely already cached; for a one-off query against
// one asset it is the more expensive path.
func initFromServiceList[S azureListService](
	runtime *plugin.Runtime,
	args map[string]*llx.RawData,
	serviceName string,
	entries func(S) *plugin.TValue[[]any],
	resourceName string,
) (map[string]*llx.RawData, plugin.Resource, error) {
	// the args are already complete; nothing to resolve
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil && ids.id != "" {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, missingResourceID(resourceName)
	}
	id, ok := args["id"].Value.(string)
	if !ok {
		return nil, nil, errors.New("id must be a non-nil string value")
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}

	res, err := NewResource(runtime, serviceName, map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	svc, ok := res.(S)
	if !ok {
		return nil, nil, fmt.Errorf("unexpected resource type for %s", serviceName)
	}

	list := entries(svc)
	if list.Error != nil {
		return nil, nil, list.Error
	}
	for i := range list.Data {
		item, ok := list.Data[i].(azureListedResource)
		if !ok {
			continue
		}
		if item.GetId().Data == id {
			return args, item, nil
		}
	}

	// Deliberately an error rather than falling through with (args, nil, nil):
	// that would have the runtime build the resource from the id alone, leaving
	// every other field unset rather than null, which reaches the client as an
	// untyped null with nothing pointing at the cause.
	return nil, nil, fmt.Errorf("%s with id %q not found", resourceName, id)
}
