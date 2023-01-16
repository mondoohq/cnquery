package gcp

import (
	"context"
	"fmt"
	"sync"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	run "cloud.google.com/go/run/apiv2"
	runpb "cloud.google.com/go/run/apiv2/runpb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectCloudrunService) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/gcp.project.cloudrunService", projectId), nil
}

func (g *mqlGcpProjectCloudrunService) init(args *resources.Args) (*resources.Args, GcpProjectCloudrunService, error) {
	if len(*args) > 0 {
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

func (g *mqlGcpProject) GetCloudrun() (interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.project.cloudrunService",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectCloudrunServiceOperation) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("gcp.project.cloudrunService.operation/%s/%s", projectId, name), nil
}

func (g *mqlGcpProjectCloudrunServiceService) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("gcp.project.cloudrunService.service/%s/%s", projectId, name), nil
}

func (g *mqlGcpProjectCloudrunServiceServiceRevisionTemplate) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectCloudrunServiceServiceRevisionTemplateContainer) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectCloudrunServiceServiceCondition) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectCloudrunService) GetRegions() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(run.DefaultAuthScopes()...)
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

func (g *mqlGcpProjectCloudrunService) GetOperations() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	regions, err := g.Regions()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(run.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	runSvc, err := run.NewServicesClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer runSvc.Close()

	var wg sync.WaitGroup
	var operations []interface{}
	wg.Add(len(regions))
	mux := &sync.Mutex{}
	for _, region := range regions {
		go func(region string) {
			defer wg.Done()
			it := runSvc.ListOperations(ctx, &longrunningpb.ListOperationsRequest{Name: fmt.Sprintf("projects/%s/locations/%s", projectId, region)})
			for {
				t, err := it.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					log.Error().Err(err).Send()
				}
				mqlOp, err := g.MotorRuntime.CreateResource("gcp.project.cloudrunService.operation",
					"projectId", projectId,
					"name", t.Name,
					"done", t.Done,
				)
				if err != nil {
					log.Error().Err(err).Send()
				}
				mux.Lock()
				operations = append(operations, mqlOp)
				mux.Unlock()
			}
		}(region.(string))
	}
	wg.Wait()
	return operations, nil
}

func (g *mqlGcpProjectCloudrunService) GetServices() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	regions, err := g.Regions()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(run.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	runSvc, err := run.NewServicesClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer runSvc.Close()

	var wg sync.WaitGroup
	var services []interface{}
	wg.Add(len(regions))
	mux := &sync.Mutex{}
	for _, region := range regions {
		go func(region string) {
			defer wg.Done()
			it := runSvc.ListServices(ctx, &runpb.ListServicesRequest{Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region)})
			for {
				s, err := it.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					log.Error().Err(err).Send()
				}
				mqlS, err := g.MotorRuntime.CreateResource("gcp.project.cloudrunService.service",
					"projectId", projectId,
					"region", region,
					"name", s.Name,
					"description", s.Description,
					"generation", s.Generation,
					"labels", core.StrMapToInterface(s.Labels),
					"annotations", core.StrMapToInterface(s.Annotations),
					"created", core.MqlTime(s.CreateTime.AsTime()),
					"updated", core.MqlTime(s.UpdateTime.AsTime()),
					"deleted", core.MqlTime(s.DeleteTime.AsTime()),
					"expired", core.MqlTime(s.ExpireTime.AsTime()),
					"creator", s.Creator,
					"lastModifier", s.LastModifier,
					"client", s.Client,
					"clientVersion", s.ClientVersion,
					"ingress", s.Ingress.String(),
					"launchStage", s.LaunchStage.String(),
					"binaryAuthorization", nil, // TODO
					"traffic", nil, // TODO
					"observedGeneration", s.ObservedGeneration,
					"terminalConditions", nil, // TODO
					"terminalCondition", nil, // TODO
					"conditions", nil, // TODO
					"latestReadyRevision", s.LatestReadyRevision,
					"latestCreatedRevision", s.LatestCreatedRevision,
					"trafficStatuses", nil, // TODO
					"uri", s.Uri,
					"reconciling", s.Reconciling,
				)
				if err != nil {
					log.Error().Err(err).Send()
				}
				mux.Lock()
				services = append(services, mqlS)
				mux.Unlock()
			}
		}(region.(string))
	}
	wg.Wait()
	return services, nil
}
