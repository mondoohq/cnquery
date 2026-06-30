// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// Platforms is the static catalog of platforms this provider can emit. The
// project is the connected root asset (Kind "api"); every other entry is a
// discovered sub-asset (Kind "stackit-object"), mirroring how the aws provider
// emits "aws-object" platforms for discovered resources. All entries share the
// "stackit" family so they group together the way aws/azure assets do.
var Platforms = []*plugin.PlatformInfo{
	{
		Name:    "stackit-project",
		Title:   "STACKIT Project",
		Family:  []string{"stackit"},
		Kind:    []string{"api"},
		Runtime: []string{"stackit"},
	},
	{Name: "stackit-server", Title: "STACKIT Server", Family: []string{"stackit"}, Kind: []string{"stackit-object"}, Runtime: []string{"stackit"}},
	{Name: "stackit-ske-cluster", Title: "STACKIT Kubernetes Cluster", Family: []string{"stackit"}, Kind: []string{"stackit-object"}, Runtime: []string{"stackit"}},
	{Name: "stackit-object-storage-bucket", Title: "STACKIT Object Storage Bucket", Family: []string{"stackit"}, Kind: []string{"stackit-object"}, Runtime: []string{"stackit"}},
	{Name: "stackit-postgres-flex-instance", Title: "STACKIT PostgreSQL Flex Instance", Family: []string{"stackit"}, Kind: []string{"stackit-object"}, Runtime: []string{"stackit"}},
	{Name: "stackit-mongodb-flex-instance", Title: "STACKIT MongoDB Flex Instance", Family: []string{"stackit"}, Kind: []string{"stackit-object"}, Runtime: []string{"stackit"}},
	{Name: "stackit-sqlserver-flex-instance", Title: "STACKIT SQLServer Flex Instance", Family: []string{"stackit"}, Kind: []string{"stackit-object"}, Runtime: []string{"stackit"}},
	{Name: "stackit-opensearch-instance", Title: "STACKIT OpenSearch Instance", Family: []string{"stackit"}, Kind: []string{"stackit-object"}, Runtime: []string{"stackit"}},
	{Name: "stackit-mariadb-instance", Title: "STACKIT MariaDB Instance", Family: []string{"stackit"}, Kind: []string{"stackit-object"}, Runtime: []string{"stackit"}},
	{Name: "stackit-redis-instance", Title: "STACKIT Redis Instance", Family: []string{"stackit"}, Kind: []string{"stackit-object"}, Runtime: []string{"stackit"}},
	{Name: "stackit-rabbitmq-instance", Title: "STACKIT RabbitMQ Instance", Family: []string{"stackit"}, Kind: []string{"stackit-object"}, Runtime: []string{"stackit"}},
	{Name: "stackit-logme-instance", Title: "STACKIT LogMe Instance", Family: []string{"stackit"}, Kind: []string{"stackit-object"}, Runtime: []string{"stackit"}},
	{Name: "stackit-secrets-manager-instance", Title: "STACKIT Secrets Manager Instance", Family: []string{"stackit"}, Kind: []string{"stackit-object"}, Runtime: []string{"stackit"}},
}

var platformsByName = plugin.PlatformsByName(Platforms)

// PlatformByName returns the catalog entry for the given platform name.
func PlatformByName(name string) *plugin.PlatformInfo {
	return platformsByName[name]
}

// objectPlatformIDPrefix is the base of every discovered sub-asset's Mondoo
// platform ID. The connected project root uses PlatformIdStackitProject.
const objectPlatformIDPrefix = "//platformid.api.mondoo.app/runtime/stackit/"

// MondooObjectID builds the stable platform ID for a discovered STACKIT sub-asset,
// e.g. //platformid.api.mondoo.app/runtime/stackit/postgres-flex/<project>/<region>/<id>.
// Region-less (project-global) services use "global".
func MondooObjectID(projectID, service, region, id string) string {
	if region == "" {
		region = "global"
	}
	return objectPlatformIDPrefix + service + "/" + projectID + "/" + region + "/" + id
}

// GetPlatformForObject returns the inventory platform for a discovered sub-asset,
// applying the static catalog entry and attaching the per-asset technology URL
// segments ({"stackit", projectID, service}) that match the provider's AssetUrlTrees.
func GetPlatformForObject(platformName, projectID, service string) *inventory.Platform {
	p := &inventory.Platform{
		TechnologyUrlSegments: []string{"stackit", projectID, service},
	}
	PlatformByName(platformName).Apply(p)
	return p
}
