// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jira

import (
	"strings"

	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
)

func (a *JiraConnection) PlatformInfo() *inventory.Platform {
	return GetPlatformForObject("atlassian-jira")
}

func GetPlatformForObject(platformName string) *inventory.Platform {
	if platformName != "atlassian-jira" && platformName != "" {
		return &inventory.Platform{
			Name:    platformName,
			Title:   "atlassian-jira",
			Kind:    "api",
			Runtime: "atlassian",
		}
	}
	return &inventory.Platform{
		Name:    "atlassian-jira",
		Title:   "atlassian-jira",
		Kind:    "api",
		Runtime: "atlassian",
	}
}

func (a *JiraConnection) PlatformID() string {
	hostname := strings.TrimPrefix(a.host, "https://")
	host := strings.Replace(hostname, ".", "-", -1)
	return "//platformid.api.mondoo.app/runtime/atlassian/jira/" + host
}
