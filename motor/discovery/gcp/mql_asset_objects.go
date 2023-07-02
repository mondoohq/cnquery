package gcp

import (
	"errors"
	"fmt"

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
			return gcpObjectPlatformInfo{}, errors.New(fmt.Sprintf("unknown gcp compute object type", o.objectType))
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
	return gcpObjectPlatformInfo{}, errors.New(fmt.Sprintf("missing runtime info for gcp object service %s type %s", o.service, o.objectType))
}

func computeInstances(m *MqlDiscovery, project string, tc *providers.Config, sfn common.QuerySecretFn) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type instanceDisk struct {
		GuestOsFeatures []string
	}
	type instance struct {
		Id                string
		Name              string
		Labels            map[string]string
		Zone              struct{ Name string }
		Status            string
		NetworkInterfaces []compute.NetworkInterface
		Disks             []instanceDisk
	}

	disksContainWindows := func(disks []instanceDisk) bool {
		for _, d := range disks {
			for _, f := range d.GuestOsFeatures {
				if f == "WINDOWS" {
					return true
				}
			}
		}
		return false
	}

	instances, err := GetList[instance](m, "return gcp.project.compute.instances.where( status == 'RUNNING' ) { id name labels zone { name } status networkInterfaces disks { guestOsFeatures } }")
	if err != nil {
		return nil, err
	}

	for _, i := range instances {
		if disksContainWindows(i.Disks) {
			log.Debug().Msgf("skipping windows instance %s", i.Name)
			continue
		}

		stringLabels := i.Labels
		stringLabels["mondoo.com/instance"] = i.Id

		connections := []*providers.Config{}
		for _, ni := range i.NetworkInterfaces {
			for _, ac := range ni.AccessConfigs {
				if len(ac.NatIP) > 0 {
					log.Debug().Str("instance", i.Name).Str("ip", ac.NatIP).Msg("found public ip")
					connections = append(connections, &providers.Config{
						Backend:  providers.ProviderType_SSH,
						Host:     ac.NatIP,
						Insecure: tc.Insecure,
					})
				}
			}
		}

		a := MqlObjectToAsset(
			mqlObject{
				name: i.Name, labels: stringLabels,
				gcpObject: gcpObject{
					project:    project,
					region:     i.Zone.Name,
					name:       i.Name,
					id:         i.Id,
					service:    "compute",
					objectType: "image",
				},
			}, tc)
		a.State = mapInstanceStatus(i.Status)
		a.Platform.Kind = providers.Kind_KIND_VIRTUAL_MACHINE
		a.Platform.Runtime = providers.RUNTIME_GCP_COMPUTE
		a.Connections = connections
		// find the secret reference for the asset
		common.EnrichAssetWithSecrets(a, sfn)
		assets = append(assets, a)
	}
	return assets, nil
}

func computeImages(m *MqlDiscovery, project string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type image struct {
		Id     string
		Name   string
		Labels map[string]string
	}
	images, err := GetList[image](m, "return gcp.project.compute.images { id name labels }")
	if err != nil {
		return nil, err
	}
	for _, i := range images {
		assets = append(assets, MqlObjectToAsset(
			mqlObject{
				name: i.Name, labels: i.Labels,
				gcpObject: gcpObject{
					project:    project,
					region:     "global", // Not region-based
					name:       i.Name,
					id:         i.Id,
					service:    "compute",
					objectType: "image",
				},
			}, tc))
	}
	return assets, nil
}

func computeNetworks(m *MqlDiscovery, project string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type network struct {
		Id   string
		Name string
	}
	networks, err := GetList[network](m, "return gcp.project.compute.networks { id name }")
	if err != nil {
		return nil, err
	}
	for _, n := range networks {
		assets = append(assets, MqlObjectToAsset(
			mqlObject{
				name: n.Name,
				gcpObject: gcpObject{
					project:    project,
					region:     "global", // Not region-based
					name:       n.Name,
					id:         n.Id,
					service:    "compute",
					objectType: "network",
				},
			}, tc))
	}
	return assets, nil
}

func computeSubnetworks(m *MqlDiscovery, project string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type subnetwork struct {
		Id        string
		Name      string
		RegionUrl string
	}
	subnets, err := GetList[subnetwork](m, "return gcp.project.compute.subnetworks { id name regionUrl }")
	if err != nil {
		return nil, err
	}
	for _, s := range subnets {
		region := gcp.RegionNameFromRegionUrl(s.RegionUrl)

		assets = append(assets, MqlObjectToAsset(
			mqlObject{
				name: s.Name,
				gcpObject: gcpObject{
					project:    project,
					region:     region,
					name:       s.Name,
					id:         s.Id,
					service:    "compute",
					objectType: "subnetwork",
				},
			}, tc))
	}
	return assets, nil
}

func computeFirewalls(m *MqlDiscovery, project string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type firewall struct {
		Id   string
		Name string
	}
	firewalls, err := GetList[firewall](m, "return gcp.project.compute.firewalls { id name }")
	if err != nil {
		return nil, err
	}
	for _, f := range firewalls {
		assets = append(assets, MqlObjectToAsset(
			mqlObject{
				name: f.Name,
				gcpObject: gcpObject{
					project:    project,
					region:     "global", // Not region-based
					name:       f.Name,
					id:         f.Id,
					service:    "compute",
					objectType: "firewall",
				},
			}, tc))
	}
	return assets, nil
}

func gkeClusters(m *MqlDiscovery, project string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type cluster struct {
		Id             string
		Name           string
		Location       string
		ResourceLabels map[string]string
	}
	clusters, err := GetList[cluster](m, "return gcp.project.gke.clusters { id name location resourceLabels }")
	if err != nil {
		return nil, err
	}
	for _, c := range clusters {
		assets = append(assets, MqlObjectToAsset(
			mqlObject{
				name: c.Name, labels: c.ResourceLabels,
				gcpObject: gcpObject{
					project:    project,
					region:     c.Location,
					name:       c.Name,
					id:         c.Id,
					service:    "gke",
					objectType: "cluster",
				},
			}, tc))
	}
	return assets, nil
}

func storageBuckets(m *MqlDiscovery, project string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type bucket struct {
		Id       string
		Name     string
		Location string
		Labels   map[string]string
	}
	buckets, err := GetList[bucket](m, "return gcp.project.storage.buckets { id name location labels }")
	if err != nil {
		return nil, err
	}
	for _, b := range buckets {
		assets = append(assets, MqlObjectToAsset(
			mqlObject{
				name: b.Name, labels: b.Labels,
				gcpObject: gcpObject{
					project:    project,
					region:     b.Location,
					name:       b.Name,
					id:         b.Id,
					service:    "storage",
					objectType: "bucket",
				},
			}, tc))
	}
	return assets, nil
}

func bigQueryDatasets(m *MqlDiscovery, project string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type dataset struct {
		Id       string
		Location string
		Labels   map[string]string
	}
	datasets, err := GetList[dataset](m, "return gcp.project.bigquery.datasets { id location labels }")
	if err != nil {
		return nil, err
	}
	for _, d := range datasets {
		assets = append(assets, MqlObjectToAsset(
			mqlObject{
				name: d.Id, labels: d.Labels,
				gcpObject: gcpObject{
					project:    project,
					region:     d.Location,
					name:       d.Id,
					id:         d.Id,
					service:    "bigquery",
					objectType: "dataset",
				},
			}, tc))
	}
	return assets, nil
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
