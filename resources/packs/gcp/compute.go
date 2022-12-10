package gcp

import (
	"context"
	"strconv"
	"sync"

	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpCompute) id() (string, error) {
	return "gcp.compute", nil
}

func (g *mqlGcpCompute) GetInstances() ([]interface{}, error) {
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

	projectName := provider.ResourceID()

	var wg sync.WaitGroup
	zones, err := computeSvc.Zones.List(projectName).Do()
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	wg.Add(len(zones.Items))
	mux := &sync.Mutex{}

	// TODO:harmonize instance list with discovery?
	for _, z := range zones.Items {
		go func(svc *compute.Service, project string, zoneName string) {
			instances, err := computeSvc.Instances.List(projectName, zoneName).Do()
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
		}(computeSvc, projectName, z.Name)
	}

	wg.Wait()
	return res, nil
}

func (g *mqlGcpComputeInstance) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpComputeServiceaccount) id() (string, error) {
	return g.Email()
}
