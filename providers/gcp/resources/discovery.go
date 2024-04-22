// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/gcp/connection"
	"go.mondoo.com/cnquery/v11/utils/stringx"
	"google.golang.org/api/cloudresourcemanager/v3"
)

const (
	// Discovery flags
	DiscoveryAuto               = "auto"
	DiscoveryAll                = "all"
	DiscoveryOrganization       = "organization"
	DiscoveryFolders            = "folders"
	DiscoveryInstances          = "instances"
	DiscoveryProjects           = "projects"
	DiscoveryComputeImages      = "compute-images"
	DiscoveryComputeNetworks    = "compute-networks"
	DiscoveryComputeSubnetworks = "compute-subnetworks"
	DiscoveryComputeFirewalls   = "compute-firewalls"
	DiscoveryGkeClusters        = "gke-clusters"
	DiscoveryStorageBuckets     = "storage-buckets"
	DiscoveryBigQueryDatasets   = "bigquery-datasets"
)

func Discover(runtime *plugin.Runtime, features cnquery.Features) (*inventory.Inventory, error) {
	conn := runtime.Connection.(*connection.GcpConnection)

	in := &inventory.Inventory{Spec: &inventory.InventorySpec{
		Assets: []*inventory.Asset{},
	}}

	if features.IsActive(cnquery.FineGrainedCloudAssets) {
		conn.Conf.Discover.Targets = append(conn.Conf.Discover.Targets, DiscoveryAll)
	}

	if conn.ResourceType() == connection.Organization {
		res, err := NewResource(runtime, "gcp.organization", nil)
		if err != nil {
			return nil, err
		}

		gcpOrg := res.(*mqlGcpOrganization)

		list, err := discoverOrganization(conn, gcpOrg)
		if err != nil {
			return nil, err
		}
		in.Spec.Assets = append(in.Spec.Assets, list...)
	} else if conn.ResourceType() == connection.Folder {
		res, err := NewResource(runtime, "gcp.folder", nil)
		if err != nil {
			return nil, err
		}

		gcpFolder := res.(*mqlGcpFolder)
		if stringx.ContainsAnyOf(conn.Conf.Discover.Targets, DiscoveryAll, DiscoveryAuto, DiscoveryFolders) {
			in.Spec.Assets = append(in.Spec.Assets, &inventory.Asset{
				PlatformIds: []string{
					connection.NewFolderPlatformID(gcpFolder.Id.Data),
				},
				Name: "GCP Folder " + gcpFolder.Id.Data,
				Platform: &inventory.Platform{
					Name:    "gcp-folder",
					Title:   "GCP Folder",
					Runtime: "gcp",
					Kind:    "gcp-object",
					Family:  []string{"google"},
				},
				Labels: map[string]string{},
				// NOTE: we explicitly do not exclude discovery here, as we want to discover the projects for the folder
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithParentConnectionId(conn.Conf.Id))},
			})
		}

		list, err := discoverFolder(conn, gcpFolder)
		if err != nil {
			return nil, err
		}
		if len(in.Spec.Assets) > 0 {
			in.Spec.Assets[0].RelatedAssets = list
		}
		in.Spec.Assets = append(in.Spec.Assets, list...)
	} else if conn.ResourceType() == connection.Project {
		res, err := NewResource(runtime, "gcp.project", nil)
		if err != nil {
			return nil, err
		}

		gcpProject := res.(*mqlGcpProject)
		if stringx.ContainsAnyOf(conn.Conf.Discover.Targets, DiscoveryAll, DiscoveryAuto, DiscoveryProjects) {
			in.Spec.Assets = append(in.Spec.Assets, &inventory.Asset{
				PlatformIds: []string{
					connection.NewProjectPlatformID(gcpProject.Id.Data),
				},
				Name: "GCP Project " + gcpProject.Id.Data,
				Platform: &inventory.Platform{
					Name:                  "gcp-project",
					Title:                 "GCP Project",
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: []string{"gcp", gcpProject.Id.Data, "project"},
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))},
			})
		}

		list, err := discoverProject(conn, gcpProject)
		if err != nil {
			return nil, err
		}
		if len(in.Spec.Assets) > 0 {
			in.Spec.Assets[0].RelatedAssets = list
		}
		in.Spec.Assets = append(in.Spec.Assets, list...)
	} else if conn.ResourceType() == connection.Gcr {
		conf := conn.Conf
		repository := "gcr.io/" + conf.Options["project-id"]
		if conf.Options["repository"] != "" {
			repository += "/" + conf.Options["repository"]
		}
		conf.Host = repository

		assets, err := resolveGcr(context.Background(), conf)
		if err != nil {
			return nil, err
		}
		in.Spec.Assets = append(in.Spec.Assets, assets...)
		// FIXME: This is a workaround to not double-resolve the GCR repository
		conn.Conf.Discover = nil
	}

	return in, nil
}

