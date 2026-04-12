// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/appmesh"
	appmesh_types "github.com/aws/aws-sdk-go-v2/service/appmesh/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func (a *mqlAwsAppmesh) id() (string, error) {
	return "aws.appmesh", nil
}

func (a *mqlAwsAppmesh) meshes() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getMeshes(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsAppmesh) getMeshes(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("appmesh>getMeshes>calling aws with region %s", region)

			svc := conn.AppMesh(region)
			ctx := context.Background()
			res := []any{}

			paginator := appmesh.NewListMeshesPaginator(svc, &appmesh.ListMeshesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Warn().Str("region", region).Msg("App Mesh is not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, mesh := range page.Meshes {
					mqlMesh, err := CreateResource(a.MqlRuntime, "aws.appmesh.mesh",
						map[string]*llx.RawData{
							"__id":      llx.StringDataPtr(mesh.Arn),
							"arn":       llx.StringDataPtr(mesh.Arn),
							"name":      llx.StringDataPtr(mesh.MeshName),
							"region":    llx.StringData(region),
							"meshOwner": llx.StringDataPtr(mesh.MeshOwner),
							"createdAt": llx.TimeDataPtr(mesh.CreatedAt),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlMesh)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsAppmeshMesh) id() (string, error) {
	return a.Arn.Data, nil
}

type mqlAwsAppmeshMeshInternal struct {
	fetched  bool
	lock     sync.Mutex
	descResp *appmesh.DescribeMeshOutput
}

func (a *mqlAwsAppmeshMesh) fetchDetail() (*appmesh.DescribeMeshOutput, error) {
	if a.fetched {
		return a.descResp, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.descResp, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	svc := conn.AppMesh(region)
	ctx := context.Background()

	name := a.Name.Data
	resp, err := svc.DescribeMesh(ctx, &appmesh.DescribeMeshInput{
		MeshName: &name,
	})
	if err != nil {
		return nil, err
	}
	a.fetched = true
	a.descResp = resp
	return resp, nil
}

func (a *mqlAwsAppmeshMesh) status() (string, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if resp.Mesh.Status == nil {
		return "", nil
	}
	return string(resp.Mesh.Status.Status), nil
}

func (a *mqlAwsAppmeshMesh) egressFilterType() (string, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if resp.Mesh.Spec == nil || resp.Mesh.Spec.EgressFilter == nil {
		return "", nil
	}
	return string(resp.Mesh.Spec.EgressFilter.Type), nil
}

func (a *mqlAwsAppmeshMesh) serviceDiscoveryIpPreference() (string, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if resp.Mesh.Spec == nil || resp.Mesh.Spec.ServiceDiscovery == nil {
		return "", nil
	}
	return string(resp.Mesh.Spec.ServiceDiscovery.IpPreference), nil
}

func (a *mqlAwsAppmeshMesh) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	svc := conn.AppMesh(region)
	ctx := context.Background()

	arn := a.Arn.Data
	tags := make(map[string]any)
	paginator := appmesh.NewListTagsForResourcePaginator(svc, &appmesh.ListTagsForResourceInput{
		ResourceArn: &arn,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, tag := range page.Tags {
			tags[convert.ToValue(tag.Key)] = convert.ToValue(tag.Value)
		}
	}
	return tags, nil
}

func (a *mqlAwsAppmeshMesh) virtualServices() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	svc := conn.AppMesh(region)
	ctx := context.Background()

	meshName := a.Name.Data
	res := []any{}
	paginator := appmesh.NewListVirtualServicesPaginator(svc, &appmesh.ListVirtualServicesInput{
		MeshName: &meshName,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, vs := range page.VirtualServices {
			mqlVs, err := CreateResource(a.MqlRuntime, "aws.appmesh.virtualService",
				map[string]*llx.RawData{
					"__id":     llx.StringDataPtr(vs.Arn),
					"arn":      llx.StringDataPtr(vs.Arn),
					"name":     llx.StringDataPtr(vs.VirtualServiceName),
					"meshName": llx.StringData(meshName),
					"region":   llx.StringData(region),
				})
			if err != nil {
				return nil, err
			}
			internal := mqlVs.(*mqlAwsAppmeshVirtualService)
			internal.region = region
			internal.meshName = meshName
			res = append(res, mqlVs)
		}
	}
	return res, nil
}

func (a *mqlAwsAppmeshMesh) virtualNodes() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	svc := conn.AppMesh(region)
	ctx := context.Background()

	meshName := a.Name.Data
	res := []any{}
	paginator := appmesh.NewListVirtualNodesPaginator(svc, &appmesh.ListVirtualNodesInput{
		MeshName: &meshName,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, vn := range page.VirtualNodes {
			mqlVn, err := CreateResource(a.MqlRuntime, "aws.appmesh.virtualNode",
				map[string]*llx.RawData{
					"__id":     llx.StringDataPtr(vn.Arn),
					"arn":      llx.StringDataPtr(vn.Arn),
					"name":     llx.StringDataPtr(vn.VirtualNodeName),
					"meshName": llx.StringData(meshName),
					"region":   llx.StringData(region),
				})
			if err != nil {
				return nil, err
			}
			internal := mqlVn.(*mqlAwsAppmeshVirtualNode)
			internal.region = region
			internal.meshName = meshName
			res = append(res, mqlVn)
		}
	}
	return res, nil
}

func (a *mqlAwsAppmeshVirtualService) id() (string, error) {
	return a.Arn.Data, nil
}

type mqlAwsAppmeshVirtualServiceInternal struct {
	fetched  bool
	lock     sync.Mutex
	descResp *appmesh.DescribeVirtualServiceOutput
	region   string
	meshName string
}

func (a *mqlAwsAppmeshVirtualService) fetchDetail() (*appmesh.DescribeVirtualServiceOutput, error) {
	if a.fetched {
		return a.descResp, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.descResp, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.AppMesh(a.region)
	ctx := context.Background()

	name := a.Name.Data
	meshName := a.meshName
	resp, err := svc.DescribeVirtualService(ctx, &appmesh.DescribeVirtualServiceInput{
		MeshName:           &meshName,
		VirtualServiceName: &name,
	})
	if err != nil {
		return nil, err
	}
	a.fetched = true
	a.descResp = resp
	return resp, nil
}

func (a *mqlAwsAppmeshVirtualService) status() (string, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if resp.VirtualService == nil || resp.VirtualService.Status == nil {
		return "", nil
	}
	return string(resp.VirtualService.Status.Status), nil
}

func (a *mqlAwsAppmeshVirtualService) providerType() (string, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if resp.VirtualService == nil || resp.VirtualService.Spec == nil || resp.VirtualService.Spec.Provider == nil {
		return "", nil
	}
	switch resp.VirtualService.Spec.Provider.(type) {
	case *appmesh_types.VirtualServiceProviderMemberVirtualNode:
		return "virtualNode", nil
	case *appmesh_types.VirtualServiceProviderMemberVirtualRouter:
		return "virtualRouter", nil
	}
	return "", nil
}

func (a *mqlAwsAppmeshVirtualService) providerName() (string, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if resp.VirtualService == nil || resp.VirtualService.Spec == nil || resp.VirtualService.Spec.Provider == nil {
		return "", nil
	}
	switch p := resp.VirtualService.Spec.Provider.(type) {
	case *appmesh_types.VirtualServiceProviderMemberVirtualNode:
		return convert.ToValue(p.Value.VirtualNodeName), nil
	case *appmesh_types.VirtualServiceProviderMemberVirtualRouter:
		return convert.ToValue(p.Value.VirtualRouterName), nil
	}
	return "", nil
}

func (a *mqlAwsAppmeshVirtualNode) id() (string, error) {
	return a.Arn.Data, nil
}

type mqlAwsAppmeshVirtualNodeInternal struct {
	fetched  bool
	lock     sync.Mutex
	descResp *appmesh.DescribeVirtualNodeOutput
	region   string
	meshName string
}

func (a *mqlAwsAppmeshVirtualNode) fetchDetail() (*appmesh.DescribeVirtualNodeOutput, error) {
	if a.fetched {
		return a.descResp, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.descResp, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.AppMesh(a.region)
	ctx := context.Background()

	nodeName := a.Name.Data
	meshName := a.meshName
	resp, err := svc.DescribeVirtualNode(ctx, &appmesh.DescribeVirtualNodeInput{
		MeshName:        &meshName,
		VirtualNodeName: &nodeName,
	})
	if err != nil {
		return nil, err
	}
	a.fetched = true
	a.descResp = resp
	return resp, nil
}

func (a *mqlAwsAppmeshVirtualNode) status() (string, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if resp.VirtualNode == nil || resp.VirtualNode.Status == nil {
		return "", nil
	}
	return string(resp.VirtualNode.Status.Status), nil
}

func (a *mqlAwsAppmeshVirtualNode) backends() (int64, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return 0, err
	}
	if resp.VirtualNode == nil || resp.VirtualNode.Spec == nil {
		return 0, nil
	}
	return int64(len(resp.VirtualNode.Spec.Backends)), nil
}

func (a *mqlAwsAppmeshVirtualNode) backendServices() ([]any, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if resp.VirtualNode == nil || resp.VirtualNode.Spec == nil {
		return []any{}, nil
	}
	res, err := convert.JsonToDictSlice(resp.VirtualNode.Spec.Backends)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (a *mqlAwsAppmeshVirtualNode) serviceDiscovery() (map[string]any, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if resp.VirtualNode == nil || resp.VirtualNode.Spec == nil || resp.VirtualNode.Spec.ServiceDiscovery == nil {
		return map[string]any{}, nil
	}
	sd := resp.VirtualNode.Spec.ServiceDiscovery
	result := map[string]any{}
	switch s := sd.(type) {
	case *appmesh_types.ServiceDiscoveryMemberDns:
		result["type"] = "dns"
		result["hostname"] = convert.ToValue(s.Value.Hostname)
		result["responseType"] = string(s.Value.ResponseType)
		result["ipPreference"] = string(s.Value.IpPreference)
	case *appmesh_types.ServiceDiscoveryMemberAwsCloudMap:
		result["type"] = "awsCloudMap"
		result["namespaceName"] = convert.ToValue(s.Value.NamespaceName)
		result["serviceName"] = convert.ToValue(s.Value.ServiceName)
	}
	return result, nil
}

// ==================== Virtual Routers ====================

func (a *mqlAwsAppmeshMesh) virtualRouters() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	svc := conn.AppMesh(region)
	ctx := context.Background()

	meshName := a.Name.Data
	res := []any{}
	paginator := appmesh.NewListVirtualRoutersPaginator(svc, &appmesh.ListVirtualRoutersInput{
		MeshName: &meshName,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, vr := range page.VirtualRouters {
			mqlVr, err := CreateResource(a.MqlRuntime, "aws.appmesh.virtualRouter",
				map[string]*llx.RawData{
					"__id":     llx.StringDataPtr(vr.Arn),
					"arn":      llx.StringDataPtr(vr.Arn),
					"name":     llx.StringDataPtr(vr.VirtualRouterName),
					"meshName": llx.StringData(meshName),
					"region":   llx.StringData(region),
				})
			if err != nil {
				return nil, err
			}
			internal := mqlVr.(*mqlAwsAppmeshVirtualRouter)
			internal.region = region
			internal.meshName = meshName
			res = append(res, mqlVr)
		}
	}
	return res, nil
}

func (a *mqlAwsAppmeshVirtualRouter) id() (string, error) {
	return a.Arn.Data, nil
}

type mqlAwsAppmeshVirtualRouterInternal struct {
	fetched  bool
	lock     sync.Mutex
	descResp *appmesh.DescribeVirtualRouterOutput
	region   string
	meshName string
}

func (a *mqlAwsAppmeshVirtualRouter) fetchDetail() (*appmesh.DescribeVirtualRouterOutput, error) {
	if a.fetched {
		return a.descResp, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.descResp, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.AppMesh(a.region)
	ctx := context.Background()

	routerName := a.Name.Data
	meshName := a.meshName
	resp, err := svc.DescribeVirtualRouter(ctx, &appmesh.DescribeVirtualRouterInput{
		MeshName:          &meshName,
		VirtualRouterName: &routerName,
	})
	if err != nil {
		return nil, err
	}
	a.fetched = true
	a.descResp = resp
	return resp, nil
}

func (a *mqlAwsAppmeshVirtualRouter) status() (string, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if resp.VirtualRouter == nil || resp.VirtualRouter.Status == nil {
		return "", nil
	}
	return string(resp.VirtualRouter.Status.Status), nil
}

func (a *mqlAwsAppmeshVirtualRouter) listeners() ([]any, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if resp.VirtualRouter == nil || resp.VirtualRouter.Spec == nil {
		return []any{}, nil
	}
	listeners, err := convert.JsonToDictSlice(resp.VirtualRouter.Spec.Listeners)
	if err != nil {
		return nil, err
	}
	return listeners, nil
}

func (a *mqlAwsAppmeshVirtualRouter) routes() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.AppMesh(a.region)
	ctx := context.Background()

	meshName := a.meshName
	routerName := a.Name.Data
	res := []any{}
	paginator := appmesh.NewListRoutesPaginator(svc, &appmesh.ListRoutesInput{
		MeshName:          &meshName,
		VirtualRouterName: &routerName,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, r := range page.Routes {
			mqlRoute, err := CreateResource(a.MqlRuntime, "aws.appmesh.route",
				map[string]*llx.RawData{
					"__id":              llx.StringDataPtr(r.Arn),
					"arn":               llx.StringDataPtr(r.Arn),
					"name":              llx.StringDataPtr(r.RouteName),
					"meshName":          llx.StringData(meshName),
					"virtualRouterName": llx.StringData(routerName),
					"region":            llx.StringData(a.region),
				})
			if err != nil {
				return nil, err
			}
			internal := mqlRoute.(*mqlAwsAppmeshRoute)
			internal.region = a.region
			internal.meshName = meshName
			internal.virtualRouterName = routerName
			res = append(res, mqlRoute)
		}
	}
	return res, nil
}

func (a *mqlAwsAppmeshVirtualRouter) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.AppMesh(a.region)
	ctx := context.Background()

	arn := a.Arn.Data
	tags := make(map[string]any)
	paginator := appmesh.NewListTagsForResourcePaginator(svc, &appmesh.ListTagsForResourceInput{
		ResourceArn: &arn,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, tag := range page.Tags {
			tags[convert.ToValue(tag.Key)] = convert.ToValue(tag.Value)
		}
	}
	return tags, nil
}

// ==================== Routes ====================

func (a *mqlAwsAppmeshRoute) id() (string, error) {
	return a.Arn.Data, nil
}

type mqlAwsAppmeshRouteInternal struct {
	fetched           bool
	lock              sync.Mutex
	descResp          *appmesh.DescribeRouteOutput
	region            string
	meshName          string
	virtualRouterName string
}

func (a *mqlAwsAppmeshRoute) fetchDetail() (*appmesh.DescribeRouteOutput, error) {
	if a.fetched {
		return a.descResp, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.descResp, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.AppMesh(a.region)
	ctx := context.Background()

	routeName := a.Name.Data
	meshName := a.meshName
	routerName := a.virtualRouterName
	resp, err := svc.DescribeRoute(ctx, &appmesh.DescribeRouteInput{
		MeshName:          &meshName,
		VirtualRouterName: &routerName,
		RouteName:         &routeName,
	})
	if err != nil {
		return nil, err
	}
	a.fetched = true
	a.descResp = resp
	return resp, nil
}

func (a *mqlAwsAppmeshRoute) status() (string, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if resp.Route == nil || resp.Route.Status == nil {
		return "", nil
	}
	return string(resp.Route.Status.Status), nil
}

func (a *mqlAwsAppmeshRoute) spec() (map[string]any, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if resp.Route == nil || resp.Route.Spec == nil {
		return map[string]any{}, nil
	}
	return convert.JsonToDict(resp.Route.Spec)
}

func (a *mqlAwsAppmeshRoute) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.AppMesh(a.region)
	ctx := context.Background()

	arn := a.Arn.Data
	tags := make(map[string]any)
	paginator := appmesh.NewListTagsForResourcePaginator(svc, &appmesh.ListTagsForResourceInput{
		ResourceArn: &arn,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, tag := range page.Tags {
			tags[convert.ToValue(tag.Key)] = convert.ToValue(tag.Value)
		}
	}
	return tags, nil
}
