// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/utils/stringx"
	"google.golang.org/api/cloudresourcemanager/v3"
)

const (
	// Discovery flags
	DiscoveryAuto = "auto"
	DiscoveryAll  = "all"

	// top-level assets
	DiscoveryFolders      = "folders"
	DiscoveryOrganization = "organization"
	DiscoveryProjects     = "projects"

	// resources
	DiscoverCloudDNSZones           = "cloud-dns-zones"
	DiscoverCloudKMSKeyrings        = "cloud-kms-keyrings"
	DiscoverMemorystoreRedis        = "memorystore-redis"
	DiscoverMemorystoreRedisCluster = "memorystore-rediscluster"
	DiscoverCloudSQLMySQL           = "cloud-sql-mysql"
	DiscoverCloudSQLPostgreSQL      = "cloud-sql-postgresql"
	DiscoverCloudSQLSQLServer       = "cloud-sql-sqlserver"
	DiscoveryBigQueryDatasets       = "bigquery-datasets"
	DiscoveryComputeFirewalls       = "compute-firewalls"
	DiscoveryComputeImages          = "compute-images"
	DiscoveryComputeNetworks        = "compute-networks"
	DiscoveryComputeSubnetworks     = "compute-subnetworks"
	DiscoveryGkeClusters            = "gke-clusters"
	DiscoveryComputeInstances       = "instances"
	DiscoveryStorageBuckets         = "storage-buckets"
	DiscoverSecretManager           = "secretmanager-secrets"
	DiscoverPubSubTopics            = "pubsub-topics"
	DiscoverPubSubSubscriptions     = "pubsub-subscriptions"
	DiscoverPubSubSnapshots         = "pubsub-snapshots"
	DiscoverCloudRunServices        = "cloudrun-services"
	DiscoverCloudRunJobs            = "cloudrun-jobs"
	DiscoverCloudFunctions          = "cloud-functions"
	DiscoverDataprocClusters        = "dataproc-clusters"
	DiscoverLoggingBuckets          = "logging-buckets"
	DiscoverApiKeys                 = "apikeys"
	DiscoverIamServiceAccounts      = "iam-service-accounts"
)

var All = []string{
	DiscoveryOrganization,
	DiscoveryFolders,
	DiscoveryProjects,
}

func allDiscovery() []string {
	return append(All, AllAPIResources...)
}

var Auto = []string{
	DiscoveryOrganization,
	DiscoveryFolders,
	DiscoveryProjects,
	DiscoveryComputeImages,
	DiscoveryComputeNetworks,
	DiscoveryComputeSubnetworks,
	DiscoveryComputeFirewalls,
	DiscoveryGkeClusters,
	DiscoveryStorageBuckets,
	DiscoveryBigQueryDatasets,
	DiscoverCloudSQLMySQL,
	DiscoverCloudSQLPostgreSQL,
	DiscoverCloudSQLSQLServer,
	DiscoverCloudDNSZones,
	DiscoverCloudKMSKeyrings,
	DiscoverMemorystoreRedis,
	DiscoverMemorystoreRedisCluster,
	DiscoveryComputeInstances,
	DiscoverSecretManager,
	DiscoverPubSubTopics,
	DiscoverPubSubSubscriptions,
	DiscoverPubSubSnapshots,
	DiscoverCloudRunServices,
	DiscoverCloudRunJobs,
	DiscoverCloudFunctions,
	DiscoverDataprocClusters,
	DiscoverLoggingBuckets,
	DiscoverApiKeys,
	DiscoverIamServiceAccounts,
}

var AllAPIResources = []string{
	DiscoveryComputeImages,
	DiscoveryComputeNetworks,
	DiscoveryComputeSubnetworks,
	DiscoveryComputeFirewalls,
	DiscoveryGkeClusters,
	DiscoveryStorageBuckets,
	DiscoveryBigQueryDatasets,
	DiscoverCloudSQLMySQL,
	DiscoverCloudSQLPostgreSQL,
	DiscoverCloudSQLSQLServer,
	DiscoverCloudDNSZones,
	DiscoverCloudKMSKeyrings,
	DiscoverMemorystoreRedis,
	DiscoverMemorystoreRedisCluster,
	DiscoveryComputeInstances,
	DiscoverSecretManager,
	DiscoverPubSubTopics,
	DiscoverPubSubSubscriptions,
	DiscoverPubSubSnapshots,
	DiscoverCloudRunServices,
	DiscoverCloudRunJobs,
	DiscoverCloudFunctions,
	DiscoverDataprocClusters,
	DiscoverLoggingBuckets,
	DiscoverApiKeys,
	DiscoverIamServiceAccounts,
}

