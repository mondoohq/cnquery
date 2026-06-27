// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/oracle/oci-go-sdk/v65/bastion"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciBastion) id() (string, error) {
	return "oci.bastion", nil
}

func (o *mqlOciBastion) bastions() ([]any, error) {
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
	poolOfJobs := jobpool.CreatePool(o.getBastions(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciBastion) getBastions(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci bastion with region %s", regionResource.Id.Data)

			svc, err := conn.BastionClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			bastions := []bastion.BastionSummary{}
			var page *string
			for {
				response, err := svc.ListBastions(ctx, bastion.ListBastionsRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				bastions = append(bastions, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range bastions {
				b := bastions[i]

				var created *time.Time
				if b.TimeCreated != nil {
					created = &b.TimeCreated.Time
				}
				var timeUpdated *time.Time
				if b.TimeUpdated != nil {
					timeUpdated = &b.TimeUpdated.Time
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.bastion.instance", map[string]*llx.RawData{
					"id":             llx.StringDataPtr(b.Id),
					"name":           llx.StringDataPtr(b.Name),
					"compartmentID":  llx.StringDataPtr(b.CompartmentId),
					"bastionType":    llx.StringDataPtr(b.BastionType),
					"state":          llx.StringData(string(b.LifecycleState)),
					"dnsProxyStatus": llx.StringData(string(b.DnsProxyStatus)),
					"created":        llx.TimeDataPtr(created),
					"timeUpdated":    llx.TimeDataPtr(timeUpdated),
					"systemTags":     llx.MapData(definedTagsToAny(b.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				mqlB := mqlInstance.(*mqlOciBastionInstance)
				mqlB.cacheTargetVcnId = stringValue(b.TargetVcnId)
				mqlB.cacheTargetSubnetId = stringValue(b.TargetSubnetId)
				res = append(res, mqlB)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciBastionInstanceInternal struct {
	cacheTargetVcnId    string
	cacheTargetSubnetId string
}

func (o *mqlOciBastionInstance) id() (string, error) {
	return "oci.bastion.instance/" + o.Id.Data, nil
}

func (o *mqlOciBastionInstance) targetSubnet() (*mqlOciNetworkSubnet, error) {
	if o.cacheTargetSubnetId == "" {
		o.TargetSubnet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlSubnet, err := NewResource(o.MqlRuntime, "oci.network.subnet", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheTargetSubnetId),
	})
	if err != nil {
		return nil, err
	}
	return mqlSubnet.(*mqlOciNetworkSubnet), nil
}

func (o *mqlOciBastionInstance) targetVcn() (*mqlOciNetworkVcn, error) {
	if o.cacheTargetVcnId == "" {
		o.TargetVcn.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlVcn, err := NewResource(o.MqlRuntime, "oci.network.vcn", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheTargetVcnId),
	})
	if err != nil {
		return nil, err
	}
	return mqlVcn.(*mqlOciNetworkVcn), nil
}
