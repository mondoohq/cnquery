// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
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
