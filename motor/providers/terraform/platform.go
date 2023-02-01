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
			Family:  []string{"terraform"},
			Kind:    providers.Kind_KIND_CODE,
			Runtime: providers.RUNTIME_TERRAFORM,
		}
	case planfile:
		return &platform.Platform{
			Name:    "terraform-plan",
			Title:   "Terraform Plan",
			Family:  []string{"terraform"},
			Kind:    providers.Kind_KIND_CODE,
			Runtime: providers.RUNTIME_TERRAFORM,
		}
	default:
		return &platform.Platform{
			Name:    "terraform-hcl",
			Title:   "Terraform HCL",
			Family:  []string{"terraform"},
			Kind:    providers.Kind_KIND_CODE,
			Runtime: providers.RUNTIME_TERRAFORM,
		}
	}
}
