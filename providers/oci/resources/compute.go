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
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/oci/connection"
)

func (e *mqlOciCompute) id() (string, error) {
	return "oci.compute", nil
}

func (o *mqlOciCompute) instances() ([]interface{}, error) {
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
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(o.getComputeInstances(conn, list.Data), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
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

func (o *mqlOciCompute) getComputeInstances(conn *connection.OciConnection, regions []interface{}) []*jobpool.Job {
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

			var res []interface{}
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

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.compute.instance", map[string]*llx.RawData{
					"id":      llx.StringDataPtr(instance.Id),
					"name":    llx.StringDataPtr(instance.DisplayName),
					"region":  llx.ResourceData(regionResource, "oci.region"),
					"created": llx.TimeDataPtr(created),
					"state":   llx.StringData(string(instance.LifecycleState)),
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

func (o *mqlOciCompute) images() ([]interface{}, error) {
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
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(o.getComputeImage(conn, list.Data), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
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

func (o *mqlOciCompute) getComputeImage(conn *connection.OciConnection, regions []interface{}) []*jobpool.Job {
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

			var res []interface{}
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

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.compute.image", map[string]*llx.RawData{
					"id":      llx.StringDataPtr(image.Id),
					"name":    llx.StringDataPtr(image.DisplayName),
					"region":  llx.ResourceData(regionResource, "oci.region"),
					"created": llx.TimeDataPtr(created),
					"state":   llx.StringData(string(image.LifecycleState)),
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