func discoverOrganization(conn *connection.GcpConnection, gcpOrg *mqlGcpOrganization) ([]*inventory.Asset, error) {
	assetList := []*inventory.Asset{}
	if stringx.ContainsAnyOf(conn.Conf.Discover.Targets, DiscoveryAll, DiscoveryAuto, DiscoveryProjects) {
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

			projectConf := conn.Conf.Clone(inventory.WithParentConnectionId(conn.Conf.Id))
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
					Name:                  "gcp-project",
					Title:                 "GCP Project " + project.Name.Data,
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: []string{"gcp", project.Id.Data, "project"},
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{projectConf}, // pass-in the parent connection config
			})
		}
	}
	if stringx.ContainsAnyOf(conn.Conf.Discover.Targets, DiscoveryAll, DiscoveryAuto, DiscoveryFolders) {
		folders := gcpOrg.GetFolders()
		if folders.Error != nil {
			return nil, folders.Error
		}

		folderList := folders.Data.GetList() // resolve all folders including nested
		if folderList.Error != nil {
			return nil, folderList.Error
		}

		for i := range folderList.Data {
			folder := folderList.Data[i].(*mqlGcpFolder)

			folderConf := conn.Conf.Clone(inventory.WithParentConnectionId(conn.Conf.Id))
			if folderConf.Options == nil {
				folderConf.Options = map[string]string{}
			}
			folderConf.Options["folder-id"] = folder.Id.Data

			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewFolderPlatformID(folder.Id.Data),
				},
				Name: "GCP Folder " + folder.Id.Data,
				Platform: &inventory.Platform{
					Name:    "gcp-folder",
					Title:   "GCP Folder",
					Runtime: "gcp",
					Kind:    "gcp-object",
					Family:  []string{"google"},
				},
				Labels: map[string]string{},
				// NOTE: we explicitly do not exclude discovery here, as we want to discover the projects for the folder
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithParentConnectionId(conn.Conf.Id))},
			})
		}
	}
	return assetList, nil
}

func discoverFolder(conn *connection.GcpConnection, gcpFolder *mqlGcpFolder) ([]*inventory.Asset, error) {
	assetList := []*inventory.Asset{}

	if stringx.ContainsAnyOf(conn.Conf.Discover.Targets, DiscoveryAll, DiscoveryAuto, DiscoveryProjects) {
		projects := gcpFolder.GetProjects()
		if projects.Error != nil {
			return nil, projects.Error
		}

		projectList := projects.Data.GetList() // resolve all projects including nested
		if projectList.Error != nil {
			return nil, projectList.Error
		}

		for i := range projectList.Data {
			project := projectList.Data[i].(*mqlGcpProject)

			projectConf := conn.Conf.Clone(inventory.WithParentConnectionId(conn.Conf.Id))
			if projectConf.Options == nil {
				projectConf.Options = map[string]string{}
			}
			delete(projectConf.Options, "folder-id")
			projectConf.Options["project-id"] = project.Id.Data

			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewProjectPlatformID(project.Id.Data),
				},
				Name: project.Name.Data,
				Platform: &inventory.Platform{
					Name:                  "gcp-project",
					Title:                 "GCP Project " + project.Name.Data,
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: []string{"gcp", project.Id.Data, "project"},
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{projectConf}, // pass-in the parent connection config
			})
		}
	}
	return assetList, nil
}

