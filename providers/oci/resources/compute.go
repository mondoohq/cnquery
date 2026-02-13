// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (e *mqlOciCompute) id() (string, error) {
	return "oci.compute", nil
}

func (o *mqlOciCompute) instances() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	// fetch regions
	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	// fetch instances
	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getComputeInstances(conn, list.Data), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciCompute) getComputeInstancesForRegion(ctx context.Context, computeClient *core.ComputeClient, compartmentID string) ([]core.Instance, error) {
	instances := []core.Instance{}
	var page *string
	for {
		request := core.ListInstancesRequest{
			CompartmentId: common.String(compartmentID),
			Page:          page,
		}

		response, err := computeClient.ListInstances(ctx, request)
		if err != nil {
			return nil, err
		}

		instances = append(instances, response.Items...)

		if response.OpcNextPage == nil {
			break
		}

		page = response.OpcNextPage
	}

	return instances, nil
}

func (o *mqlOciCompute) getComputeInstances(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}

		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionResource.Id.Data)

			svc, err := conn.ComputeClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var res []any
			instances, err := o.getComputeInstancesForRegion(ctx, svc, conn.TenantID())
			if err != nil {
				return nil, err
			}

			for i := range instances {
				instance := instances[i]

				var created *time.Time
				if instance.TimeCreated != nil {
					created = &instance.TimeCreated.Time
				}

				freeformTags := make(map[string]interface{})
				for k, v := range instance.FreeformTags {
					freeformTags[k] = v
				}

				definedTags := make(map[string]interface{})
				for k, v := range instance.DefinedTags {
					definedTags[k] = v
				}

				metadata := make(map[string]interface{})
				for k, v := range instance.Metadata {
					metadata[k] = v
				}

				platformConfig, err := convert.JsonToDict(instance.PlatformConfig)
				if err != nil {
					return nil, err
				}

				launchOptions, err := convert.JsonToDict(instance.LaunchOptions)
				if err != nil {
					return nil, err
				}

				instanceOptions, err := convert.JsonToDict(instance.InstanceOptions)
				if err != nil {
					return nil, err
				}

				shapeConfig, err := convert.JsonToDict(instance.ShapeConfig)
				if err != nil {
					return nil, err
				}

				sourceDetails, err := convert.JsonToDict(instance.SourceDetails)
				if err != nil {
					return nil, err
				}

				// Create compartment resource reference
				compartment, err := CreateResource(o.MqlRuntime, "oci.compartment", map[string]*llx.RawData{
					"id": llx.StringDataPtr(instance.CompartmentId),
				})
				if err != nil {
					return nil, err
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.compute.instance", map[string]*llx.RawData{
					"id":                 llx.StringDataPtr(instance.Id),
					"name":               llx.StringDataPtr(instance.DisplayName),
					"region":             llx.ResourceData(regionResource, "oci.region"),
					"created":            llx.TimeDataPtr(created),
					"state":              llx.StringData(string(instance.LifecycleState)),
					"shape":              llx.StringDataPtr(instance.Shape),
					"availabilityDomain": llx.StringDataPtr(instance.AvailabilityDomain),
					"compartment":        llx.ResourceData(compartment, "oci.compartment"),
					"faultDomain":        llx.StringDataPtr(instance.FaultDomain),
					"imageId":            llx.StringDataPtr(instance.ImageId),
					"dedicatedVmHostId":  llx.StringDataPtr(instance.DedicatedVmHostId),
					"platformConfig":     llx.DictData(platformConfig),
					"launchOptions":      llx.DictData(launchOptions),
					"instanceOptions":    llx.DictData(instanceOptions),
					"shapeConfig":        llx.DictData(shapeConfig),
					"sourceDetails":      llx.DictData(sourceDetails),
					"metadata":           llx.MapData(metadata, types.String),
					"freeformTags":       llx.MapData(freeformTags, types.String),
					"definedTags":        llx.MapData(definedTags, types.Any),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (o *mqlOciComputeInstance) id() (string, error) {
	return "oci.compute.instance/" + o.Id.Data, nil
}

func (o *mqlOciCompute) images() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	// fetch regions
	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	// fetch images
	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getComputeImage(conn, list.Data), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciCompute) getComputeImagesForRegion(ctx context.Context, computeClient *core.ComputeClient, compartmentID string) ([]core.Image, error) {
	images := []core.Image{}
	var page *string
	for {
		request := core.ListImagesRequest{
			CompartmentId: common.String(compartmentID),
			Page:          page,
		}

		response, err := computeClient.ListImages(ctx, request)
		if err != nil {
			return nil, err
		}

		images = append(images, response.Items...)

		if response.OpcNextPage == nil {
			break
		}

		page = response.OpcNextPage
	}

	return images, nil
}

func (o *mqlOciCompute) getComputeImage(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionResource.Id.Data)

			svc, err := conn.ComputeClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var res []any
			images, err := o.getComputeImagesForRegion(ctx, svc, conn.TenantID())
			if err != nil {
				return nil, err
			}

			for i := range images {
				image := images[i]

				var created *time.Time
				if image.TimeCreated != nil {
					created = &image.TimeCreated.Time
				}

				freeformTags := make(map[string]interface{})
				for k, v := range image.FreeformTags {
					freeformTags[k] = v
				}

				definedTags := make(map[string]interface{})
				for k, v := range image.DefinedTags {
					definedTags[k] = v
				}

				var sizeInMBs int64
				if image.SizeInMBs != nil {
					sizeInMBs = *image.SizeInMBs
				}

				// Create compartment resource reference
				compartment, err := CreateResource(o.MqlRuntime, "oci.compartment", map[string]*llx.RawData{
					"id": llx.StringDataPtr(image.CompartmentId),
				})
				if err != nil {
					return nil, err
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.compute.image", map[string]*llx.RawData{
					"id":                     llx.StringDataPtr(image.Id),
					"name":                   llx.StringDataPtr(image.DisplayName),
					"region":                 llx.ResourceData(regionResource, "oci.region"),
					"created":                llx.TimeDataPtr(created),
					"state":                  llx.StringData(string(image.LifecycleState)),
					"compartment":            llx.ResourceData(compartment, "oci.compartment"),
					"operatingSystem":        llx.StringDataPtr(image.OperatingSystem),
					"operatingSystemVersion": llx.StringDataPtr(image.OperatingSystemVersion),
					"sizeInMBs":              llx.IntData(sizeInMBs),
					"freeformTags":           llx.MapData(freeformTags, types.String),
					"definedTags":            llx.MapData(definedTags, types.Any),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (o *mqlOciComputeImage) id() (string, error) {
	return "oci.compute.image/" + o.Id.Data, nil
}

func (o *mqlOciCompute) blockVolumes() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getBlockVolumes(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciCompute) getBlockVolumesForRegion(ctx context.Context, client *core.BlockstorageClient, compartmentID string) ([]core.Volume, error) {
	volumes := []core.Volume{}
	var page *string
	for {
		request := core.ListVolumesRequest{
			CompartmentId: common.String(compartmentID),
			Page:          page,
		}

		response, err := client.ListVolumes(ctx, request)
		if err != nil {
			return nil, err
		}

		volumes = append(volumes, response.Items...)

		if response.OpcNextPage == nil {
			break
		}

		page = response.OpcNextPage
	}

	return volumes, nil
}

func (o *mqlOciCompute) getBlockVolumes(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionResource.Id.Data)

			svc, err := conn.BlockstorageClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var res []any
			volumes, err := o.getBlockVolumesForRegion(ctx, svc, conn.TenantID())
			if err != nil {
				return nil, err
			}

			for i := range volumes {
				vol := volumes[i]

				var created *time.Time
				if vol.TimeCreated != nil {
					created = &vol.TimeCreated.Time
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.compute.blockVolume", map[string]*llx.RawData{
					"id":                 llx.StringDataPtr(vol.Id),
					"name":               llx.StringDataPtr(vol.DisplayName),
					"compartmentID":      llx.StringDataPtr(vol.CompartmentId),
					"availabilityDomain": llx.StringDataPtr(vol.AvailabilityDomain),
					"sizeInGBs":          llx.IntDataPtr(vol.SizeInGBs),
					"vpusPerGB":          llx.IntDataPtr(vol.VpusPerGB),
					"state":              llx.StringData(string(vol.LifecycleState)),
					"isHydrated":         llx.BoolDataPtr(vol.IsHydrated),
					"kmsKeyId":           llx.StringDataPtr(vol.KmsKeyId),
					"isAutoTuneEnabled":  llx.BoolDataPtr(vol.IsAutoTuneEnabled),
					"created":            llx.TimeDataPtr(created),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (o *mqlOciComputeBlockVolume) id() (string, error) {
	return "oci.compute.blockVolume/" + o.Id.Data, nil
}

func (o *mqlOciCompute) bootVolumes() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getBootVolumes(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciCompute) getBootVolumesForRegion(ctx context.Context, client *core.BlockstorageClient, compartmentID string) ([]core.BootVolume, error) {
	bootVolumes := []core.BootVolume{}
	var page *string
	for {
		request := core.ListBootVolumesRequest{
			CompartmentId: common.String(compartmentID),
			Page:          page,
		}

		response, err := client.ListBootVolumes(ctx, request)
		if err != nil {
			return nil, err
		}

		bootVolumes = append(bootVolumes, response.Items...)

		if response.OpcNextPage == nil {
			break
		}

		page = response.OpcNextPage
	}

	return bootVolumes, nil
}

func (o *mqlOciCompute) getBootVolumes(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionResource.Id.Data)

			svc, err := conn.BlockstorageClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var res []any
			bootVols, err := o.getBootVolumesForRegion(ctx, svc, conn.TenantID())
			if err != nil {
				return nil, err
			}

			for i := range bootVols {
				bv := bootVols[i]

				var created *time.Time
				if bv.TimeCreated != nil {
					created = &bv.TimeCreated.Time
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.compute.bootVolume", map[string]*llx.RawData{
					"id":                 llx.StringDataPtr(bv.Id),
					"name":               llx.StringDataPtr(bv.DisplayName),
					"compartmentID":      llx.StringDataPtr(bv.CompartmentId),
					"availabilityDomain": llx.StringDataPtr(bv.AvailabilityDomain),
					"sizeInGBs":          llx.IntDataPtr(bv.SizeInGBs),
					"imageId":            llx.StringDataPtr(bv.ImageId),
					"state":              llx.StringData(string(bv.LifecycleState)),
					"kmsKeyId":           llx.StringDataPtr(bv.KmsKeyId),
					"created":            llx.TimeDataPtr(created),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (o *mqlOciComputeBootVolume) id() (string, error) {
	return "oci.compute.bootVolume/" + o.Id.Data, nil
}
