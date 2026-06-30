// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"strconv"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// Discovery targets
const (
	DiscoveryAuto         = "auto"
	DiscoveryAll          = "all"
	DiscoveryEnvironments = "environments"
	DiscoveryDocker       = "docker"
	DiscoveryKubernetes   = "kubernetes"
	DiscoveryEdge         = "edge"
)

// Connection options
const (
	OptionAddress  = "address"
	OptionInsecure = "insecure"
	// OptionEnvironmentID scopes a connection to a single managed environment
	// (endpoint) discovered from the Portainer instance.
	OptionEnvironmentID = "environment-id"
	// OptionEnvironmentType carries the Portainer endpoint type of a discovered
	// environment so the sub-asset platform can be labelled correctly.
	OptionEnvironmentType = "environment-type"
)

// EnvironmentType returns a human-readable name for a Portainer endpoint type.
// See https://github.com/portainer/portainer for the canonical enum.
func EnvironmentType(t int64) string {
	switch t {
	case 1:
		return "docker"
	case 2:
		return "agent-docker"
	case 3:
		return "azure-aci"
	case 4:
		return "edge-agent-docker"
	case 5:
		return "kubernetes"
	case 6:
		return "agent-kubernetes"
	case 7:
		return "edge-agent-kubernetes"
	default:
		return "unknown"
	}
}

// IsDockerEnvironment reports whether the endpoint type is a Docker/Swarm one.
func IsDockerEnvironment(t int64) bool {
	return t == 1 || t == 2 || t == 4
}

// IsKubernetesEnvironment reports whether the endpoint type is a Kubernetes one.
func IsKubernetesEnvironment(t int64) bool {
	return t == 5 || t == 6 || t == 7
}

// IsEdgeEnvironment reports whether the endpoint type is an Edge-agent one.
func IsEdgeEnvironment(t int64) bool {
	return t == 4 || t == 7
}

// EnvironmentStatus maps the Portainer endpoint status enum to a string.
func EnvironmentStatus(s int64) string {
	switch s {
	case 1:
		return "up"
	case 2:
		return "down"
	default:
		return "unknown"
	}
}

// AuthenticationMethod maps the Portainer settings authentication-method enum.
func AuthenticationMethod(m int64) string {
	switch m {
	case 1:
		return "internal"
	case 2:
		return "ldap"
	case 3:
		return "oauth"
	default:
		return "unknown"
	}
}

// UserRole maps the Portainer user role enum to a string.
func UserRole(r int64) string {
	switch r {
	case 1:
		return "administrator"
	case 2:
		return "standard"
	default:
		return "unknown"
	}
}

// EdgeStackDeploymentType maps the Portainer Edge stack deployment-type enum.
func EdgeStackDeploymentType(t int64) string {
	switch t {
	case 0:
		return "compose"
	case 1:
		return "kubernetes"
	default:
		return "unknown"
	}
}

// MembershipRole maps the Portainer team-membership role enum to a string.
func MembershipRole(r int64) string {
	switch r {
	case 1:
		return "leader"
	case 2:
		return "member"
	default:
		return "unknown"
	}
}

func InstancePlatform() *inventory.Platform {
	return &inventory.Platform{
		Name:                  "portainer",
		Family:                []string{"portainer"},
		Kind:                  "api",
		Runtime:               "portainer",
		Title:                 "Portainer",
		TechnologyUrlSegments: []string{"virtualization", "portainer", "instance"},
	}
}

func EnvironmentPlatform(envType int64) *inventory.Platform {
	return &inventory.Platform{
		Name:                  "portainer-environment",
		Family:                []string{"portainer"},
		Kind:                  "api",
		Runtime:               "portainer",
		Title:                 "Portainer Environment (" + EnvironmentType(envType) + ")",
		TechnologyUrlSegments: []string{"virtualization", "portainer", "environment"},
	}
}

func NewInstancePlatformID(instanceID string) string {
	return "//platformid.api.mondoo.app/runtime/portainer/instance/" + instanceID
}

func NewEnvironmentPlatformID(instanceID string, envID int64) string {
	return NewInstancePlatformID(instanceID) + "/environment/" + strconv.FormatInt(envID, 10)
}

// SubAssetPlatform returns the platform, platform id and name for a connection
// that has been scoped to a single discovered environment. When the connection
// is a plain instance connection it returns nil.
func (c *PortainerConnection) SubAssetPlatform() (*inventory.Platform, string, string) {
	envIDStr := c.Conf.Options[OptionEnvironmentID]
	if envIDStr == "" {
		return nil, "", ""
	}
	envID, err := strconv.ParseInt(envIDStr, 10, 64)
	if err != nil {
		return nil, "", ""
	}
	// The environment type is stored as a connection option at discovery time;
	// fall back to 0 ("unknown") only if it is missing or malformed.
	envType, _ := strconv.ParseInt(c.Conf.Options[OptionEnvironmentType], 10, 64)
	return EnvironmentPlatform(envType), NewEnvironmentPlatformID(c.instanceID, envID), "Portainer environment " + envIDStr
}
