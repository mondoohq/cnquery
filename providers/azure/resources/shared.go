// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"time"

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
