// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import "go.mondoo.com/cnquery/providers-sdk/v1/inventory"

func (a *AwsConnection) PlatformInfo(name string) *inventory.Platform {
	// p.info.PlatformOverride
	return getPlatformForObject(name)
}

func getPlatformForObject(platformName string) *inventory.Platform {
	if platformName != "aws" && platformName != "" {
		return &inventory.Platform{
			Name: platformName,
			// Title:   getTitleForPlatformName(platformName),
			Kind:    "aws_object",
			Runtime: "aws",
		}
	}
	return &inventory.Platform{
		Name:    "aws",
		Title:   "Amazon Web Services",
		Kind:    "api",
		Runtime: "aws",
	}
}
