package gcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/dataproc/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectDataprocService) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/gcp.project.dataprocService", projectId), nil
}

func (g *mqlGcpProject) GetDataproc() (interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.project.dataprocService",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectDataprocService) GetRegions() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(dataproc.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	regions, err := computeSvc.Regions.List(projectId).Do()
	if err != nil {
		return nil, err
	}

	regionNames := make([]interface{}, 0, len(regions.Items))
	for _, region := range regions.Items {
		regionNames = append(regionNames, region.Name)
	}
	return regionNames, nil
}

func (g *mqlGcpProjectDataprocService) GetClusters() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	regions, err := g.GetRegions()
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(dataproc.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	dataprocSvc, err := dataproc.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	var mqlClusters []interface{}
	wg.Add(len(regions))
	mux := &sync.Mutex{}
	for _, region := range regions {
		go func(projectId, regionName string) {
			defer wg.Done()
			clusters, err := dataprocSvc.Projects.Regions.Clusters.List(projectId, regionName).Do()
			if err != nil {
				log.Error().Err(err).Send()
			}

			for _, c := range clusters.Clusters {
				mqlCluster, err := g.MotorRuntime.CreateResource("gcp.project.dataprocService.cluster",
					"projectId", projectId,
					"clusterName", c.ClusterName,
					"clusterUuid", c.ClusterUuid,
					"config", nil, // TODO
					"labels", core.StrMapToInterface(c.Labels),
					"metrics", nil, // TODO
					"status", nil, // TODO
					"statusHistory", nil, // TODO
					"virtualClusterConfig", nil, // TODO
				)
				if err != nil {
					log.Error().Err(err).Send()
				}
				mux.Lock()
				mqlClusters = append(mqlClusters, mqlCluster)
				mux.Unlock()
			}
		}(projectId, region.(string))
	}
	wg.Wait()
	return mqlClusters, nil
}

func (g *mqlGcpProjectDataprocServiceCluster) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	name, err := g.ClusterName()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", projectId, name), nil
}

func (g *mqlGcpProjectDataprocServiceClusterConfig) id() (string, error) {
	parentResource, err := g.ParentResourcePath()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/config", parentResource), nil
}

func (g *mqlGcpProjectDataprocServiceClusterMetrics) id() (string, error) {
	parentResource, err := g.ParentResourcePath()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/metrics", parentResource), nil
}

func (g *mqlGcpProjectDataprocServiceClusterStatus) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectDataprocServiceClusterVirtualClusterConfig) id() (string, error) {
	parentResource, err := g.ParentResourcePath()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/virtualClusterConfig", parentResource), nil
}
