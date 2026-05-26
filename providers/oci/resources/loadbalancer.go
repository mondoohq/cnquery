// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/loadbalancer"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciLoadBalancer) id() (string, error) {
	return "oci.loadBalancer", nil
}

func (o *mqlOciLoadBalancer) loadBalancers() ([]any, error) {
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
	poolOfJobs := jobpool.CreatePool(o.getLoadBalancers(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciLoadBalancer) getLoadBalancers(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci load balancer with region %s", regionResource.Id.Data)

			svc, err := conn.LoadBalancerClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			lbs := []loadbalancer.LoadBalancer{}
			var page *string
			for {
				response, err := svc.ListLoadBalancers(ctx, loadbalancer.ListLoadBalancersRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				lbs = append(lbs, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range lbs {
				lb := lbs[i]

				var created *time.Time
				if lb.TimeCreated != nil {
					created = &lb.TimeCreated.Time
				}

				freeformTags := make(map[string]interface{}, len(lb.FreeformTags))
				for k, v := range lb.FreeformTags {
					freeformTags[k] = v
				}
				definedTags := make(map[string]interface{}, len(lb.DefinedTags))
				for k, v := range lb.DefinedTags {
					definedTags[k] = v
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.loadBalancer.loadBalancer", map[string]*llx.RawData{
					"id":                        llx.StringDataPtr(lb.Id),
					"name":                      llx.StringDataPtr(lb.DisplayName),
					"compartmentID":             llx.StringDataPtr(lb.CompartmentId),
					"shape":                     llx.StringDataPtr(lb.ShapeName),
					"isPrivate":                 llx.BoolDataPtr(lb.IsPrivate),
					"isDeleteProtectionEnabled": llx.BoolDataPtr(lb.IsDeleteProtectionEnabled),
					"state":                     llx.StringData(string(lb.LifecycleState)),
					"created":                   llx.TimeDataPtr(created),
					"freeformTags":              llx.MapData(freeformTags, types.String),
					"definedTags":               llx.MapData(definedTags, types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlLb := mqlInstance.(*mqlOciLoadBalancerLoadBalancer)
				mqlLb.cacheListeners = lb.Listeners
				mqlLb.cacheBackendSets = lb.BackendSets
				mqlLb.cacheRegion = regionResource.Id.Data
				res = append(res, mqlLb)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciLoadBalancerLoadBalancerInternal struct {
	cacheListeners   map[string]loadbalancer.Listener
	cacheBackendSets map[string]loadbalancer.BackendSet
	cacheRegion      string
}

func initOciLoadBalancerLoadBalancer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	idVal := ociArgString(args, "id")
	if idVal == "" {
		conn := runtime.Connection.(*connection.OciConnection)
		if conn.Conf == nil || conn.Conf.PlatformId == "" {
			return args, nil, nil
		}
		parsed, ok := parseOciObjectPlatformID(conn.Conf.PlatformId)
		if !ok || parsed.service != "loadbalancer" || parsed.objectType != "loadBalancer" {
			return args, nil, nil
		}
		idVal = parsed.id
	}

	obj, err := CreateResource(runtime, "oci.loadBalancer", nil)
	if err != nil {
		return nil, nil, err
	}
	lb := obj.(*mqlOciLoadBalancer)

	rawLBs := lb.GetLoadBalancers()
	if rawLBs.Error != nil {
		return nil, nil, rawLBs.Error
	}

	for _, raw := range rawLBs.Data {
		l := raw.(*mqlOciLoadBalancerLoadBalancer)
		if l.Id.Data == idVal {
			return args, l, nil
		}
	}

	return nil, nil, errors.New("oci.loadBalancer.loadBalancer not found: " + idVal)
}

func (o *mqlOciLoadBalancerLoadBalancer) id() (string, error) {
	return "oci.loadBalancer.loadBalancer/" + o.Id.Data, nil
}

func (o *mqlOciLoadBalancerLoadBalancer) listeners() ([]any, error) {
	res := []any{}
	for name, listener := range o.cacheListeners {
		var sslProtocols []any
		var sslCipherSuiteName string
		var sslVerifyPeerCert bool
		if listener.SslConfiguration != nil {
			for _, p := range listener.SslConfiguration.Protocols {
				sslProtocols = append(sslProtocols, p)
			}
			sslCipherSuiteName = stringValue(listener.SslConfiguration.CipherSuiteName)
			sslVerifyPeerCert = boolValue(listener.SslConfiguration.VerifyPeerCertificate)
		}

		lbId := o.Id.Data
		listenerId := lbId + "/listener/" + name
		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.loadBalancer.listener", map[string]*llx.RawData{
			"__id":                     llx.StringData(listenerId),
			"name":                     llx.StringData(name),
			"port":                     llx.IntDataPtr(listener.Port),
			"protocol":                 llx.StringDataPtr(listener.Protocol),
			"defaultBackendSetName":    llx.StringDataPtr(listener.DefaultBackendSetName),
			"sslProtocols":             llx.ArrayData(sslProtocols, types.String),
			"sslCipherSuiteName":       llx.StringData(sslCipherSuiteName),
			"sslVerifyPeerCertificate": llx.BoolData(sslVerifyPeerCert),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}
	return res, nil
}

func (o *mqlOciLoadBalancerLoadBalancer) backendSets() ([]any, error) {
	res := []any{}
	for name, bs := range o.cacheBackendSets {
		healthChecker, err := convert.JsonToDict(bs.HealthChecker)
		if err != nil {
			return nil, err
		}

		lbId := o.Id.Data
		bsId := lbId + "/backendSet/" + name
		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.loadBalancer.backendSet", map[string]*llx.RawData{
			"__id":          llx.StringData(bsId),
			"name":          llx.StringData(name),
			"policy":        llx.StringDataPtr(bs.Policy),
			"healthChecker": llx.DictData(healthChecker),
			"backendCount":  llx.IntData(int64(len(bs.Backends))),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}
	return res, nil
}

func (o *mqlOciLoadBalancerListener) id() (string, error) {
	return o.__id, nil
}

func (o *mqlOciLoadBalancerBackendSet) id() (string, error) {
	return o.__id, nil
}