// List of all CloudSQL types, this will be used during discovery
var AllCloudSQLTypes = []string{DiscoverCloudSQLPostgreSQL, DiscoverCloudSQLSQLServer, DiscoverCloudSQLMySQL}

func getDiscoveryTargets(config *inventory.Config) []string {
	targets := config.Discover.Targets

	if len(targets) == 0 {
		return Auto
	}

	if stringx.ContainsAnyOf(targets, DiscoveryAll) {
		// return the All list + All Api Resources list
		return allDiscovery()
	}
	if stringx.ContainsAnyOf(targets, DiscoveryAuto) {
		for i, target := range targets {
			if target == DiscoveryAuto {
				// remove the auto keyword
				targets = slices.Delete(targets, i, i+1)
			}
		}
		// add in the required discovery targets
		return append(targets, Auto...)
	}
	// random assortment of targets
	return targets
}

func Discover(runtime *plugin.Runtime) (*inventory.Inventory, error) {
	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not a GCP connection")
	}

	in := &inventory.Inventory{Spec: &inventory.InventorySpec{
		Assets: []*inventory.Asset{},
	}}
	discoveryTargets := getDiscoveryTargets(conn.Conf)

	if conn.ResourceType() == connection.Organization {
		res, err := NewResource(runtime, "gcp.organization", nil)
		if err != nil {
			return nil, err
		}

		gcpOrg := res.(*mqlGcpOrganization)

		list, err := discoverOrganization(conn, gcpOrg, discoveryTargets)
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
		if stringx.Contains(discoveryTargets, DiscoveryFolders) {
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

		list, err := discoverFolder(conn, gcpFolder, discoveryTargets)
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
		if stringx.Contains(discoveryTargets, DiscoveryProjects) {
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

		list, err := discoverProject(conn, gcpProject, discoveryTargets)
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

func discoverOrganization(conn *connection.GcpConnection, gcpOrg *mqlGcpOrganization, discoveryTargets []string) ([]*inventory.Asset, error) {
	assetList := []*inventory.Asset{}
	if stringx.Contains(discoveryTargets, DiscoveryProjects) {
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
				Labels:      mapStrInterfaceToMapStrStr(project.GetLabels().Data),
				Connections: []*inventory.Config{projectConf}, // pass-in the parent connection config
			})

			projectAssets, err := discoverProject(conn, project, discoveryTargets)
			if err != nil {
				return nil, err
			}
			assetList = append(assetList, projectAssets...)
		}
	}
	if stringx.Contains(discoveryTargets, DiscoveryFolders) {
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

func discoverFolder(conn *connection.GcpConnection, gcpFolder *mqlGcpFolder, discoveryTargets []string) ([]*inventory.Asset, error) {
	assetList := []*inventory.Asset{}

	if stringx.Contains(discoveryTargets, DiscoveryProjects) {
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
				Labels:      mapStrInterfaceToMapStrStr(project.GetLabels().Data),
				Connections: []*inventory.Config{projectConf}, // pass-in the parent connection config
			})
		}
	}
	return assetList, nil
}

