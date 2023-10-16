// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package admin

import (
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
)

func (a *AdminConnection) PlatformInfo() *inventory.Platform {
	return GetPlatformForObject("atlassian-admin")
}

func GetPlatformForObject(platformName string) *inventory.Platform {
	if platformName != "atlassian-admin" && platformName != "" {
		return &inventory.Platform{
			Name:    platformName,
			Title:   "atlassian-admin",
			Kind:    "api",
			Runtime: "atlassian",
		}
	}
	return &inventory.Platform{
		Name:    "atlassian-admin",
		Title:   "atlassian-admin",
		Kind:    "api",
		Runtime: "atlassian",
	}
}

func (a *AdminConnection) PlatformID() string {
	return "//platformid.api.mondoo.app/runtime/atlassian/admin"
}
