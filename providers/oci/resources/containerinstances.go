// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/containerinstances"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciContainerInstances) id() (string, error) {
	return "oci.containerInstances", nil
}

func (o *mqlOciContainerInstances) instances() ([]any, error) {
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

	return ociRunRegionPool(o.getContainerInstances(conn, list.Data))
}

func (o *mqlOciContainerInstances) getContainerInstances(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci container instances with region %s", regionResource.Id.Data)

			svc, err := conn.ContainerInstanceClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var items []containerinstances.ContainerInstanceSummary
			var page *string
			for {
				response, err := svc.ListContainerInstances(ctx, containerinstances.ListContainerInstancesRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				items = append(items, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range items {
				ci := items[i]

				var created *time.Time
				if ci.TimeCreated != nil {
					created = &ci.TimeCreated.Time
				}
				var timeUpdated *time.Time
				if ci.TimeUpdated != nil {
					timeUpdated = &ci.TimeUpdated.Time
				}

				shapeConfig, err := convert.JsonToDict(ci.ShapeConfig)
				if err != nil {
					return nil, err
				}

				freeformTags := make(map[string]interface{}, len(ci.FreeformTags))
				for k, v := range ci.FreeformTags {
					freeformTags[k] = v
				}
				definedTags := make(map[string]interface{}, len(ci.DefinedTags))
				for k, v := range ci.DefinedTags {
					definedTags[k] = v
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.containerInstances.instance", map[string]*llx.RawData{
					"id":                               llx.StringDataPtr(ci.Id),
					"name":                             llx.StringDataPtr(ci.DisplayName),
					"compartmentID":                    llx.StringDataPtr(ci.CompartmentId),
					"availabilityDomain":               llx.StringDataPtr(ci.AvailabilityDomain),
					"state":                            llx.StringData(string(ci.LifecycleState)),
					"shape":                            llx.StringDataPtr(ci.Shape),
					"shapeConfig":                      llx.DictData(shapeConfig),
					"containerCount":                   llx.IntData(intValue(ci.ContainerCount)),
					"containerRestartPolicy":           llx.StringData(string(ci.ContainerRestartPolicy)),
					"faultDomain":                      llx.StringDataPtr(ci.FaultDomain),
					"gracefulShutdownTimeoutInSeconds": llx.IntData(int64Value(ci.GracefulShutdownTimeoutInSeconds)),
					"volumeCount":                      llx.IntData(intValue(ci.VolumeCount)),
					"created":                          llx.TimeDataPtr(created),
					"timeUpdated":                      llx.TimeDataPtr(timeUpdated),
					"freeformTags":                     llx.MapData(freeformTags, types.String),
					"definedTags":                      llx.MapData(definedTags, types.Any),
					"systemTags":                       llx.MapData(definedTagsToAny(ci.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				mqlCI := mqlInstance.(*mqlOciContainerInstancesInstance)
				mqlCI.region = regionResource.Id.Data
				res = append(res, mqlCI)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciContainerInstancesInstanceInternal struct {
	region string
}

func (o *mqlOciContainerInstancesInstance) id() (string, error) {
	return "oci.containerInstances.instance/" + o.Id.Data, nil
}

func (o *mqlOciContainerInstancesInstance) containers() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	ctx := context.Background()

	svc, err := conn.ContainerInstanceClient(o.region)
	if err != nil {
		return nil, err
	}

	var items []containerinstances.ContainerSummary
	var page *string
	for {
		response, err := svc.ListContainers(ctx, containerinstances.ListContainersRequest{
			CompartmentId:       common.String(o.CompartmentID.Data),
			ContainerInstanceId: common.String(o.Id.Data),
			Page:                page,
		})
		if err != nil {
			return nil, err
		}

		items = append(items, response.Items...)

		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	res := make([]any, 0, len(items))
	for i := range items {
		c := items[i]

		var created *time.Time
		if c.TimeCreated != nil {
			created = &c.TimeCreated.Time
		}
		var timeUpdated *time.Time
		if c.TimeUpdated != nil {
			timeUpdated = &c.TimeUpdated.Time
		}
		resourceConfig, err := convert.JsonToDict(c.ResourceConfig)
		if err != nil {
			return nil, err
		}

		freeformTags := make(map[string]interface{}, len(c.FreeformTags))
		for k, v := range c.FreeformTags {
			freeformTags[k] = v
		}
		definedTags := make(map[string]interface{}, len(c.DefinedTags))
		for k, v := range c.DefinedTags {
			definedTags[k] = v
		}

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.containerInstances.container", map[string]*llx.RawData{
			"id":                          llx.StringDataPtr(c.Id),
			"name":                        llx.StringDataPtr(c.DisplayName),
			"compartmentID":               llx.StringDataPtr(c.CompartmentId),
			"availabilityDomain":          llx.StringDataPtr(c.AvailabilityDomain),
			"state":                       llx.StringData(string(c.LifecycleState)),
			"containerInstanceId":         llx.StringDataPtr(c.ContainerInstanceId),
			"imageUrl":                    llx.StringDataPtr(c.ImageUrl),
			"isResourcePrincipalDisabled": llx.BoolDataPtr(c.IsResourcePrincipalDisabled),
			"resourceConfig":              llx.DictData(resourceConfig),
			"created":                     llx.TimeDataPtr(created),
			"timeUpdated":                 llx.TimeDataPtr(timeUpdated),
			"freeformTags":                llx.MapData(freeformTags, types.String),
			"definedTags":                 llx.MapData(definedTags, types.Any),
			"systemTags":                  llx.MapData(definedTagsToAny(c.SystemTags), types.Dict),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (o *mqlOciContainerInstancesContainer) id() (string, error) {
	return "oci.containerInstances.container/" + o.Id.Data, nil
}
