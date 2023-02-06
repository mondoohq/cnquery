package google

import (
	"errors"

	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
)

func (t *Provider) Identifier() (string, error) {
	switch t.ResourceType() {
	case Organization:
		return "//platformid.api.mondoo.app/runtime/gcp/organizations/" + t.id, nil
	case Project:
		return "//platformid.api.mondoo.app/runtime/gcp/projects/" + t.id, nil
	case Workspace:
		return "//platformid.api.mondoo.app/runtime/googleworkspace/customer/" + t.id, nil
	default:
		return "", errors.New("unsupported resource type")
	}
}

func (p *Provider) ResourceType() ResourceType {
	return p.resourceType
}

func (p *Provider) ResourceID() string {
	return p.id
}

func (p *Provider) PlatformInfo() (*platform.Platform, error) {
	// TODO: this is a hack and we need to find a better way to do this
	if p.platformOverride != "" && p.platformOverride != "gcp" && p.platformOverride != "google-workspace" {
		return &platform.Platform{
			Name:    p.platformOverride,
			Title:   getTitleForPlatformName(p.platformOverride),
			Family:  []string{"google"},
			Kind:    providers.Kind_KIND_GCP_OBJECT,
			Runtime: providers.RUNTIME_GCP,
		}, nil
	}

	switch p.resourceType {
	case Project:
		return &platform.Platform{
			Name:    "gcp",
			Title:   "Google Cloud Platform",
			Family:  []string{"google"},
			Kind:    providers.Kind_KIND_GCP_OBJECT,
			Runtime: p.Runtime(),
		}, nil
	case Workspace:
		return &platform.Platform{
			Name:    "google-workspace",
			Title:   "Google Workspace",
			Family:  []string{"google"},
			Kind:    providers.Kind_KIND_API,
			Runtime: p.Runtime(),
		}, nil
	}

	return nil, errors.New("unsupported resource type")
}

func getTitleForPlatformName(name string) string {
	switch name {
	case "gcp-project":
		return "GCP Project"
	case "gcp-organization":
		return "GCP Organization"
	case "gcp-compute-image":
		return "GCP Compute Image"
	case "gcp-compute-network":
		return "GCP Compute Network"
	case "gcp-compute-subnetwork":
		return "GCP Compute Subnetwork"
	case "gcp-compute-firewall":
		return "GCP Compute Firewall"
	case "gcp-gke-cluster":
		return "GCP GKE Cluster"
	case "gcp-storage-bucket":
		return "GCP Storage Bucket"
	case "gcp-bigquery-dataset":
		return "GCP BigQuery Dataset"
	}
	return "Google Cloud Platform"
}
