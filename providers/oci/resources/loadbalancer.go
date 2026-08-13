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
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciLoadBalancer) id() (string, error) {
	return "oci.loadBalancer", nil
}

func (o *mqlOciLoadBalancer) loadBalancers() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	// Load balancers belong to the compartment of the application they front, and
	// ListLoadBalancers has no subtree flag. Scanning only the tenancy root meant
	// the exposure surface an internet-facing balancer represents, along with its
	// listener TLS settings and backend sets, never appeared in the inventory at
	// all.
	return ociCollect(o.MqlRuntime, ociScopeAllCompartments,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			log.Debug().Msgf("calling oci load balancer with region %s", region)

			svc, err := conn.LoadBalancerClient(region)
			if err != nil {
				return nil, err
			}

			lbs, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]loadbalancer.LoadBalancer, *string, error) {
				response, err := svc.ListLoadBalancers(ctx, loadbalancer.ListLoadBalancersRequest{
					CompartmentId: common.String(compartmentID),
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
			for i := range lbs {
				lb := lbs[i]

				var created *time.Time
				if lb.TimeCreated != nil {
					created = &lb.TimeCreated.Time
				}

				ipAddresses := make([]any, 0, len(lb.IpAddresses))
				for _, ip := range lb.IpAddresses {
					// isPublic is optional on the SDK model, and exposure()
					// reads this key back to decide internet reachability. A
					// missing flag on a non-private load balancer means public,
					// so falling back to false would clear a genuinely
					// internet-facing balancer.
					entry := map[string]any{
						"ipAddress": stringValue(ip.IpAddress),
						"isPublic":  ociIpIsPublic(ip.IsPublic, boolValue(lb.IsPrivate)),
					}
					if ip.ReservedIp != nil {
						entry["reservedIpId"] = stringValue(ip.ReservedIp.Id)
					}
					ipAddresses = append(ipAddresses, entry)
				}

				mqlInstance, err := createOciResourceInCompartment(o.MqlRuntime, "oci.loadBalancer.loadBalancer", stringValue(lb.CompartmentId), map[string]*llx.RawData{
					"id":                        llx.StringDataPtr(lb.Id),
					"name":                      llx.StringDataPtr(lb.DisplayName),
					"shape":                     llx.StringDataPtr(lb.ShapeName),
					"isPrivate":                 llx.BoolDataPtr(lb.IsPrivate),
					"ipAddresses":               llx.ArrayData(ipAddresses, types.Dict),
					"isDeleteProtectionEnabled": llx.BoolDataPtr(lb.IsDeleteProtectionEnabled),
					"state":                     llx.StringData(string(lb.LifecycleState)),
					"created":                   llx.TimeDataPtr(created),
					"freeformTags":              llx.MapData(strMapToAny(lb.FreeformTags), types.String),
					"definedTags":               llx.MapData(definedTagsToAny(lb.DefinedTags), types.Any),
					"systemTags":                llx.MapData(definedTagsToAny(lb.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				mqlLb := mqlInstance.(*mqlOciLoadBalancerLoadBalancer)
				mqlLb.cacheNsgIDs = convert.SliceAnyToInterface(lb.NetworkSecurityGroupIds)
				mqlLb.cacheListeners = lb.Listeners
				mqlLb.cacheBackendSets = lb.BackendSets
				mqlLb.cacheRegion = region
				mqlLb.cacheSubnetIDs = lb.SubnetIds
				res = append(res, mqlLb)
			}

			return res, nil
		})
}

type mqlOciLoadBalancerLoadBalancerInternal struct {
	ociCompartmentRef
	cacheNsgIDs      []any
	cacheListeners   map[string]loadbalancer.Listener
	cacheBackendSets map[string]loadbalancer.BackendSet
	cacheRegion      string
	cacheSubnetIDs   []string
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
		var sslVerifyPeerCert, hasSessionResumption bool
		var certificateName string
		var certificateIds, trustedCaIds []any
		if listener.SslConfiguration != nil {
			ssl := listener.SslConfiguration
			for _, p := range ssl.Protocols {
				sslProtocols = append(sslProtocols, p)
			}
			sslCipherSuiteName = stringValue(ssl.CipherSuiteName)
			sslVerifyPeerCert = boolValue(ssl.VerifyPeerCertificate)
			hasSessionResumption = boolValue(ssl.HasSessionResumption)
			certificateName = stringValue(ssl.CertificateName)
			certificateIds = convert.SliceAnyToInterface(ssl.CertificateIds)
			trustedCaIds = convert.SliceAnyToInterface(ssl.TrustedCertificateAuthorityIds)
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
			"hasSessionResumption":     llx.BoolData(hasSessionResumption),
			"certificateName":          llx.StringData(certificateName),
		})
		if err != nil {
			return nil, err
		}
		mqlListener := mqlInstance.(*mqlOciLoadBalancerListener)
		mqlListener.cacheCertificateIDs = certificateIds
		mqlListener.cacheTrustedCaIDs = trustedCaIds
		res = append(res, mqlInstance)
	}
	return res, nil
}

type mqlOciLoadBalancerListenerInternal struct {
	cacheCertificateIDs []any
	cacheTrustedCaIDs   []any
}

func (o *mqlOciLoadBalancerListener) certificates() ([]any, error) {
	return resolveOciCertificates(o.MqlRuntime, o.cacheCertificateIDs)
}

func (o *mqlOciLoadBalancerListener) trustedCertificateAuthorities() ([]any, error) {
	return resolveOciCertRefsByType(o.MqlRuntime, o.cacheTrustedCaIDs, "certificateauthority", "oci.certificates.certificateAuthority")
}

func (o *mqlOciLoadBalancerListener) trustedCaBundles() ([]any, error) {
	return resolveOciCertRefsByType(o.MqlRuntime, o.cacheTrustedCaIDs, "cabundle", "oci.certificates.caBundle")
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
