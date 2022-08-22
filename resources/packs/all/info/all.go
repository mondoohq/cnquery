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
	githubInfo "go.mondoo.io/mondoo/resources/packs/github/info"
	gitlabInfo "go.mondoo.io/mondoo/resources/packs/github/info"
	ms365Info "go.mondoo.io/mondoo/resources/packs/ms365"
	osInfo "go.mondoo.io/mondoo/resources/packs/os/info"
)

var Registry = resources.NewRegistry()

func init() {
	Registry.Add(coreInfo.Registry)
	Registry.Add(osInfo.Registry)
	Registry.Add(awsInfo.Registry)
	Registry.Add(azureInfo.Registry)
	Registry.Add(gcpInfo.Registry)
	Registry.Add(ms365Info.Registry)
	Registry.Add(githubInfo.Registry)
	Registry.Add(gitlabInfo.Registry)
}
