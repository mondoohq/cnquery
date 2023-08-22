// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"strings"

	"go.mondoo.com/cnquery/resources"
)

type assetIdentifier struct {
	name string
	id   string
}

func getAssetIdentifier(runtime *resources.Runtime) *assetIdentifier {
	a := runtime.Motor.GetAsset()
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
