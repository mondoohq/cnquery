// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package confluence

import (
	"strings"

	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
)

func (a *ConfluenceConnection) PlatformInfo() *inventory.Platform {
	return GetPlatformForObject("atlassian-confluence")
}

func GetPlatformForObject(platformName string) *inventory.Platform {
	if platformName != "atlassian-confluence" && platformName != "" {
		return &inventory.Platform{
			Name:    platformName,
			Title:   "Atlassian Confluence",
			Kind:    "api",
			Runtime: "atlassian",
		}
	}
	return &inventory.Platform{
		Name:    "atlassian-confluence",
		Title:   "Atlassian Confluence",
		Kind:    "api",
		Runtime: "atlassian",
	}
}

func (a *ConfluenceConnection) PlatformID() string {
	hostname := strings.TrimPrefix(a.name, "https://")
	host := strings.Replace(hostname, ".", "-", -1)
	return "//platformid.api.mondoo.app/runtime/atlassian/confluence/" + host
}
