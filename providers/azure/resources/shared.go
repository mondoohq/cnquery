// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/azure/connection"
)

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
