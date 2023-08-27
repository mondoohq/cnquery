// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/gcp/config"
	"go.mondoo.com/cnquery/providers/gcp/connection"
)

func Discover(runtime *plugin.Runtime) (*inventory.Inventory, error) {
	conn := runtime.Connection.(*connection.GcpConnection)

	in := &inventory.Inventory{Spec: &inventory.InventorySpec{
		Assets: []*inventory.Asset{},
	}}

	if conn.ResourceType() == connection.Organization {
		res, err := runtime.CreateResource(runtime, "gcp.organization", nil)
		if err != nil {
			return nil, err
		}

		gcpOrg := res.(*mqlGcpOrganization)

		for i := range conn.Conf.Discover.Targets {
			target := conn.Conf.Discover.Targets[i]
			list, err := discoverOrganization(conn, gcpOrg, target)
			if err != nil {
				return nil, err
			}
			in.Spec.Assets = append(in.Spec.Assets, list...)
		}
	} else if conn.ResourceType() == connection.Project {
		res, err := runtime.CreateResource(runtime, "gcp.project", nil)
		if err != nil {
			return nil, err
		}

		gcpProject := res.(*mqlGcpProject)

		for i := range conn.Conf.Discover.Targets {
			target := conn.Conf.Discover.Targets[i]
			list, err := discoverProject(conn, gcpProject, target)
			if err != nil {
				return nil, err
			}
			in.Spec.Assets = append(in.Spec.Assets, list...)
		}
	}

	return in, nil
}

func discoverOrganization(conn *connection.GcpConnection, gcpOrg *mqlGcpOrganization, target string) ([]*inventory.Asset, error) {
	assetList := []*inventory.Asset{}
	switch target {
	case config.DiscoveryProjects:
		projects := gcpOrg.GetProjects()
		if projects.Error != nil {
			return nil, projects.Error
		}

		projectList := projects.Data.GetList() // resolve all projects including nested
		if projectList.Error != nil {
			return nil, projectList.Error
		}

		for i := range projectList.Data {
			project := projectList.Data[i].(*mqlGcpProject)

			projectConf := conn.Conf.Clone()
			if projectConf.Options == nil {
				projectConf.Options = map[string]string{}
			}
			delete(projectConf.Options, "organization-id")
			projectConf.Options["project-id"] = project.Id.Data

			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewProjectPlatformID(project.Id.Data),
				},
				Name: project.Name.Data,
				Platform: &inventory.Platform{
					Name:  "gcp-project",
					Title: "GCP Project " + project.Name.Data,
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{projectConf}, // pass-in the parent connection config
			})
		}
	}
	return assetList, nil
}

