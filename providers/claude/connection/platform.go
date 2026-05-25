// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import "go.mondoo.com/mql/v13/providers-sdk/v1/inventory"

const (
	DiscoveryAll        = "all"
	DiscoveryAuto       = "auto"
	DiscoveryOrg        = "organization"
	DiscoveryWorkspaces = "workspaces"
)

var (
	PlatformIdOrg       = "//platformid.api.mondoo.app/runtime/claude/organization/"
	PlatformIdWorkspace = "//platformid.api.mondoo.app/runtime/claude/workspace/"
)

func NewClaudeOrgPlatform(orgID string) *inventory.Platform {
	return &inventory.Platform{
		Name:                  "claude-organization",
		Title:                 "Claude Organization",
		Family:                []string{"claude"},
		Kind:                  "api",
		Runtime:               "claude",
		TechnologyUrlSegments: []string{"ai", "claude", "organization", orgID},
	}
}

func NewClaudeWorkspacePlatform(orgID, workspaceID string) *inventory.Platform {
	return &inventory.Platform{
		Name:                  "claude-workspace",
		Title:                 "Claude Workspace",
		Family:                []string{"claude"},
		Kind:                  "api",
		Runtime:               "claude",
		TechnologyUrlSegments: []string{"ai", "claude", "organization", orgID, "workspace", workspaceID},
	}
}

func NewClaudeAPIPlatform(host string) *inventory.Platform {
	return &inventory.Platform{
		Name:                  "claude",
		Title:                 "Claude",
		Family:                []string{"claude"},
		Kind:                  "api",
		Runtime:               "claude",
		TechnologyUrlSegments: []string{"ai", "claude", host},
	}
}

func NewClaudeOrgIdentifier(orgID string) string {
	return PlatformIdOrg + orgID
}

func NewClaudeWorkspaceIdentifier(workspaceID string) string {
	return PlatformIdWorkspace + workspaceID
}