func discoverProject(conn *connection.GcpConnection, gcpProject *mqlGcpProject) ([]*inventory.Asset, error) {
	assetList := []*inventory.Asset{}
	if stringx.ContainsAnyOf(conn.Conf.Discover.Targets, DiscoveryAll, DiscoveryInstances) {
		compute := gcpProject.GetCompute()
		if compute.Error != nil {
			return nil, compute.Error
		}
		instances := compute.Data.GetInstances()
		if instances.Error != nil {
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
					Name:                  "gcp-compute-instance",
					Title:                 "GCP Compute Instance",
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("compute", gcpProject.Id.Data, zone.Data.Name.Data, "instance", instance.Name.Data),
				},
				Labels: labels,
				// TODO: the current connection handling does not work well for instances
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))}, // pass-in the parent connection config
			})
		}
	}
	if stringx.ContainsAnyOf(conn.Conf.Discover.Targets, DiscoveryAll, DiscoveryComputeImages) {
		compute := gcpProject.GetCompute()
		if compute.Error != nil {
			return nil, compute.Error
		}
		images := compute.Data.GetImages()
		if images.Error != nil {
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
					Name:                  "gcp-compute-image",
					Title:                 "GCP Compute Image",
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("compute", gcpProject.Id.Data, "global", "image", image.Name.Data),
				},
				Labels:      labels,
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))}, // pass-in the parent connection config
			})
		}
	}
	if stringx.ContainsAnyOf(conn.Conf.Discover.Targets, DiscoveryAll, DiscoveryComputeNetworks) {
		compute := gcpProject.GetCompute()
		if compute.Error != nil {
			return nil, compute.Error
		}
		networks := compute.Data.GetNetworks()
		if networks.Error != nil {
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
					Name:                  "gcp-compute-network",
					Title:                 "GCP Compute Network",
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("compute", gcpProject.Id.Data, "global", "network", network.Name.Data),
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))}, // pass-in the parent connection config
			})
		}
	}
	if stringx.ContainsAnyOf(conn.Conf.Discover.Targets, DiscoveryAll, DiscoveryComputeSubnetworks) {
		compute := gcpProject.GetCompute()
		if compute.Error != nil {
			return nil, compute.Error
		}
		networks := compute.Data.GetSubnetworks()
		if networks.Error != nil {
			return nil, networks.Error
		}
		for i := range networks.Data {
			network := networks.Data[i].(*mqlGcpProjectComputeServiceSubnetwork)
			region := network.GetRegionUrl()
			if region.Error != nil {
				return nil, region.Error
			}
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("compute", gcpProject.Id.Data, RegionNameFromRegionUrl(region.Data), "subnetwork", network.Name.Data),
				},
				Name: network.Name.Data,
				Platform: &inventory.Platform{
					Name:                  "gcp-compute-subnetwork",
					Title:                 "GCP Compute Subnetwork",
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("compute", gcpProject.Id.Data, RegionNameFromRegionUrl(region.Data), "subnetwork", network.Name.Data),
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))}, // pass-in the parent connection config
			})
		}
	}
	if stringx.ContainsAnyOf(conn.Conf.Discover.Targets, DiscoveryAll, DiscoveryComputeFirewalls) {
		compute := gcpProject.GetCompute()
		if compute.Error != nil {
			return nil, compute.Error
		}
		firewalls := compute.Data.GetFirewalls()
		if firewalls.Error != nil {
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
					Name:                  "gcp-compute-firewall",
					Title:                 "GCP Compute Firewall",
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("compute", gcpProject.Id.Data, "global", "firewall", firewall.Name.Data),
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))}, // pass-in the parent connection config
			})
		}
	}
	if stringx.ContainsAnyOf(conn.Conf.Discover.Targets, DiscoveryAll, DiscoveryGkeClusters) {
		gke := gcpProject.GetGke()
		if gke.Error != nil {
			return nil, gke.Error
		}
		clusters := gke.Data.GetClusters()
		if clusters.Error != nil {
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
					Name:                  "gcp-gke-cluster",
					Title:                 "GCP GKE Cluster",
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("gke", gcpProject.Id.Data, cluster.GetLocation().Data, "cluster", cluster.Name.Data),
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))}, // pass-in the parent connection config
			})
		}
	}
	if stringx.ContainsAnyOf(conn.Conf.Discover.Targets, DiscoveryAll, DiscoveryStorageBuckets) {
		storage := gcpProject.GetStorage()
		if storage.Error != nil {
			return nil, storage.Error
		}
		buckets := storage.Data.GetBuckets()
		if buckets == nil {
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
					Name:                  "gcp-storage-bucket",
					Title:                 "GCP Storage Bucket",
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("storage", gcpProject.Id.Data, bucket.GetLocation().Data, "bucket", bucket.Name.Data),
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))}, // pass-in the parent connection config
			})
		}
	}
	if stringx.ContainsAnyOf(conn.Conf.Discover.Targets, DiscoveryAll, DiscoveryBigQueryDatasets) {
		bq := gcpProject.GetBigquery()
		if bq.Error != nil {
			return nil, bq.Error
		}
		datasets := bq.Data.GetDatasets()
		if datasets.Error != nil {
			return nil, datasets.Error
		}
		for i := range datasets.Data {
			dataset := datasets.Data[i].(*mqlGcpProjectBigqueryServiceDataset)
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("bigquery", gcpProject.Id.Data, dataset.GetLocation().Data, "dataset", dataset.Id.Data),
				},
				Name: dataset.Id.Data,
				Platform: &inventory.Platform{
					Name:                  "gcp-bigquery-dataset",
					Title:                 "GCP BigQuery Dataset",
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("bigquery", gcpProject.Id.Data, dataset.GetLocation().Data, "dataset", dataset.Id.Data),
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))}, // pass-in the parent connection config
			})
		}
	}

	return assetList, nil
}

