// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/azure/connection"
)

// parseAzureTimestamp parses an RFC 3339 timestamp string into a *time.Time,
// returning nil when the input is nil, empty, or not valid RFC 3339. Some Azure
// SDK models expose creation timestamps as strings rather than typed time
// values (e.g. the Cognitive Services account DateCreated field).
func parseAzureTimestamp(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil
	}
	return &t
}

// isAzureNotConfigured reports whether an error means the optional feature is
// simply not configured on this resource, rather than a real failure.
//
// Azure answers 404 for sub-resources that were never created (a SQL server
// with no vulnerability-assessment storage account, a policy that does not
// apply to a database's SKU, a Defender plan that was never enabled) and 403
// when the caller holds the parent's read permission but not the narrower
// data action on the sub-resource. Neither should fail the surrounding query:
// the honest answer is a null field, not an error on every row.
//
// It deliberately does NOT match 429 or 5xx. A throttled or failing call
// proves nothing about configuration, and swallowing it would report an
// authoritative "not configured" for a resource that may well be configured.
func isAzureNotConfigured(err error) bool {
	var respErr *azcore.ResponseError
	if !errors.As(err, &respErr) {
		return false
	}
	return respErr.StatusCode == http.StatusNotFound || respErr.StatusCode == http.StatusForbidden
}

// missingResourceID reports that a resource was asked for without an id, and
// the scanned asset could not supply one either.
//
// An init in that position used to fall through with `return args, nil, nil`,
// which is not the harmless no-op it reads as: the runtime takes it as
// permission to build the resource from the args it has, so it creates one with
// no id and no fields set. Every field then crosses the plugin boundary as an
// empty DataRes and surfaces client-side as
//
//	provider returned no data and no error for a field ... field=tags id=
//	llx: encountered a primitive with no type information, coercing to null
//
// once per field, per asset, with an empty id to identify it by. An error says
// what actually happened and says it once.
//
// This is reachable whenever one of these resources is queried bare against an
// asset that is not itself that resource -- a subscription asset, for instance,
// whose platform ids carry only the //platformid.api.mondoo.app form and never
// the /subscriptions/... ARM id getAssetIdentifier looks for.
func missingResourceID(resourceName string) error {
	return fmt.Errorf(
		"%s requires an id: reach it from the list it belongs to, or scan the %s itself",
		resourceName, resourceName)
}

// notReachableDirectly reports that a sub-resource was queried by its own
// dotted name instead of through the parent that builds it.
//
// These resources are named after the path that reaches them: the resource
// `azure.subscription.aksService.cluster` plus its field `autoUpgradeProfile`
// spells a name that is itself a resource. The compiler resolves the longest
// matching resource name first (mqlc.compileResource), so writing the full
// path compiles to a bare resource with no arguments rather than a field read
// on a cluster. The runtime then builds that resource with no id and no fields
// set, and every field read on it crosses the plugin boundary as an empty
// DataRes:
//
//	provider returned no data and no error for a field ... field=upgradeChannel id=
//	llx: encountered a primitive with no type information, coercing to null
//
// once per field, per asset, with an empty id to identify it by -- while the
// query itself quietly evaluates against nulls. Failing here reports the real
// problem once and names a query that works.
//
// Only the bare path reaches this: the parent builds these through
// CreateResource, which does not run init.
func notReachableDirectly(resourceName string, example string) error {
	return fmt.Errorf(
		"%s cannot be queried on its own, it is only available through the resource that owns it: try %s",
		resourceName, example)
}

type assetIdentifier struct {
	name string
	id   string
}

func getAssetIdentifier(runtime *plugin.Runtime) *assetIdentifier {
	a := runtime.Connection.(*connection.AzureConnection).Asset()
	if a == nil {
		return nil
	}
	azureId := ""
	for _, id := range a.PlatformIds {
		if strings.HasPrefix(id, "/subscriptions/") {
			azureId = id
		}
	}
	return &assetIdentifier{name: a.Name, id: azureId}
}
