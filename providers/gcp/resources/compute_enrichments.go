// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strconv"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

// Health checks

func (g *mqlGcpProjectComputeService) healthChecks() ([]any, error) {
	if !g.GetEnabled().Data {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	client, err := conn.Client(compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var res []any
	req := computeSvc.HealthChecks.AggregatedList(projectId)
	if err := req.Pages(ctx, func(page *compute.HealthChecksAggregatedList) error {
		for _, scoped := range page.Items {
			for _, hc := range scoped.HealthChecks {
				httpHC, _ := convert.JsonToDict(hc.HttpHealthCheck)
				httpsHC, _ := convert.JsonToDict(hc.HttpsHealthCheck)
				tcpHC, _ := convert.JsonToDict(hc.TcpHealthCheck)
				sslHC, _ := convert.JsonToDict(hc.SslHealthCheck)
				http2HC, _ := convert.JsonToDict(hc.Http2HealthCheck)
				grpcHC, _ := convert.JsonToDict(hc.GrpcHealthCheck)
				logCfg, _ := convert.JsonToDict(hc.LogConfig)

				mqlHC, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.healthCheck", map[string]*llx.RawData{
					"id":                 llx.StringData(strconv.FormatUint(hc.Id, 10)),
					"projectId":          llx.StringData(projectId),
					"name":               llx.StringData(hc.Name),
					"description":        llx.StringData(hc.Description),
					"type":               llx.StringData(hc.Type),
					"checkIntervalSec":   llx.IntData(hc.CheckIntervalSec),
					"timeoutSec":         llx.IntData(hc.TimeoutSec),
					"healthyThreshold":   llx.IntData(hc.HealthyThreshold),
					"unhealthyThreshold": llx.IntData(hc.UnhealthyThreshold),
					"created":            llx.TimeDataPtr(parseTime(hc.CreationTimestamp)),
					"selfLink":           llx.StringData(hc.SelfLink),
					"httpHealthCheck":    llx.DictData(httpHC),
					"httpsHealthCheck":   llx.DictData(httpsHC),
					"tcpHealthCheck":     llx.DictData(tcpHC),
					"sslHealthCheck":     llx.DictData(sslHC),
					"http2HealthCheck":   llx.DictData(http2HC),
					"grpcHealthCheck":    llx.DictData(grpcHC),
					"logConfig":          llx.DictData(logCfg),
					"regionUrl":          llx.StringData(hc.Region),
				})
				if err != nil {
					return err
				}
				res = append(res, mqlHC)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return res, nil
}

func (g *mqlGcpProjectComputeServiceHealthCheck) id() (string, error) {
	return "gcloud.compute.healthCheck/" + g.Id.Data, g.Id.Error
}

// URL maps

func (g *mqlGcpProjectComputeService) urlMaps() ([]any, error) {
	if !g.GetEnabled().Data {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	client, err := conn.Client(compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var res []any
	req := computeSvc.UrlMaps.AggregatedList(projectId)
	if err := req.Pages(ctx, func(page *compute.UrlMapsAggregatedList) error {
		for _, scoped := range page.Items {
			for _, um := range scoped.UrlMaps {
				hostRules, _ := convert.JsonToDictSlice(um.HostRules)
				pathMatchers, _ := convert.JsonToDictSlice(um.PathMatchers)
				tests, _ := convert.JsonToDictSlice(um.Tests)

				mqlUM, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.urlMap", map[string]*llx.RawData{
					"id":             llx.StringData(strconv.FormatUint(um.Id, 10)),
					"projectId":      llx.StringData(projectId),
					"name":           llx.StringData(um.Name),
					"description":    llx.StringData(um.Description),
					"defaultService": llx.StringData(um.DefaultService),
					"hostRules":      llx.ArrayData(hostRules, types.Dict),
					"pathMatchers":   llx.ArrayData(pathMatchers, types.Dict),
					"tests":          llx.ArrayData(tests, types.Dict),
					"created":        llx.TimeDataPtr(parseTime(um.CreationTimestamp)),
					"selfLink":       llx.StringData(um.SelfLink),
					"regionUrl":      llx.StringData(um.Region),
				})
				if err != nil {
					return err
				}
				res = append(res, mqlUM)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return res, nil
}

func (g *mqlGcpProjectComputeServiceUrlMap) id() (string, error) {
	return "gcloud.compute.urlMap/" + g.Id.Data, g.Id.Error
}

// Target HTTP proxies

func (g *mqlGcpProjectComputeService) targetHttpProxies() ([]any, error) {
	if !g.GetEnabled().Data {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	client, err := conn.Client(compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var res []any
	req := computeSvc.TargetHttpProxies.AggregatedList(projectId)
	if err := req.Pages(ctx, func(page *compute.TargetHttpProxyAggregatedList) error {
		for _, scoped := range page.Items {
			for _, proxy := range scoped.TargetHttpProxies {
				mqlProxy, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.targetHttpProxy", map[string]*llx.RawData{
					"id":        llx.StringData(strconv.FormatUint(proxy.Id, 10)),
					"projectId": llx.StringData(projectId),
					"name":      llx.StringData(proxy.Name),
					"description": llx.StringData(proxy.Description),
					"urlMapUrl": llx.StringData(proxy.UrlMap),
					"created":   llx.TimeDataPtr(parseTime(proxy.CreationTimestamp)),
					"selfLink":  llx.StringData(proxy.SelfLink),
					"proxyBind": llx.BoolData(proxy.ProxyBind),
					"regionUrl": llx.StringData(proxy.Region),
				})
				if err != nil {
					return err
				}
				res = append(res, mqlProxy)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return res, nil
}

func (g *mqlGcpProjectComputeServiceTargetHttpProxy) id() (string, error) {
	return "gcloud.compute.targetHttpProxy/" + g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectComputeServiceTargetHttpProxy) urlMap() (*mqlGcpProjectComputeServiceUrlMap, error) {
	if g.UrlMapUrl.Error != nil {
		return nil, g.UrlMapUrl.Error
	}
	url := g.UrlMapUrl.Data
	if url == "" {
		g.UrlMap.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	um, err := getUrlMapByUrl(url, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if um == nil {
		g.UrlMap.State = plugin.StateIsNull | plugin.StateIsSet
	}
	return um, nil
}

// Target HTTPS proxies

func (g *mqlGcpProjectComputeService) targetHttpsProxies() ([]any, error) {
	if !g.GetEnabled().Data {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	client, err := conn.Client(compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var res []any
	req := computeSvc.TargetHttpsProxies.AggregatedList(projectId)
	if err := req.Pages(ctx, func(page *compute.TargetHttpsProxyAggregatedList) error {
		for _, scoped := range page.Items {
			for _, proxy := range scoped.TargetHttpsProxies {
				mqlProxy, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.targetHttpsProxy", map[string]*llx.RawData{
					"id":                 llx.StringData(strconv.FormatUint(proxy.Id, 10)),
					"projectId":          llx.StringData(projectId),
					"name":               llx.StringData(proxy.Name),
					"description":        llx.StringData(proxy.Description),
					"urlMapUrl":          llx.StringData(proxy.UrlMap),
					"sslCertificateUrls": llx.ArrayData(convert.SliceAnyToInterface(proxy.SslCertificates), types.String),
					"sslPolicyUrl":       llx.StringData(proxy.SslPolicy),
					"quicOverride":       llx.StringData(proxy.QuicOverride),
					"created":            llx.TimeDataPtr(parseTime(proxy.CreationTimestamp)),
					"selfLink":           llx.StringData(proxy.SelfLink),
					"proxyBind":          llx.BoolData(proxy.ProxyBind),
					"regionUrl":          llx.StringData(proxy.Region),
				})
				if err != nil {
					return err
				}
				res = append(res, mqlProxy)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return res, nil
}

func (g *mqlGcpProjectComputeServiceTargetHttpsProxy) id() (string, error) {
	return "gcloud.compute.targetHttpsProxy/" + g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectComputeServiceTargetHttpsProxy) urlMap() (*mqlGcpProjectComputeServiceUrlMap, error) {
	if g.UrlMapUrl.Error != nil {
		return nil, g.UrlMapUrl.Error
	}
	url := g.UrlMapUrl.Data
	if url == "" {
		g.UrlMap.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	um, err := getUrlMapByUrl(url, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if um == nil {
		g.UrlMap.State = plugin.StateIsNull | plugin.StateIsSet
	}
	return um, nil
}

func (g *mqlGcpProjectComputeServiceTargetHttpsProxy) sslPolicy() (*mqlGcpProjectComputeServiceSslPolicy, error) {
	if g.SslPolicyUrl.Error != nil {
		return nil, g.SslPolicyUrl.Error
	}
	url := g.SslPolicyUrl.Data
	if url == "" {
		g.SslPolicy.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return getSslPolicyByUrl(url, g.MqlRuntime)
}

// Helper to resolve URL map references

func getUrlMapByUrl(urlMapUrl string, runtime *plugin.Runtime) (*mqlGcpProjectComputeServiceUrlMap, error) {
	if urlMapUrl == "" {
		return nil, nil
	}
	// Format: https://www.googleapis.com/compute/v1/projects/{project}/global/urlMaps/{name}
	// or regional: .../projects/{project}/regions/{region}/urlMaps/{name}
	name := parseResourceName(urlMapUrl)
	// Extract project from URL
	params := trimComputeURL(urlMapUrl)
	parts := strings.Split(params, "/")
	if len(parts) < 2 {
		return nil, nil
	}
	projectId := parts[1]

	res, err := CreateResource(runtime, "gcp.project.computeService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}
	svc := res.(*mqlGcpProjectComputeService)
	urlMaps := svc.GetUrlMaps()
	if urlMaps.Error != nil {
		return nil, urlMaps.Error
	}
	for _, u := range urlMaps.Data {
		um := u.(*mqlGcpProjectComputeServiceUrlMap)
		if um.Name.Data == name {
			return um, nil
		}
	}
	return nil, nil
}

func getSslPolicyByUrl(sslPolicyUrl string, runtime *plugin.Runtime) (*mqlGcpProjectComputeServiceSslPolicy, error) {
	if sslPolicyUrl == "" {
		return nil, nil
	}
	name := parseResourceName(sslPolicyUrl)
	params := trimComputeURL(sslPolicyUrl)
	parts := strings.Split(params, "/")
	if len(parts) < 2 {
		return nil, nil
	}
	projectId := parts[1]

	res, err := CreateResource(runtime, "gcp.project.computeService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}
	svc := res.(*mqlGcpProjectComputeService)
	sslPolicies := svc.GetSslPolicies()
	if sslPolicies.Error != nil {
		return nil, sslPolicies.Error
	}
	for _, s := range sslPolicies.Data {
		sp := s.(*mqlGcpProjectComputeServiceSslPolicy)
		if sp.Name.Data == name {
			return sp, nil
		}
	}
	return nil, nil
}

func trimComputeURL(url string) string {
	url = strings.TrimPrefix(url, "https://www.googleapis.com/compute/v1/")
	url = strings.TrimPrefix(url, "https://compute.googleapis.com/compute/v1/")
	return url
}
