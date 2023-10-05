// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"fmt"

	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
)

func NewOrganizationPlatformID(id string) string {
	return "//platformid.api.mondoo.app/runtime/gcp/organizations/" + id
}

func NewProjectPlatformID(id string) string {
	return "//platformid.api.mondoo.app/runtime/gcp/projects/" + id
}

func NewFolderPlatformID(id string) string {
	return "//platformid.api.mondoo.app/runtime/gcp/folders/" + id
}

func NewResourcePlatformID(service, project, region, objectType, name string) string {
	return "//platformid.api.mondoo.app/runtime/gcp/" + service + "/v1/projects/" + project + "/regions/" + region + "/" + objectType + "/" + name
}

func (c *GcpConnection) Identifier() (string, error) {
	switch c.ResourceType() {
	case Organization:
		return NewOrganizationPlatformID(c.resourceID), nil
	case Project:
		return NewProjectPlatformID(c.resourceID), nil
	case Folder:
		return NewFolderPlatformID(c.resourceID), nil
	default:
		return "", fmt.Errorf("unsupported resource type %d", c.ResourceType())
	}
}

func (c *GcpConnection) ResourceType() ResourceType {
	return c.resourceType
}

func (c *GcpConnection) ResourceID() string {
	return c.resourceID
}

func (c *GcpConnection) PlatformInfo() (*inventory.Platform, error) {
	// TODO: this is a hack and we need to find a better way to do this
	if c.platformOverride != "" && c.platformOverride != "gcp" {
		return &inventory.Platform{
			Name:   c.platformOverride,
			Title:  getTitleForPlatformName(c.platformOverride),
			Family: []string{"google"},
			// Kind:    providers.Kind_KIND_GCP_OBJECT,
			// Runtime: providers.RUNTIME_GCP,
		}, nil
	}

	switch c.resourceType {
	case Organization:
		return &inventory.Platform{
			Name:   "gcp-org",
			Title:  "GCP Organization",
			Family: []string{"google"},
			// Kind:    providers.Kind_KIND_GCP_OBJECT,
			// Runtime: p.Runtime(),
		}, nil
	case Project:
		return &inventory.Platform{
			Name:   "gcp-project",
			Title:  "GCP Project",
			Family: []string{"google"},
			// Kind:    providers.Kind_KIND_GCP_OBJECT,
			// Runtime: p.Runtime(),
		}, nil
	case Folder:
		return &inventory.Platform{
			Name:   "gcp-folder",
			Title:  "GCP Folder",
			Family: []string{"google"},
			// Kind:    providers.Kind_KIND_GCP_OBJECT,
			// Runtime: p.Runtime(),
		}, nil
	}

	return nil, errors.New("unsupported resource type")
}

func getTitleForPlatformName(name string) string {
	switch name {
	case "gcp-organization":
		return "GCP Organization"
	case "gcp-folder":
		return "GCP Folder"
	case "gcp-project":
		return "GCP Project"
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
