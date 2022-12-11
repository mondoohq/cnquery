package gcp

import (
	"context"
	"errors"
	"strconv"
	"sync"

	"go.mondoo.com/cnquery/resources"

	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpCompute) init(args *resources.Args) (*resources.Args, GcpCompute, error) {
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

func (g *mqlGcpCompute) id() (string, error) {
	id, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return "gcp.compute/" + id, nil
}

func (g *mqlGcpComputeRegion) id() (string, error) {
	id, err := g.Name()
	if err != nil {
		return "", err
	}
	return "gcp.compute.region/" + id, nil
}

func (g *mqlGcpComputeRegion) init(args *resources.Args) (*resources.Args, GcpComputeRegion, error) {
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

func (g *mqlGcpCompute) GetRegions() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
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

	res := []interface{}{}
	for i := range regions.Items {
		r := regions.Items[i]

		deprecated, err := core.JsonToDict(r.Deprecated)
		if err != nil {
			return nil, err
		}

		quotas := map[string]interface{}{}
		for i := range r.Quotas {
			q := r.Quotas[i]
			quotas[q.Metric] = q.Limit
		}

		mqlServiceaccount, err := g.MotorRuntime.CreateResource("gcp.compute.region",
			"id", r.SelfLink,
			"name", r.Name,
			"description", r.Description,
			"status", r.Status,
			"created", parseTime(r.CreationTimestamp),
			"quotas", quotas,
			"deprecated", deprecated,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlServiceaccount)
	}

	return res, nil
}

func (g *mqlGcpComputeZone) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "gcp.compute.zone/" + id, nil
}

func (g *mqlGcpComputeZone) GetRegion() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (g *mqlGcpCompute) GetZones() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	zones, err := computeSvc.Zones.List(projectId).Do()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range zones.Items {
		z := zones.Items[i]

		mqlServiceaccount, err := g.MotorRuntime.CreateResource("gcp.compute.zone",
			"id", z.SelfLink,
			"name", z.Name,
			"description", z.Description,
			"status", z.Status,
			"created", parseTime(z.CreationTimestamp),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlServiceaccount)
	}

	return res, nil
}

func (g *mqlGcpComputeInstance) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpCompute) GetInstances() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	zones, err := computeSvc.Zones.List(projectId).Do()
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	wg.Add(len(zones.Items))
	mux := &sync.Mutex{}

	// TODO:harmonize instance list with discovery?
	for _, z := range zones.Items {
		go func(svc *compute.Service, project string, zoneName string) {
			instances, err := computeSvc.Instances.List(projectId, zoneName).Do()
			if err == nil {
				mux.Lock()
				for i := range instances.Items {
					instance := instances.Items[i]

					metadata := map[string]string{}
					for m := range instance.Metadata.Items {
						item := instance.Metadata.Items[m]
						metadata[item.Key] = core.ToString(item.Value)
					}

					mqlServiceAccounts := []interface{}{}
					for i := range instance.ServiceAccounts {
						sa := instance.ServiceAccounts[i]

						mqlServiceaccount, err := g.MotorRuntime.CreateResource("gcp.compute.serviceaccount",
							"email", sa.Email,
							"scopes", core.StrSliceToInterface(sa.Scopes),
						)
						if err == nil {
							mqlServiceAccounts = append(mqlServiceAccounts, mqlServiceaccount)
						}
					}

					mqlInstance, err := g.MotorRuntime.CreateResource("gcp.compute.instance",
						"id", strconv.FormatUint(instance.Id, 10),
						"name", instance.Name,
						"cpuPlatform", instance.CpuPlatform,
						"deletionProtection", instance.DeletionProtection,
						"description", instance.Description,
						"hostname", instance.Hostname,
						"labels", core.StrMapToInterface(instance.Labels),
						"status", instance.Status,
						"statusMessage", instance.StatusMessage,
						"tags", core.StrSliceToInterface(instance.Tags.Items),
						"metadata", core.StrMapToInterface(metadata),
						"serviceAccounts", mqlServiceAccounts,
					)
					if err == nil {
						res = append(res, mqlInstance)
					}
				}
				mux.Unlock()
			}
			wg.Done()
		}(computeSvc, projectId, z.Name)
	}

	wg.Wait()
	return res, nil
}

func (g *mqlGcpComputeServiceaccount) id() (string, error) {
	return g.Email()
}
