package all

import (
	"go.mondoo.com/cnquery/resources/packs/all/info"
	"go.mondoo.com/cnquery/resources/packs/aws"
	"go.mondoo.com/cnquery/resources/packs/azure"
	"go.mondoo.com/cnquery/resources/packs/core"
	"go.mondoo.com/cnquery/resources/packs/gcp"
	"go.mondoo.com/cnquery/resources/packs/github"
	"go.mondoo.com/cnquery/resources/packs/gitlab"
	"go.mondoo.com/cnquery/resources/packs/k8s"
	"go.mondoo.com/cnquery/resources/packs/ms365"
	"go.mondoo.com/cnquery/resources/packs/okta"
	"go.mondoo.com/cnquery/resources/packs/os"
	"go.mondoo.com/cnquery/resources/packs/terraform"
	"go.mondoo.com/cnquery/resources/packs/vsphere"
)

// These functions are needed to be located here to avoid dependency cycles

// we import this from Info to fill in all the metadata first
var (
	Registry     = info.Registry
	ResourceDocs = info.ResourceDocs
)

func init() {
	Registry.Add(core.Registry)
	Registry.Add(os.Registry)
	Registry.Add(aws.Registry)
	Registry.Add(azure.Registry)
	Registry.Add(gcp.Registry)
	Registry.Add(ms365.Registry)
	Registry.Add(github.Registry)
	Registry.Add(gitlab.Registry)
	Registry.Add(terraform.Registry)
	Registry.Add(k8s.Registry)
	Registry.Add(vsphere.Registry)
	Registry.Add(okta.Registry)
}
