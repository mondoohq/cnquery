package oci

import (
	"context"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/rs/zerolog/log"
	oci_provider "go.mondoo.com/cnquery/motor/providers/oci"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	corePack "go.mondoo.com/cnquery/resources/packs/core"
)

func (e *mqlOciCompute) id() (string, error) {
	return "oci.compute", nil
}

func (o *mqlOciCompute) GetInstances() ([]interface{}, error) {
	provider, err := ociProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(o.getComputeInstances(provider), 5)
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

func (o *mqlOciCompute) getComputeInstances(provider *oci_provider.Provider) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions(ctx)
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionVal)

			svc, err := provider.ComputeClient(*regionVal.RegionKey)
			if err != nil {
				return nil, err
			}

			var res []interface{}
			instances, err := o.getComputeInstancesForRegion(ctx, svc, provider.TenantID())
			if err != nil {
				return nil, err
			}

			for i := range instances {
				instance := instances[i]

				var created *time.Time
				if instance.TimeCreated != nil {
					created = &instance.TimeCreated.Time
				}

				mqlInstance, err := o.MotorRuntime.CreateResource("oci.compute.instance",
					"id", corePack.ToString(instance.Id),
					"name", corePack.ToString(instance.DisplayName),
					"region", region,
					"created", created,
					"lifecycleState", string(instance.LifecycleState),
				)
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
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "oci.compute.instance/" + id, nil
}

func (o *mqlOciCompute) GetImages() ([]interface{}, error) {
	provider, err := ociProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(o.getComputeInstances(provider), 5)
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

func (o *mqlOciCompute) getComputeImage(provider *oci_provider.Provider) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions(ctx)
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionVal)

			svc, err := provider.ComputeClient(*regionVal.RegionKey)
			if err != nil {
				return nil, err
			}

			var res []interface{}
			images, err := o.getComputeInstancesForRegion(ctx, svc, provider.TenantID())
			if err != nil {
				return nil, err
			}

			for i := range images {
				image := images[i]

				var created *time.Time
				if image.TimeCreated != nil {
					created = &image.TimeCreated.Time
				}

				mqlInstance, err := o.MotorRuntime.CreateResource("oci.compute.image",
					"id", corePack.ToString(image.Id),
					"name", corePack.ToString(image.DisplayName),
					"region", region,
					"created", created,
					"lifecycleState", string(image.LifecycleState),
				)
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
	return "oci.compute.image", nil
}
