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

// orZero returns p when it is set, and a pointer to the zero value of T
// otherwise, so a caller can read fields off it without a nil check.
//
// ARM models almost every nested block as an optional pointer, including the
// `properties` of a list row. Reading entry.Properties.Field directly panics on
// a row that omits it -- and a panic in a provider accessor is unrecoverable:
// the executor runs blocks in goroutines, so it takes down the whole scan
// rather than the one query. Going through this helper keeps the row in the
// result with its fields reading null, which is the honest answer, instead of
// dropping it or crashing.
func orZero[T any](p *T) *T {
	if p != nil {
		return p
	}
	return new(T)
}

// subResourceCacheID builds the cache key for a sub-resource that ARM normally
// identifies by its own resource id.
//
// A resource created with neither an explicit "__id" argument nor an id()
// method gets the empty cache key. CreateResource returns the cached occupant
// of a key it has already seen, so every such instance in the scan aliases to
// the first one created and the collection reports one row's data N times. The
// failure is silent: the list has the right length, every entry has the wrong
// contents.
//
// armID is preferred because it is already parent-qualified. When the service
// omits it, parentID+collection+name reproduces the same shape from values the
// caller always has, so an absent id degrades to a longer key rather than to a
// shared one.
func subResourceCacheID(armID *string, parentID, collection, name string) string {
	if armID != nil && *armID != "" {
		return *armID
	}
	return parentID + "/" + collection + "/" + name
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
