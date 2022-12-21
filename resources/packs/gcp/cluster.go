package gcp

import (
	"context"
	"fmt"

	container "cloud.google.com/go/container/apiv1"
	"cloud.google.com/go/container/apiv1/containerpb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/option"
)

func (g *mqlGcpCluster) id() (string, error) {
	id, err := g.Name()
	if err != nil {
		return "", err
	}
	return "gcp.cluster/" + id, nil
}

func (g *mqlGcpCluster) init(args *resources.Args) (*resources.Args, GcpCluster, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	projectId := provider.ResourceID()
	(*args)["projectId"] = projectId

	return args, nil, nil
}

func (g *mqlGcpProject) GetClusters() ([]interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(container.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	containerSvc, err := container.NewClusterManagerClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}

	// List the clusters in the current projects for all locations
	resp, err := containerSvc.ListClusters(ctx, &containerpb.ListClustersRequest{Parent: fmt.Sprintf("projects/%s/locations/-", projectId)})
	if err != nil {
		log.Error().Err(err).Msg("failed to list clusters")
		return nil, err
	}
	res := []interface{}{}

	for i := range resp.Clusters {
		c := resp.Clusters[i]

		mqlCluster, err := g.MotorRuntime.CreateResource("gcp.cluster",
			"projectId", projectId,
			"id", c.Id,
			"name", c.Name,
			"description", c.Description,
			"loggingService", c.LoggingService,
			"monitoringService", c.MonitoringService,
			"network", c.Network,
			"clusterIpv4Cidr", c.ClusterIpv4Cidr,
			"subnetwork", c.Subnetwork,
			"locations", core.StrSliceToInterface(c.Locations),
			"enableKubernetesAlpha", c.EnableKubernetesAlpha,
			"autopilotEnabled", c.Autopilot.Enabled,
			"resourceLabels", core.StrMapToInterface(c.ResourceLabels),
			"created", parseTime(c.CreateTime),
			"expirationTime", parseTime(c.ExpireTime),
			// "location", c.Location,
			// "status", c.Status,
			// "created", parseTime(c.CreateTime),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCluster)
	}

	return res, nil
}
