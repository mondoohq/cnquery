package all

import (
	"go.mondoo.io/mondoo/resources/packs/all/info"
	"go.mondoo.io/mondoo/resources/packs/aws"
	"go.mondoo.io/mondoo/resources/packs/azure"
	"go.mondoo.io/mondoo/resources/packs/core"
	"go.mondoo.io/mondoo/resources/packs/gcp"
	"go.mondoo.io/mondoo/resources/packs/os"
	"go.mondoo.io/mondoo/resources/packs/services"
)

// These functions are needed to be located here to avoid dependency cycles

// we import this from Info to fill in all the metadata first
var Registry = info.Registry
var ResourceDocs = core.ResourceDocs

func init() {
	Registry.Add(core.Registry)
	Registry.Add(os.Registry)
	Registry.Add(aws.Registry)
	Registry.Add(azure.Registry)
	Registry.Add(gcp.Registry)
	Registry.Add(services.Registry)
}