func discoverProject(conn *connection.GcpConnection, gcpProject *mqlGcpProject, target string) ([]*inventory.Asset, error) {
	assetList := []*inventory.Asset{}
	switch target {
	case config.DiscoveryInstances:
		compute := gcpProject.GetCompute()
		if compute.Error != nil {
			return nil, compute.Error
		}
		instances := compute.Data.GetInstances()
		if instances != nil {
			return nil, instances.Error
		}

		for i := range instances.Data {
			instance := instances.Data[i].(*mqlGcpProjectComputeServiceInstance)
			status := instance.GetStatus()
			if status.Data != "RUNNING" {
				continue
			}

			labels := map[string]string{}
			labels["mondoo.com/instance"] = instance.Id.Data
			instancelabels := instance.GetLabels()
			for k, v := range instancelabels.Data {
				labels[k] = v.(string)
			}

			zone := instance.GetZone()
			if zone.Error != nil {
				return nil, zone.Error
			}

			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("compute", gcpProject.Id.Data, zone.Data.Name.Data, "instance", instance.Name.Data),
				},
				Name: instance.Name.Data,
				Platform: &inventory.Platform{
					Name:  "gcp-compute-instance",
					Title: "GCP Compute Instance",
				},
				Labels: labels,
				// TODO: the current connection handling does not work well for instances
				Connections: []*inventory.Config{conn.Conf.Clone()}, // pass-in the parent connection config
			})
		}

	case config.DiscoveryComputeImages:
		compute := gcpProject.GetCompute()
		if compute.Error != nil {
			return nil, compute.Error
		}
		images := compute.Data.GetImages()
		if images != nil {
			return nil, images.Error
		}

		for i := range images.Data {
			image := images.Data[i].(*mqlGcpProjectComputeServiceImage)
			labels := map[string]string{}
			for k, v := range image.GetLabels().Data {
				labels[k] = v.(string)
			}
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("compute", gcpProject.Id.Data, "global", "image", image.Name.Data),
				},
				Name: image.Name.Data,
				Platform: &inventory.Platform{
					Name:  "gcp-compute-image",
					Title: "GCP Compute Image",
				},
				Labels:      labels,
				Connections: []*inventory.Config{conn.Conf.Clone()}, // pass-in the parent connection config
			})
		}
	case config.DiscoveryComputeNetworks:
		compute := gcpProject.GetCompute()
		if compute.Error != nil {
			return nil, compute.Error
		}
		networks := compute.Data.GetNetworks()
		if networks != nil {
			return nil, networks.Error
		}
		for i := range networks.Data {
			network := networks.Data[i].(*mqlGcpProjectComputeServiceNetwork)
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("compute", gcpProject.Id.Data, "global", "network", network.Name.Data),
				},
				Name: network.Name.Data,
				Platform: &inventory.Platform{
					Name:  "gcp-compute-network",
					Title: "GCP Compute Network",
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{conn.Conf.Clone()}, // pass-in the parent connection config
			})
		}
	case config.DiscoveryComputeSubnetworks:
		compute := gcpProject.GetCompute()
		if compute.Error != nil {
			return nil, compute.Error
		}
		networks := compute.Data.GetSubnetworks()
		if networks != nil {
			return nil, networks.Error
		}
		for i := range networks.Data {
			network := networks.Data[i].(*mqlGcpProjectComputeServiceSubnetwork)
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("compute", gcpProject.Id.Data, "global", "subnetwork", network.Name.Data),
				},
				Name: network.Name.Data,
				Platform: &inventory.Platform{
					Name:  "gcp-compute-subnetwork",
					Title: "GCP Compute Subnetwork",
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{conn.Conf.Clone()}, // pass-in the parent connection config
			})
		}
	case config.DiscoveryComputeFirewalls:
		compute := gcpProject.GetCompute()
		if compute.Error != nil {
			return nil, compute.Error
		}
		firewalls := compute.Data.GetFirewalls()
		if firewalls != nil {
			return nil, firewalls.Error
		}
		for i := range firewalls.Data {
			firewall := firewalls.Data[i].(*mqlGcpProjectComputeServiceFirewall)
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("compute", gcpProject.Id.Data, "global", "firewall", firewall.Name.Data),
				},
				Name: firewall.Name.Data,
				Platform: &inventory.Platform{
					Name:  "gcp-compute-firewall",
					Title: "GCP Compute Firewall",
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{conn.Conf.Clone()}, // pass-in the parent connection config
			})
		}
	case config.DiscoveryGkeClusters:
		gke := gcpProject.GetGke()
		if gke.Error != nil {
			return nil, gke.Error
		}
		clusters := gke.Data.GetClusters()
		if clusters != nil {
			return nil, clusters.Error
		}
		for i := range clusters.Data {
			cluster := clusters.Data[i].(*mqlGcpProjectGkeServiceCluster)
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("gke", gcpProject.Id.Data, cluster.GetLocation().Data, "cluster", cluster.Name.Data),
				},
				Name: cluster.Name.Data,
				Platform: &inventory.Platform{
					Name:  "gcp-container-cluster",
					Title: "GCP Container Cluster",
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{conn.Conf.Clone()}, // pass-in the parent connection config
			})
		}
	case config.DiscoveryStorageBuckets:
		storage := gcpProject.GetStorage()
		if storage.Error != nil {
			return nil, storage.Error
		}
		buckets := storage.Data.GetBuckets()
		if buckets != nil {
			return nil, buckets.Error
		}
		for i := range buckets.Data {
			bucket := buckets.Data[i].(*mqlGcpProjectStorageServiceBucket)
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("storage", gcpProject.Id.Data, bucket.GetLocation().Data, "bucket", bucket.Name.Data),
				},
				Name: bucket.Name.Data,
				Platform: &inventory.Platform{
					Name:  "gcp-storage-bucket",
					Title: "GCP Storage Bucket",
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{conn.Conf.Clone()}, // pass-in the parent connection config
			})
		}
	case config.DiscoveryBigQueryDatasets:
		bq := gcpProject.GetBigquery()
		if bq.Error != nil {
			return nil, bq.Error
		}
		datasets := bq.Data.GetDatasets()
		if datasets != nil {
			return nil, datasets.Error
		}
		for i := range datasets.Data {
			dataset := datasets.Data[i].(*mqlGcpProjectBigqueryServiceDataset)
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("bigquery", gcpProject.Id.Data, dataset.GetLocation().Data, "dataset", dataset.Name.Data),
				},
				Name: dataset.Name.Data,
				Platform: &inventory.Platform{
					Name:  "gcp-bigquery-dataset",
					Title: "GCP BigQuery Dataset",
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{conn.Conf.Clone()}, // pass-in the parent connection config
			})
		}
	}
	return assetList, nil
}
