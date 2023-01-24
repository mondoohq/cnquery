package gcp

import (
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/providers"
)

func getTitleFamily(o gcpObject) (gcpObjectPlatformInfo, error) {
	switch o.service {
	case "compute":
		if o.objectType == "image" {
			return gcpObjectPlatformInfo{title: "GCP Compute Image", platform: "gcp-compute-image"}, nil
		}
	case "gke":
		if o.objectType == "cluster" {
			return gcpObjectPlatformInfo{title: "GCP GKE Cluster", platform: "gcp-gke-cluster"}, nil
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

func gkeClusters(m *MqlDiscovery, project string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}
	images := m.GetList("gcp.project.gke.clusters { id name zone resourceLabels }")
	for i := range images {
		b := images[i].(map[string]interface{})
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
