package gcp

import (
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/resources/packs/gcp"
)

func getTitleFamily(o gcpObject) (gcpObjectPlatformInfo, error) {
	switch o.service {
	case "compute":
		switch o.objectType {
		case "image":
			return gcpObjectPlatformInfo{title: "GCP Compute Image", platform: "gcp-compute-image"}, nil
		case "network":
			return gcpObjectPlatformInfo{title: "GCP Compute Network", platform: "gcp-compute-network"}, nil
		case "subnetwork":
			return gcpObjectPlatformInfo{title: "GCP Compute Subnetwork", platform: "gcp-compute-subnetwork"}, nil
		case "firewall":
			return gcpObjectPlatformInfo{title: "GCP Compute Firewall", platform: "gcp-compute-firewall"}, nil
		default:
			return gcpObjectPlatformInfo{}, errors.Newf("unknown gcp compute object type", o.objectType)
		}
	case "gke":
		if o.objectType == "cluster" {
			return gcpObjectPlatformInfo{title: "GCP GKE Cluster", platform: "gcp-gke-cluster"}, nil
		}
	case "storage":
		if o.objectType == "bucket" {
			return gcpObjectPlatformInfo{title: "GCP Storage Bucket", platform: "gcp-storage-bucket"}, nil
		}
	}
	return gcpObjectPlatformInfo{}, errors.Newf("missing runtime info for gcp object service %s type %s", o.service, o.objectType)
}

func computeImages(m *MqlDiscovery, project string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}
	images := m.GetList("gcp.project.compute.images { id name labels }")
	for i := range images {
		b := images[i].(map[string]interface{})
		id := b["id"].(string)
		name := b["name"].(string)
		tags := b["labels"].(map[string]interface{})
		stringLabels := make(map[string]string)
		for k, v := range tags {
			stringLabels[k] = v.(string)
		}

		assets = append(assets, MqlObjectToAsset(project,
			mqlObject{
				name: name, labels: stringLabels,
				gcpObject: gcpObject{
					project:    project,
					region:     "global", // Not region-based
					name:       name,
					id:         id,
					service:    "compute",
					objectType: "image",
				},
			}, tc))
	}
	return assets
}

func computeNetworks(m *MqlDiscovery, project string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}
	networks := m.GetList("gcp.project.compute.networks { id name }")
	for i := range networks {
		b := networks[i].(map[string]interface{})
		id := b["id"].(string)
		name := b["name"].(string)

		assets = append(assets, MqlObjectToAsset(project,
			mqlObject{
				name: name,
				gcpObject: gcpObject{
					project:    project,
					region:     "global", // Not region-based
					name:       name,
					id:         id,
					service:    "compute",
					objectType: "network",
				},
			}, tc))
	}
	return assets
}

func computeSubnetworks(m *MqlDiscovery, project string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}
	subnets := m.GetList("gcp.project.compute.subnetworks { id name regionUrl }")
	for i := range subnets {
		b := subnets[i].(map[string]interface{})
		id := b["id"].(string)
		name := b["name"].(string)
		regionUrl := b["regionUrl"].(string)
		region := gcp.RegionNameFromRegionUrl(regionUrl)

		assets = append(assets, MqlObjectToAsset(project,
			mqlObject{
				name: name,
				gcpObject: gcpObject{
					project:    project,
					region:     region,
					name:       name,
					id:         id,
					service:    "compute",
					objectType: "subnetwork",
				},
			}, tc))
	}
	return assets
}

func computeFirewalls(m *MqlDiscovery, project string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}
	firewalls := m.GetList("gcp.project.compute.firewalls { id name }")
	for i := range firewalls {
		b := firewalls[i].(map[string]interface{})
		id := b["id"].(string)
		name := b["name"].(string)

		assets = append(assets, MqlObjectToAsset(project,
			mqlObject{
				name: name,
				gcpObject: gcpObject{
					project:    project,
					region:     "global", // Not region-based
					name:       name,
					id:         id,
					service:    "compute",
					objectType: "firewall",
				},
			}, tc))
	}
	return assets
}

func gkeClusters(m *MqlDiscovery, project string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}
	clusters := m.GetList("gcp.project.gke.clusters { id name zone resourceLabels }")
	for i := range clusters {
		b := clusters[i].(map[string]interface{})
		id := b["id"].(string)
		name := b["name"].(string)
		zone := b["zone"].(string)
		tags := b["resourceLabels"].(map[string]interface{})
		stringLabels := make(map[string]string)
		for k, v := range tags {
			stringLabels[k] = v.(string)
		}

		assets = append(assets, MqlObjectToAsset(project,
			mqlObject{
				name: name, labels: stringLabels,
				gcpObject: gcpObject{
					project:    project,
					region:     zone,
					name:       name,
					id:         id,
					service:    "gke",
					objectType: "cluster",
				},
			}, tc))
	}
	return assets
}

func storageBuckets(m *MqlDiscovery, project string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}
	buckets := m.GetList("gcp.project.storage.buckets { id name location labels }")
	for i := range buckets {
		b := buckets[i].(map[string]interface{})
		id := b["id"].(string)
		name := b["name"].(string)
		location := b["location"].(string)
		tags := b["labels"].(map[string]interface{})
		stringLabels := make(map[string]string)
		for k, v := range tags {
			stringLabels[k] = v.(string)
		}

		assets = append(assets, MqlObjectToAsset(project,
			mqlObject{
				name: name, labels: stringLabels,
				gcpObject: gcpObject{
					project:    project,
					region:     location,
					name:       name,
					id:         id,
					service:    "storage",
					objectType: "bucket",
				},
			}, tc))
	}
	return assets
}
