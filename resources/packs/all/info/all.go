package info

/*

This file contains the metadata for MQL's default resource registry.
No implementation code is loaded. It is also a prerequisite to a fully
functioning registry.

*/

import (
	_ "embed"

	"go.mondoo.io/mondoo/resources"
	awsInfo "go.mondoo.io/mondoo/resources/packs/aws/info"
	azureInfo "go.mondoo.io/mondoo/resources/packs/azure/info"
	coreInfo "go.mondoo.io/mondoo/resources/packs/core/info"
	gcpInfo "go.mondoo.io/mondoo/resources/packs/gcp/info"
	osInfo "go.mondoo.io/mondoo/resources/packs/os/info"
	servicesInfo "go.mondoo.io/mondoo/resources/packs/services/info"
)

var Registry = resources.NewRegistry()

func init() {
	Registry.Add(coreInfo.Registry)
	Registry.Add(osInfo.Registry)
	Registry.Add(awsInfo.Registry)
	Registry.Add(azureInfo.Registry)
	Registry.Add(gcpInfo.Registry)
	Registry.Add(servicesInfo.Registry)
}
