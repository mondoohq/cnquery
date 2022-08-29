package terraform

import (
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
)

func (p *Provider) PlatformInfo() *platform.Platform {
	switch p.assetType {
	case statefile:
		return &platform.Platform{
			Name:    "terraform-state",
			Title:   "Terraform State",
			Kind:    providers.Kind_KIND_CODE,
			Runtime: "terraform-state",
		}
	case planfile:
		return &platform.Platform{
			Name:    "terraform-plan",
			Title:   "Terraform Plan",
			Kind:    providers.Kind_KIND_CODE,
			Runtime: "terraform-plan",
		}
	default:
		return &platform.Platform{
			Name:    "terraform",
			Title:   "Terraform",
			Kind:    providers.Kind_KIND_CODE,
			Runtime: "terraform-configuration",
		}
	}
}