func resolveGcr(ctx context.Context, conf *inventory.Config) ([]*inventory.Asset, error) {
	resolved := []*inventory.Asset{}
	repository := conf.Host

	log.Debug().Str("registry", repository).Msg("fetch meta information from gcr registry")
	gcrImages := NewGCRImages()
	assetList, err := gcrImages.ListRepository(repository, true)
	if err != nil {
		log.Error().Err(err).Msg("could not fetch gcr images")
		return nil, err
	}

	for i := range assetList {
		log.Debug().Str("name", assetList[i].Name).Str("image", assetList[i].Connections[0].Host+assetList[i].Connections[0].Path).Msg("resolved image")
		resolved = append(resolved, assetList[i])
	}

	return resolved, nil
}

func NewGCRImages() *GcrImages {
	return &GcrImages{}
}

type GcrImages struct{}

func (a *GcrImages) Name() string {
	return "GCP Container Registry Discover"
}

// lists a repository like "gcr.io/mondoo-base-infra"
func (a *GcrImages) ListRepository(repository string, recursive bool) ([]*inventory.Asset, error) {
	repo, err := name.NewRepository(repository)
	if err != nil {
		log.Fatal().Err(err).Str("repository", repository).Msg("could not create repository")
	}

	auth, err := google.Keychain.Resolve(repo.Registry)
	if err != nil {
		log.Fatal().Err(err).Str("repository", repository).Msg("failed to get auth for repository")
	}

	imgs := []*inventory.Asset{}

	toAssetFunc := func(repo name.Repository, tags *google.Tags, err error) error {
		if err != nil {
			return err
		}

		for digest := range tags.Manifests {
			repoURL := repo.String()
			imageUrl := repoURL + "@" + digest

			asset := &inventory.Asset{
				Connections: []*inventory.Config{
					{
						Type: "container-registry",
						Host: imageUrl,
					},
				},
			}
			imgs = append(imgs, asset)
		}
		return nil
	}

	// walk nested repos
	if recursive {
		err := google.Walk(repo, toAssetFunc, google.WithAuth(auth))
		if err != nil {
			return nil, err
		}
		return imgs, nil
	}

	// NOTE: since we're not recursing, we ignore tags.Children
	tags, err := google.List(repo, google.WithAuth(auth))
	if err != nil {
		return nil, err
	}

	err = toAssetFunc(repo, tags, nil)
	if err != nil {
		return nil, err
	}
	return imgs, nil
}

// List uses your GCP credentials to iterate over all your projects to identify potential repos
func (a *GcrImages) List() ([]*inventory.Asset, error) {
	assets := []*inventory.Asset{}

	resSrv, err := cloudresourcemanager.NewService(context.Background())
	if err != nil {
		return nil, err
	}

	projectsResp, err := resSrv.Projects.List().Do()
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup

	wg.Add(len(projectsResp.Projects))
	mux := &sync.Mutex{}
	for i := range projectsResp.Projects {

		project := projectsResp.Projects[i].Name
		go func() {
			repoAssets, err := a.ListRepository("gcr.io/"+project, true)
			if err == nil && repoAssets != nil {
				mux.Lock()
				assets = append(assets, repoAssets...)
				mux.Unlock()
			}
			wg.Done()
		}()
	}

	wg.Wait()
	return assets, nil
}
