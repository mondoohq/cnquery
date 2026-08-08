// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
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

	return ociRunRegionPool(o.getBastions(conn, list.Data))
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

			bastions, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]bastion.BastionSummary, *string, error) {
				response, err := svc.ListBastions(ctx, bastion.ListBastionsRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, nil, err
				}
				return response.Items, response.OpcNextPage, nil
			})
			if err != nil {
				return nil, err
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
				mqlB.region = regionResource.Id.Data
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
	region              string

	lock    sync.Mutex
	fetched atomic.Bool
	bastion *bastion.Bastion
}

func (o *mqlOciBastionInstance) id() (string, error) {
	return "oci.bastion.instance/" + o.Id.Data, nil
}

// getBastionDetails lazily fetches the full bastion. ListBastions returns a
// summary that omits the session-control settings - most importantly the client
// CIDR allow list, which decides who on the internet may open a session - so
// they are only reachable through this call. Five accessors share it, and the
// runtime resolves them concurrently, so the fetch is guarded.
func (o *mqlOciBastionInstance) getBastionDetails() (*bastion.Bastion, error) {
	if o.fetched.Load() {
		return o.bastion, nil
	}
	o.lock.Lock()
	defer o.lock.Unlock()
	if o.fetched.Load() {
		return o.bastion, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	region := o.region
	if region == "" {
		region = ociRegionFromOCID(o.Id.Data)
	}
	svc, err := conn.BastionClient(region)
	if err != nil {
		return nil, err
	}
	resp, err := svc.GetBastion(context.Background(), bastion.GetBastionRequest{
		BastionId: common.String(o.Id.Data),
	})
	if err != nil {
		return nil, err
	}

	o.bastion = &resp.Bastion
	o.fetched.Store(true)
	return o.bastion, nil
}

func (o *mqlOciBastionInstance) clientCidrBlockAllowList() ([]any, error) {
	b, err := o.getBastionDetails()
	if err != nil {
		return nil, err
	}
	return stringsToAny(b.ClientCidrBlockAllowList), nil
}

func (o *mqlOciBastionInstance) staticJumpHostIpAddresses() ([]any, error) {
	b, err := o.getBastionDetails()
	if err != nil {
		return nil, err
	}
	return stringsToAny(b.StaticJumpHostIpAddresses), nil
}

func (o *mqlOciBastionInstance) maxSessionTtlInSeconds() (int64, error) {
	b, err := o.getBastionDetails()
	if err != nil {
		return 0, err
	}
	if b.MaxSessionTtlInSeconds == nil {
		o.MaxSessionTtlInSeconds.State = plugin.StateIsSet | plugin.StateIsNull
		return 0, nil
	}
	return int64(*b.MaxSessionTtlInSeconds), nil
}

func (o *mqlOciBastionInstance) maxSessionsAllowed() (int64, error) {
	b, err := o.getBastionDetails()
	if err != nil {
		return 0, err
	}
	if b.MaxSessionsAllowed == nil {
		o.MaxSessionsAllowed.State = plugin.StateIsSet | plugin.StateIsNull
		return 0, nil
	}
	return int64(*b.MaxSessionsAllowed), nil
}

func (o *mqlOciBastionInstance) privateEndpointIpAddress() (string, error) {
	b, err := o.getBastionDetails()
	if err != nil {
		return "", err
	}
	if b.PrivateEndpointIpAddress == nil {
		o.PrivateEndpointIpAddress.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}
	return *b.PrivateEndpointIpAddress, nil
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
