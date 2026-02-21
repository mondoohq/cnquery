// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"fmt"
	"strings"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
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
			Title:   GetTitleForPlatformName(c.opts.platformOverride),
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

func GetTitleForPlatformName(name string) string {
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
		return "GCP Cloud SQL for MySQL"
	case "gcp-sql-postgresql":
		return "GCP Cloud SQL for PostgreSQL"
	case "gcp-sql-sqlserver":
		return "GCP Cloud SQL for SQL Server"
	case "gcp-dns-zone":
		return "GCP Cloud DNS Zone"
	case "gcp-kms-keyring":
		return "GCP Cloud KMS Keyring"
	case "gcp-memorystore-redis":
		return "GCP Memorystore for Redis"
	case "gcp-memorystore-rediscluster":
		return "GCP Memorystore for Redis Cluster"
	case "gcp-secretmanager-secret":
		return "GCP Secret Manager Secret"
	case "gcp-compute-instance":
		return "GCP Compute Instance"
	case "gcp-pubsub-topic":
		return "GCP Pub/Sub Topic"
	case "gcp-pubsub-subscription":
		return "GCP Pub/Sub Subscription"
	case "gcp-pubsub-snapshot":
		return "GCP Pub/Sub Snapshot"
	case "gcp-cloudrun-service":
		return "GCP Cloud Run Service"
	case "gcp-cloudrun-job":
		return "GCP Cloud Run Job"
	case "gcp-cloud-function":
		return "GCP Cloud Function"
	case "gcp-dataproc-cluster":
		return "GCP Dataproc Cluster"
	case "gcp-logging-bucket":
		return "GCP Logging Bucket"
	case "gcp-apikey":
		return "GCP API Key"
	case "gcp-iam-service-account":
		return "GCP IAM Service Account"
	}
	return "Google Cloud Platform"
}

func ParseCloudSQLType(googleType string) string {
	switch lower := strings.ToLower(googleType); {
	case lower == "postgres":
		return "postgresql"
	default:
		return lower
	}
}

func ResourceTechnologyUrl(service, project, region, objectType, name string) []string {
	switch service {
	case "compute":
		switch objectType {
		case "instance":
			return []string{"gcp", project, "compute", region, "instance", "resource"}
		case "image", "network", "subnetwork", "firewall":
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
		case "mysql", "postgresql", "sqlserver":
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
	case "memorystore":
		switch objectType {
		case "redis":
			return []string{"gcp", project, "memorystore", region, "redis"}
		case "rediscluster":
			return []string{"gcp", project, "memorystore", region, "rediscluster"}
		default:
			return []string{"gcp", project, "memorystore", region, "other"}
		}
	case "secretmanager":
		switch objectType {
		case "secret":
			return []string{"gcp", project, "secretmanager", region, "secret"}
		default:
			return []string{"gcp", project, "secretmanager", region, "other"}
		}
	case "pubsub":
		switch objectType {
		case "topic", "subscription", "snapshot":
			return []string{"gcp", project, "pubsub", region, objectType}
		default:
			return []string{"gcp", project, "pubsub", region, "other"}
		}
	case "cloudrun":
		switch objectType {
		case "service", "job":
			return []string{"gcp", project, "cloudrun", region, objectType}
		default:
			return []string{"gcp", project, "cloudrun", region, "other"}
		}
	case "cloud-functions":
		switch objectType {
		case "function":
			return []string{"gcp", project, "cloud-functions", region, "function"}
		default:
			return []string{"gcp", project, "cloud-functions", region, "other"}
		}
	case "dataproc":
		switch objectType {
		case "cluster":
			return []string{"gcp", project, "dataproc", region, "cluster"}
		default:
			return []string{"gcp", project, "dataproc", region, "other"}
		}
	case "logging":
		switch objectType {
		case "bucket":
			return []string{"gcp", project, "logging", region, "bucket"}
		default:
			return []string{"gcp", project, "logging", region, "other"}
		}
	case "apikeys":
		switch objectType {
		case "key":
			return []string{"gcp", project, "apikeys", region, "key"}
		default:
			return []string{"gcp", project, "apikeys", region, "other"}
		}
	case "iam":
		switch objectType {
		case "service-account":
			return []string{"gcp", project, "iam", region, "service-account"}
		default:
			return []string{"gcp", project, "iam", region, "other"}
		}
	default:
		return []string{"gcp", project, "other"}
	}
}