func discoverProject(conn *connection.GcpConnection, gcpProject *mqlGcpProject, discoveryTargets []string) ([]*inventory.Asset, error) {
	assetList := []*inventory.Asset{}
	if stringx.Contains(discoveryTargets, DiscoveryComputeInstances) {
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
	if stringx.Contains(discoveryTargets, DiscoveryComputeImages) {
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
	if stringx.Contains(discoveryTargets, DiscoverCloudKMSKeyrings) {
		kmsservice := gcpProject.GetKms()
		if kmsservice.Error != nil {
			return nil, kmsservice.Error
		}
		keyrings := kmsservice.Data.GetKeyrings()
		if keyrings.Error != nil {
			return nil, keyrings.Error
		}

		for i := range keyrings.Data {
			keyring := keyrings.Data[i].(*mqlGcpProjectKmsServiceKeyring)
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("cloud-kms", gcpProject.Id.Data, keyring.Location.Data, "keyring", keyring.Name.Data),
				},
				Name: keyring.Name.Data,
				Platform: &inventory.Platform{
					Name:                  "gcp-kms-keyring",
					Title:                 "GCP Cloud KMS Keyring",
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("cloud-kms", gcpProject.Id.Data, keyring.Location.Data, "keyring", keyring.Name.Data),
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))}, // pass-in the parent connection config
			})
		}
	}
	if stringx.Contains(discoveryTargets, DiscoverCloudDNSZones) {
		dnsservice := gcpProject.GetDns()
		if dnsservice.Error != nil {
			return nil, dnsservice.Error
		}
		managedzones := dnsservice.Data.GetManagedZones()
		if managedzones.Error != nil {
			return nil, managedzones.Error
		}

		for i := range managedzones.Data {
			managedzone := managedzones.Data[i].(*mqlGcpProjectDnsServiceManagedzone)
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("cloud-dns", gcpProject.Id.Data, "global", "zone", managedzone.Id.Data),
				},
				Name: managedzone.Name.Data,
				Platform: &inventory.Platform{
					Name:                  "gcp-dns-zone",
					Title:                 "GCP Cloud DNS Zone",
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("cloud-dns", gcpProject.Id.Data, "global", "zone", managedzone.Name.Data),
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))}, // pass-in the parent connection config
			})
		}
	}
	// all Cloud SQL discovery flags/types
	if stringx.ContainsAnyOf(discoveryTargets, AllCloudSQLTypes...) {
		sqlservice := gcpProject.GetSql()
		if sqlservice.Error != nil {
			return nil, sqlservice.Error
		}
		sqlinstances := sqlservice.Data.GetInstances()
		if sqlinstances.Error != nil {
			return nil, sqlinstances.Error
		}

		for i := range sqlinstances.Data {
			var (
				sqlinstance    = sqlinstances.Data[i].(*mqlGcpProjectSqlServiceInstance)
				sqlTypeVersion = strings.Split(sqlinstance.DatabaseInstalledVersion.Data, "_")
				sqlType        = connection.ParseCloudSQLType(sqlTypeVersion[0])
				platformName   = fmt.Sprintf("gcp-sql-%s", sqlType)
			)

			if !slices.Contains(discoveryTargets, fmt.Sprintf("cloud-sql-%s", sqlType)) {
				log.Debug().
					Str("sql_type", sqlType).
					Msg("gcp.discovery> skipping cloud sql instance")
				continue // only discover known sql types
			}

			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("cloud-sql", gcpProject.Id.Data, sqlinstance.Region.Data, sqlType, sqlinstance.Name.Data),
				},
				Name: sqlinstance.Name.Data,
				Platform: &inventory.Platform{
					Name:                  platformName,
					Title:                 connection.GetTitleForPlatformName(platformName),
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("cloud-sql", gcpProject.Id.Data, sqlinstance.Region.Data, sqlType, sqlinstance.Name.Data),
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))}, // pass-in the parent connection config
			})
		}
	}
	if stringx.ContainsAnyOf(discoveryTargets, DiscoverMemorystoreRedis) {
		redisService := gcpProject.GetRedis()
		if redisService.Error != nil {
			return nil, redisService.Error
		}
		redisInstances := redisService.Data.GetInstances()
		if redisInstances.Error != nil {
			return nil, redisInstances.Error
		}

		for i := range redisInstances.Data {
			redisInstance := redisInstances.Data[i].(*mqlGcpProjectRedisServiceInstance)

			// Extract instance name from full resource path
			// (projects/{project}/locations/{location}/instances/{instance_id})
			nameParts := strings.Split(redisInstance.Name.Data, "/")
			instanceName := nameParts[len(nameParts)-1]

			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("memorystore", gcpProject.Id.Data, redisInstance.LocationId.Data, "redis", instanceName),
				},
				Name: instanceName,
				Platform: &inventory.Platform{
					Name:                  "gcp-memorystore-redis",
					Title:                 connection.GetTitleForPlatformName("gcp-memorystore-redis"),
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("memorystore", gcpProject.Id.Data, redisInstance.LocationId.Data, "redis", instanceName),
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))},
			})
		}
	}
	if stringx.ContainsAnyOf(discoveryTargets, DiscoverMemorystoreRedisCluster) {
		redisService := gcpProject.GetRedis()
		if redisService.Error != nil {
			return nil, redisService.Error
		}
		redisClusters := redisService.Data.GetClusters()
		if redisClusters.Error != nil {
			return nil, redisClusters.Error
		}

		for i := range redisClusters.Data {
			redisCluster := redisClusters.Data[i].(*mqlGcpProjectRedisServiceCluster)

			// Extract cluster name and location from full resource path
			// (projects/{project}/locations/{location}/clusters/{cluster_id})
			nameParts := strings.Split(redisCluster.Name.Data, "/")
			clusterName := nameParts[len(nameParts)-1]
			var location string
			if len(nameParts) >= 4 {
				location = nameParts[3]
			}

			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("memorystore", gcpProject.Id.Data, location, "rediscluster", clusterName),
				},
				Name: clusterName,
				Platform: &inventory.Platform{
					Name:                  "gcp-memorystore-rediscluster",
					Title:                 connection.GetTitleForPlatformName("gcp-memorystore-rediscluster"),
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("memorystore", gcpProject.Id.Data, location, "rediscluster", clusterName),
				},
				Labels:      map[string]string{},
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))},
			})
		}
	}
	if stringx.ContainsAnyOf(discoveryTargets, DiscoveryComputeNetworks) {
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
	if stringx.ContainsAnyOf(discoveryTargets, DiscoveryComputeSubnetworks) {
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
	if stringx.ContainsAnyOf(discoveryTargets, DiscoveryComputeFirewalls) {
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
	if stringx.ContainsAnyOf(discoveryTargets, DiscoveryGkeClusters) {
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
				Labels:      mapStrInterfaceToMapStrStr(cluster.GetResourceLabels().Data),
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))}, // pass-in the parent connection config
			})
		}
	}
	if stringx.ContainsAnyOf(discoveryTargets, DiscoveryStorageBuckets) {
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
				Labels:      mapStrInterfaceToMapStrStr(bucket.GetLabels().Data),
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))}, // pass-in the parent connection config
			})
		}
	}
	if stringx.ContainsAnyOf(discoveryTargets, DiscoveryBigQueryDatasets) {
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
				Labels:      mapStrInterfaceToMapStrStr(dataset.GetLabels().Data),
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))}, // pass-in the parent connection config
			})
		}
	}
	if stringx.ContainsAnyOf(discoveryTargets, DiscoverSecretManager) {
		secretmanagerService := gcpProject.GetSecretmanager()
		if secretmanagerService.Error != nil {
			return nil, secretmanagerService.Error
		}
		secrets := secretmanagerService.Data.GetSecrets()
		if secrets.Error != nil {
			return nil, secrets.Error
		}
		for i := range secrets.Data {
			secret := secrets.Data[i].(*mqlGcpProjectSecretmanagerServiceSecret)
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("secretmanager", gcpProject.Id.Data, "global", "secret", secret.Name.Data),
				},
				Name: secret.Name.Data,
				Platform: &inventory.Platform{
					Name:                  "gcp-secretmanager-secret",
					Title:                 connection.GetTitleForPlatformName("gcp-secretmanager-secret"),
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("secretmanager", gcpProject.Id.Data, "global", "secret", secret.Name.Data),
				},
				Labels:      mapStrInterfaceToMapStrStr(secret.GetLabels().Data),
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))},
			})
		}
	}

	if stringx.ContainsAnyOf(discoveryTargets, DiscoverPubSubTopics) {
		pubsubService := gcpProject.GetPubsub()
		if pubsubService.Error != nil {
			return nil, pubsubService.Error
		}
		topics := pubsubService.Data.GetTopics()
		if topics.Error != nil {
			return nil, topics.Error
		}
		for i := range topics.Data {
			topic := topics.Data[i].(*mqlGcpProjectPubsubServiceTopic)
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("pubsub", gcpProject.Id.Data, "global", "topic", topic.Name.Data),
				},
				Name: topic.Name.Data,
				Platform: &inventory.Platform{
					Name:                  "gcp-pubsub-topic",
					Title:                 connection.GetTitleForPlatformName("gcp-pubsub-topic"),
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("pubsub", gcpProject.Id.Data, "global", "topic", topic.Name.Data),
				},
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))},
			})
		}
	}

	if stringx.ContainsAnyOf(discoveryTargets, DiscoverPubSubSubscriptions) {
		pubsubService := gcpProject.GetPubsub()
		if pubsubService.Error != nil {
			return nil, pubsubService.Error
		}
		subscriptions := pubsubService.Data.GetSubscriptions()
		if subscriptions.Error != nil {
			return nil, subscriptions.Error
		}
		for i := range subscriptions.Data {
			sub := subscriptions.Data[i].(*mqlGcpProjectPubsubServiceSubscription)
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("pubsub", gcpProject.Id.Data, "global", "subscription", sub.Name.Data),
				},
				Name: sub.Name.Data,
				Platform: &inventory.Platform{
					Name:                  "gcp-pubsub-subscription",
					Title:                 connection.GetTitleForPlatformName("gcp-pubsub-subscription"),
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("pubsub", gcpProject.Id.Data, "global", "subscription", sub.Name.Data),
				},
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))},
			})
		}
	}

	if stringx.ContainsAnyOf(discoveryTargets, DiscoverPubSubSnapshots) {
		pubsubService := gcpProject.GetPubsub()
		if pubsubService.Error != nil {
			return nil, pubsubService.Error
		}
		snapshots := pubsubService.Data.GetSnapshots()
		if snapshots.Error != nil {
			return nil, snapshots.Error
		}
		for i := range snapshots.Data {
			snap := snapshots.Data[i].(*mqlGcpProjectPubsubServiceSnapshot)
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("pubsub", gcpProject.Id.Data, "global", "snapshot", snap.Name.Data),
				},
				Name: snap.Name.Data,
				Platform: &inventory.Platform{
					Name:                  "gcp-pubsub-snapshot",
					Title:                 connection.GetTitleForPlatformName("gcp-pubsub-snapshot"),
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("pubsub", gcpProject.Id.Data, "global", "snapshot", snap.Name.Data),
				},
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))},
			})
		}
	}

	if stringx.ContainsAnyOf(discoveryTargets, DiscoverCloudRunServices) {
		cloudRunService := gcpProject.GetCloudRun()
		if cloudRunService.Error != nil {
			return nil, cloudRunService.Error
		}
		services := cloudRunService.Data.GetServices()
		if services.Error != nil {
			return nil, services.Error
		}
		for i := range services.Data {
			svc := services.Data[i].(*mqlGcpProjectCloudRunServiceService)
			region := svc.Region.Data
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("cloudrun", gcpProject.Id.Data, region, "service", svc.Name.Data),
				},
				Name: svc.Name.Data,
				Platform: &inventory.Platform{
					Name:                  "gcp-cloudrun-service",
					Title:                 connection.GetTitleForPlatformName("gcp-cloudrun-service"),
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("cloudrun", gcpProject.Id.Data, region, "service", svc.Name.Data),
				},
				Labels:      mapStrInterfaceToMapStrStr(svc.GetLabels().Data),
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))},
			})
		}
	}

	if stringx.ContainsAnyOf(discoveryTargets, DiscoverCloudRunJobs) {
		cloudRunService := gcpProject.GetCloudRun()
		if cloudRunService.Error != nil {
			return nil, cloudRunService.Error
		}
		jobs := cloudRunService.Data.GetJobs()
		if jobs.Error != nil {
			return nil, jobs.Error
		}
		for i := range jobs.Data {
			job := jobs.Data[i].(*mqlGcpProjectCloudRunServiceJob)
			region := job.Region.Data
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("cloudrun", gcpProject.Id.Data, region, "job", job.Name.Data),
				},
				Name: job.Name.Data,
				Platform: &inventory.Platform{
					Name:                  "gcp-cloudrun-job",
					Title:                 connection.GetTitleForPlatformName("gcp-cloudrun-job"),
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("cloudrun", gcpProject.Id.Data, region, "job", job.Name.Data),
				},
				Labels:      mapStrInterfaceToMapStrStr(job.GetLabels().Data),
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))},
			})
		}
	}

	if stringx.ContainsAnyOf(discoveryTargets, DiscoverCloudFunctions) {
		funcs := gcpProject.GetCloudFunctions()
		if funcs.Error != nil {
			return nil, funcs.Error
		}
		for i := range funcs.Data {
			fn := funcs.Data[i].(*mqlGcpProjectCloudFunction)
			location := fn.Location.Data
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("cloud-functions", gcpProject.Id.Data, location, "function", fn.Name.Data),
				},
				Name: fn.Name.Data,
				Platform: &inventory.Platform{
					Name:                  "gcp-cloud-function",
					Title:                 connection.GetTitleForPlatformName("gcp-cloud-function"),
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("cloud-functions", gcpProject.Id.Data, location, "function", fn.Name.Data),
				},
				Labels:      mapStrInterfaceToMapStrStr(fn.GetLabels().Data),
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))},
			})
		}
	}

	if stringx.ContainsAnyOf(discoveryTargets, DiscoverDataprocClusters) {
		dataprocService := gcpProject.GetDataproc()
		if dataprocService.Error != nil {
			return nil, dataprocService.Error
		}
		clusters := dataprocService.Data.GetClusters()
		if clusters.Error != nil {
			return nil, clusters.Error
		}
		for i := range clusters.Data {
			cluster := clusters.Data[i].(*mqlGcpProjectDataprocServiceCluster)
			location := cluster.Location.Data
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("dataproc", gcpProject.Id.Data, location, "cluster", cluster.Name.Data),
				},
				Name: cluster.Name.Data,
				Platform: &inventory.Platform{
					Name:                  "gcp-dataproc-cluster",
					Title:                 connection.GetTitleForPlatformName("gcp-dataproc-cluster"),
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("dataproc", gcpProject.Id.Data, location, "cluster", cluster.Name.Data),
				},
				Labels:      mapStrInterfaceToMapStrStr(cluster.GetLabels().Data),
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))},
			})
		}
	}

	if stringx.ContainsAnyOf(discoveryTargets, DiscoverLoggingBuckets) {
		loggingService := gcpProject.GetLogging()
		if loggingService.Error != nil {
			return nil, loggingService.Error
		}
		buckets := loggingService.Data.GetBuckets()
		if buckets.Error != nil {
			return nil, buckets.Error
		}
		for i := range buckets.Data {
			bucket := buckets.Data[i].(*mqlGcpProjectLoggingserviceBucket)
			bucketName := parseResourceName(bucket.Name.Data)
			location := bucket.Location.Data
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("logging", gcpProject.Id.Data, location, "bucket", bucketName),
				},
				Name: bucketName,
				Platform: &inventory.Platform{
					Name:                  "gcp-logging-bucket",
					Title:                 connection.GetTitleForPlatformName("gcp-logging-bucket"),
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("logging", gcpProject.Id.Data, location, "bucket", bucketName),
				},
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))},
			})
		}
	}

	if stringx.ContainsAnyOf(discoveryTargets, DiscoverApiKeys) {
		keys := gcpProject.GetApiKeys()
		if keys.Error != nil {
			return nil, keys.Error
		}
		for i := range keys.Data {
			key := keys.Data[i].(*mqlGcpProjectApiKey)
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("apikeys", gcpProject.Id.Data, "global", "key", key.Id.Data),
				},
				Name: key.Name.Data,
				Platform: &inventory.Platform{
					Name:                  "gcp-apikey",
					Title:                 connection.GetTitleForPlatformName("gcp-apikey"),
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("apikeys", gcpProject.Id.Data, "global", "key", key.Id.Data),
				},
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))},
			})
		}
	}

	if stringx.ContainsAnyOf(discoveryTargets, DiscoverIamServiceAccounts) {
		iamSvc := gcpProject.GetIam()
		if iamSvc.Error != nil {
			return nil, iamSvc.Error
		}
		sas := iamSvc.Data.GetServiceAccounts()
		if sas.Error != nil {
			return nil, sas.Error
		}
		for i := range sas.Data {
			sa := sas.Data[i].(*mqlGcpProjectIamServiceServiceAccount)
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{
					connection.NewResourcePlatformID("iam", gcpProject.Id.Data, "global", "service-account", sa.UniqueId.Data),
				},
				Name: sa.Email.Data,
				Platform: &inventory.Platform{
					Name:                  "gcp-iam-service-account",
					Title:                 connection.GetTitleForPlatformName("gcp-iam-service-account"),
					Runtime:               "gcp",
					Kind:                  "gcp-object",
					Family:                []string{"google"},
					TechnologyUrlSegments: connection.ResourceTechnologyUrl("iam", gcpProject.Id.Data, "global", "service-account", sa.UniqueId.Data),
				},
				Connections: []*inventory.Config{conn.Conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.Conf.Id))},
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

func mapStrInterfaceToMapStrStr(m map[string]any) map[string]string {
	strMap := make(map[string]string)
	for k, v := range m {
		if v != nil {
			strMap[k] = v.(string)
		}
	}
	return strMap
}
