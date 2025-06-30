// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"fmt"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
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
		return NewOrganizationPlatformID(c.ResourceID()), nil
	case Project:
		return NewProjectPlatformID(c.ResourceID()), nil
	case Folder:
		return NewFolderPlatformID(c.ResourceID()), nil
	default:
		return "", fmt.Errorf("unsupported resource type %d", c.ResourceType())
	}
}

func (c *GcpConnection) ResourceType() ResourceType {
	return c.opts.resourceType
}

func (c *GcpConnection) ResourceID() string {
	return c.opts.resourceID
}

func (c *GcpConnection) PlatformInfo() (*inventory.Platform, error) {
	// TODO: this is a hack and we need to find a better way to do this
	if c.opts.platformOverride != "" && c.opts.platformOverride != "gcp" {
		return &inventory.Platform{
			Name:    c.opts.platformOverride,
			Title:   getTitleForPlatformName(c.opts.platformOverride),
			Family:  []string{"google"},
			Kind:    "gcp-object",
			Runtime: "gcp",
		}, nil
	}

	switch c.ResourceType() {
	case Organization:
		return &inventory.Platform{
			Name:    "gcp-org",
			Title:   "GCP Organization",
			Family:  []string{"google"},
			Kind:    "gcp-object",
			Runtime: "gcp",
		}, nil
	case Project:
		return &inventory.Platform{
			Name:    "gcp-project",
			Title:   "GCP Project",
			Family:  []string{"google"},
			Kind:    "gcp-object",
			Runtime: "gcp",
		}, nil
	case Folder:
		return &inventory.Platform{
			Name:    "gcp-folder",
			Title:   "GCP Folder",
			Family:  []string{"google"},
			Kind:    "gcp-object",
			Runtime: "gcp",
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
	case "gcp-sql-mysql":
		return "GCP Cloud SQL MySQL"
	case "gcp-sql-postgres":
		return "GCP Cloud SQL Postgres"
	case "gcp-sql-sqlserver":
		return "GCP Cloud SQL SQL Server"
	case "gcp-dns-zone":
		return "GCP Cloud DNS Zone"
	case "gcp-kms-keyring":
		return "GCP Cloud KMS Keyring"
	}
	return "Google Cloud Platform"
}

func ResourceTechnologyUrl(service, project, region, objectType, name string) []string {
	switch service {
	case "compute":
		switch objectType {
		case "instance":
			return []string{"gcp", project, "compute", region, "instance", "resource"}
		case "image", "network", "subnetwork":
			return []string{"gcp", project, "compute", region, objectType}
		default:
			return []string{"gcp", project, "compute", region, "other"}
		}
	case "storage":
		switch objectType {
		case "bucket":
			return []string{"gcp", project, "storage", region, objectType}
		default:
			return []string{"gcp", project, "storage", region, "other"}
		}
	case "gke":
		switch objectType {
		case "cluster":
			return []string{"gcp", project, "gke", region, objectType}
		default:
			return []string{"gcp", project, "gke", region, "other"}
		}
	case "cloud-sql":
		switch objectType {
		case "mysql", "postgres", "sqlserver":
			return []string{"gcp", project, "cloud-sql", region, objectType}
		default:
			return []string{"gcp", project, "cloud-sql", region, "other"}
		}
	case "cloud-dns":
		switch objectType {
		case "zone":
			return []string{"gcp", project, "cloud-dns", region, "zone"}
		default:
			return []string{"gcp", project, "cloud-dns", region, "other"}
		}
	case "cloud-kms":
		switch objectType {
		case "keyring":
			return []string{"gcp", project, "cloud-kms", region, "keyring"}
		default:
			return []string{"gcp", project, "cloud-kms", region, "other"}
		}
	default:
		return []string{"gcp", project, "other"}
	}
}
