package gcp

import (
	"context"
	"strconv"
	"sync"

	"github.com/hashicorp/go-multierror"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcloudCompute) id() (string, error) {
	return "gcloud.compute", nil
}

func (g *mqlGcloudCompute) GetInstances() ([]interface{}, error) {
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

						mqlServiceaccount, err := g.MotorRuntime.CreateResource("gcloud.compute.serviceaccount",
							"email", sa.Email,
							"scopes", core.StrSliceToInterface(sa.Scopes),
						)
						if err == nil {
							mqlServiceAccounts = append(mqlServiceAccounts, mqlServiceaccount)
						}
					}

					mqlInstance, err := g.MotorRuntime.CreateResource("gcloud.compute.instance",
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

func (g *mqlGcloudComputeInstance) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", nil
	}

	return "gcloud.compute.instance/" + id, nil
}

func (g *mqlGcloudComputeServiceaccount) id() (string, error) {
	email, err := g.Email()
	if err != nil {
		return "", nil
	}
	return "gcloud.compute.serviceaccount/" + email, nil
}

func (g *mqlGcloudCompute) GetDisks() ([]interface{}, error) {
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

	var result error

	for _, z := range zones.Items {
		go func(svc *compute.Service, project string, zoneName string) {
			disks, err := computeSvc.Disks.List(projectName, zoneName).Do()
			if err == nil {
				mux.Lock()
				for i := range disks.Items {
					disk := disks.Items[i]

					zone, err := newMqlGcpComputeZone(g.MotorRuntime, disk.Zone)
					if err != nil {
						result = multierror.Append(result, err)
					}

					mqlInstance, err := g.MotorRuntime.CreateResource("gcloud.compute.disk",
						"id", strconv.FormatUint(disk.Id, 10),
						"name", disk.Name,
						"architecture", disk.Architecture,
						"description", disk.Description,
						"labels", core.StrMapToInterface(disk.Labels),
						"locationHint", disk.LocationHint,
						"physicalBlockSizeBytes", disk.PhysicalBlockSizeBytes,
						"provisionedIops", disk.ProvisionedIops,
						//"region", disk.Region, // TODO: parse the region url
						//"replicaZones", core.StrSliceToInterface(disk.ReplicaZones),
						//"resourcePolicies", core.StrSliceToInterface(disk.ResourcePolicies),
						"sizeGb", disk.SizeGb,
						// TODO: link to resources
						//"sourceDiskId", disk.SourceDiskId,
						//"sourceImageId", disk.SourceImageId,
						//"sourceSnapshotId", disk.SourceSnapshotId,
						"status", disk.Status,
						"zone", zone,
					)
					if err != nil {
						result = multierror.Append(result, err)
					} else {
						res = append(res, mqlInstance)
					}
				}
				mux.Unlock()
			}
			wg.Done()
		}(computeSvc, projectName, z.Name)
	}

	wg.Wait()

	return res, result
}

func (g *mqlGcloudComputeDisk) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", nil
	}

	return "gcloud.compute.disk/" + id, nil
}

func (g *mqlGcloudComputeZone) id() (string, error) {
	project, err := g.ProjectId()
	if err != nil {
		return "", nil
	}

	name, err := g.Name()
	if err != nil {
		return "", nil
	}

	return "gcloud.compute.zone/projects/" + project + "/zones/" + name, nil
}

func newMqlGcpComputeZone(runtime *resources.Runtime, zoneResourceName string) (interface{}, error) {
	zone, err := parseZone(zoneResourceName)
	if err != nil {
		return nil, err
	}

	return runtime.CreateResource("gcloud.compute.zone",
		"projectId", zone.ProjectID,
		"name", zone.Name,
	)
}
