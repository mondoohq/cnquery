package info

/*

This file contains the metadata for MQL's default resource registry.
No implementation code is loaded. It is also a prerequisite to a fully
functioning registry.

*/

import (
	_ "embed"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/lr/docs"
	awsInfo "go.mondoo.com/cnquery/resources/packs/aws/info"
	azureInfo "go.mondoo.com/cnquery/resources/packs/azure/info"
	coreInfo "go.mondoo.com/cnquery/resources/packs/core/info"
	gcpInfo "go.mondoo.com/cnquery/resources/packs/gcp/info"
	githubInfo "go.mondoo.com/cnquery/resources/packs/github/info"
	gitlabInfo "go.mondoo.com/cnquery/resources/packs/gitlab/info"
	k8sInfo "go.mondoo.com/cnquery/resources/packs/k8s/info"
	ms365Info "go.mondoo.com/cnquery/resources/packs/ms365/info"
	oktaInfo "go.mondoo.com/cnquery/resources/packs/okta/info"
	osInfo "go.mondoo.com/cnquery/resources/packs/os/info"
	terraformInfo "go.mondoo.com/cnquery/resources/packs/terraform/info"
	vsphereInfo "go.mondoo.com/cnquery/resources/packs/vsphere/info"
)

var Registry = resources.NewRegistry()

// TODO: migrate the remaining manifests over
var ResourceDocs = coreInfo.ResourceDocs

func init() {
	Registry.Add(coreInfo.Registry)
	Registry.Add(osInfo.Registry)
	Registry.Add(awsInfo.Registry)
	Registry.Add(azureInfo.Registry)
	Registry.Add(gcpInfo.Registry)
	Registry.Add(ms365Info.Registry)
	Registry.Add(githubInfo.Registry)
	Registry.Add(gitlabInfo.Registry)
	Registry.Add(oktaInfo.Registry)

	ResourceDocs = mergeDocs(
		coreInfo.ResourceDocs,
		osInfo.ResourceDocs,
		awsInfo.ResourceDocs,
		azureInfo.ResourceDocs,
		gcpInfo.ResourceDocs,
		ms365Info.ResourceDocs,
		githubInfo.ResourceDocs,
		githubInfo.ResourceDocs,
		gitlabInfo.ResourceDocs,
		terraformInfo.ResourceDocs,
		k8sInfo.ResourceDocs,
		vsphereInfo.ResourceDocs,
		oktaInfo.ResourceDocs,
	)
}

func mergeDocs(rDocs ...docs.LrDocs) docs.LrDocs {
	d := docs.LrDocs{
		Resources: make(map[string]*docs.LrDocsEntry),
	}
	for _, ld := range rDocs {
		for k, r := range ld.Resources {
			d.Resources[k] = r
		}
	}
	return d
}
