package gcp

import (
	"encoding/json"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/resources/packs/gcp"
	"google.golang.org/api/compute/v1"
)

func getTitleFamily(o gcpObject) (gcpObjectPlatformInfo, error) {
	switch o.service {
	case "compute":
		switch o.objectType {
		case "instance":
			return gcpObjectPlatformInfo{title: "GCP Compute Instance", platform: "gcp-compute-instance"}, nil
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
	case "bigquery":
		if o.objectType == "dataset" {
			return gcpObjectPlatformInfo{title: "GCP BigQuery Dataset", platform: "gcp-bigquery-dataset"}, nil
		}
	}
	return gcpObjectPlatformInfo{}, errors.Newf("missing runtime info for gcp object service %s type %s", o.service, o.objectType)
}

func computeInstances(m *MqlDiscovery, project string, tc *providers.Config, sfn common.QuerySecretFn) []*asset.Asset {
	assets := []*asset.Asset{}
	instances := m.GetList("return gcp.project.compute.instances { id name labels zone status networkInterfaces }")
	for i := range instances {
		b := instances[i].(map[string]interface{})
		id := b["id"].(string)
		name := b["name"].(string)
		tags := b["labels"].(map[string]interface{})
		zone := b["zone"].(map[string]interface{})
		zoneName := zone["name"].(string)
		status := b["status"].(string)
		stringLabels := make(map[string]string)
		for k, v := range tags {
			stringLabels[k] = v.(string)
		}
		stringLabels["mondoo.com/instance"] = id

		data, err := json.Marshal(b["networkInterfaces"])
		if err != nil {
			log.Error().Msgf("failed to marshal network interfaces for gcp compute instance %s", name)
			continue
		}

		var networkIfaces []*compute.NetworkInterface
		if err := json.Unmarshal(data, &networkIfaces); err != nil {
			log.Error().Msgf("failed to unmarshal network interfaces for gcp compute instance %s", name)
			continue
		}

		connections := []*providers.Config{}
		for _, ni := range networkIfaces {
			for _, ac := range ni.AccessConfigs {
				if len(ac.NatIP) > 0 {
					log.Debug().Str("instance", name).Str("ip", ac.NatIP).Msg("found public ip")
					connections = append(connections, &providers.Config{
						Backend:  providers.ProviderType_SSH,
						Host:     ac.NatIP,
						Insecure: tc.Insecure,
					})
				}
			}
		}

		a := MqlObjectToAsset(project,
			mqlObject{
				name: name, labels: stringLabels,
				gcpObject: gcpObject{
					project:    project,
					region:     zoneName,
					name:       name,
					id:         id,
					service:    "compute",
					objectType: "image",
				},
			}, tc)
		a.State = mapInstanceStatus(status)
		a.Platform.Kind = providers.Kind_KIND_VIRTUAL_MACHINE
		a.Platform.Runtime = providers.RUNTIME_GCP_COMPUTE
		a.Connections = connections
		// find the secret reference for the asset
		common.EnrichAssetWithSecrets(a, sfn)

		if a.State == asset.State_STATE_RUNNING {
			assets = append(assets, a)
		}
	}
	return assets
}

func computeImages(m *MqlDiscovery, project string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}
	images := m.GetList("return gcp.project.compute.images { id name labels }")
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
	networks := m.GetList("return gcp.project.compute.networks { id name }")
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
	subnets := m.GetList("return gcp.project.compute.subnetworks { id name regionUrl }")
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
	firewalls := m.GetList("return gcp.project.compute.firewalls { id name }")
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
	clusters := m.GetList("return gcp.project.gke.clusters { id name location resourceLabels }")
	for i := range clusters {
		b := clusters[i].(map[string]interface{})
		id := b["id"].(string)
		name := b["name"].(string)
		zone := b["location"].(string)
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
	buckets := m.GetList("return gcp.project.storage.buckets { id name location labels }")
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

func bigQueryDatasets(m *MqlDiscovery, project string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}
	datasets := m.GetList("return gcp.project.bigquery.datasets { id location labels }")
	for i := range datasets {
		b := datasets[i].(map[string]interface{})
		id := b["id"].(string)
		name := b["id"].(string)
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
					service:    "bigquery",
					objectType: "dataset",
				},
			}, tc))
	}
	return assets
}

func mapInstanceStatus(state string) asset.State {
	switch state {
	case "RUNNING":
		return asset.State_STATE_RUNNING
	case "PROVISIONING":
		return asset.State_STATE_PENDING
	case "STAGING":
		return asset.State_STATE_PENDING
	case "STOPPED":
		return asset.State_STATE_STOPPED
	case "STOPPING":
		return asset.State_STATE_STOPPING
	case "SUSPENDED":
		return asset.State_STATE_STOPPED
	case "SUSPENDING":
		return asset.State_STATE_STOPPING
	case "TERMINATED":
		return asset.State_STATE_TERMINATED
	default:
		log.Warn().Str("state", state).Msg("unknown gcp instance state")
		return asset.State_STATE_UNKNOWN
	}
}
