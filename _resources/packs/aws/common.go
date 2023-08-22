// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package aws

import (
	"strings"

	"go.mondoo.com/cnquery/resources"
)

type assetIdentifier struct {
	name string
	arn  string
}

func getAssetIdentifier(runtime *resources.Runtime) *assetIdentifier {
	a := runtime.Motor.GetAsset()
	if a == nil {
		return nil
	}
	arn := ""
	for _, id := range a.PlatformIds {
		if strings.HasPrefix(id, "arn:aws:") {
			arn = id
		}
	}
	return &assetIdentifier{name: a.Name, arn: arn}
}
