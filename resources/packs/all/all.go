package all

import (
	"go.mondoo.io/mondoo/resources/packs/all/info"
	"go.mondoo.io/mondoo/resources/packs/aws"
	"go.mondoo.io/mondoo/resources/packs/azure"
	"go.mondoo.io/mondoo/resources/packs/core"
	"go.mondoo.io/mondoo/resources/packs/gcp"
	"go.mondoo.io/mondoo/resources/packs/github"
	"go.mondoo.io/mondoo/resources/packs/gitlab"
	"go.mondoo.io/mondoo/resources/packs/ms365"
	"go.mondoo.io/mondoo/resources/packs/os"
	"go.mondoo.io/mondoo/resources/packs/terraform"
)

// These functions are needed to be located here to avoid dependency cycles

// we import this from Info to fill in all the metadata first
var (
	Registry     = info.Registry
	ResourceDocs = core.ResourceDocs
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
}
